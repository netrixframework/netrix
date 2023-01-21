package rl

import (
	"sync"
	"time"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type RLStrategyConfig struct {
	Interpreter       Interpreter
	Policy            Policy
	AgentTickDuration time.Duration
	MetricsPath       string
	AllowTimeouts     bool
}

type RLStrategy struct {
	*types.BaseService
	config      *RLStrategyConfig
	interpreter Interpreter
	actions     *types.Channel[*strategies.Action]
	policy      Policy
	replicas    map[types.ReplicaID]bool

	pendingMessages    *types.Map[types.MessageID, *types.Message]
	pendingActions     map[types.MessageID]chan struct{}
	pendingActionsLock *sync.Mutex
	agentTicker        *time.Ticker
	metrics            *Metrics
	stepCount          int
	stepCountLock      *sync.Mutex
}

func NewRLStrategy(config *RLStrategyConfig) (*RLStrategy, error) {
	metrics, err := NewMetrics(config.MetricsPath)
	if err != nil {
		return nil, err
	}
	return &RLStrategy{
		BaseService: types.NewBaseService("RLExploration", nil),
		config:      config,
		interpreter: config.Interpreter,
		policy:      config.Policy,
		actions:     types.NewChannel[*strategies.Action](),
		replicas:    make(map[types.ReplicaID]bool),

		pendingMessages:    types.NewMap[types.MessageID, *types.Message](),
		pendingActions:     make(map[types.MessageID]chan struct{}),
		pendingActionsLock: new(sync.Mutex),
		agentTicker:        time.NewTicker(config.AgentTickDuration),
		metrics:            metrics,
		stepCount:          0,
		stepCountLock:      new(sync.Mutex),
	}, nil
}

var _ strategies.Strategy = &RLStrategy{}

func (r *RLStrategy) ActionsCh() *types.Channel[*strategies.Action] {
	return r.actions
}

func (r *RLStrategy) Step(e *types.Event, ctx *strategies.Context) {

	if e.IsMessageSend() {
		if message, ok := ctx.GetMessage(e); ok {
			r.pendingMessages.Add(message.ID, message)
		}
	} else if e.IsMessageReceive() {
		if message, ok := ctx.GetMessage(e); ok {
			r.pendingMessages.Remove(message.ID)
			r.pendingActionsLock.Lock()
			if ch, ok := r.pendingActions[message.ID]; ok {
				close(ch)
				delete(r.pendingActions, message.ID)
			}
			r.pendingActionsLock.Unlock()
		}
	}
	r.interpreter.Update(e, ctx)
}

func (r *RLStrategy) EndCurIteration(ctx *strategies.Context) {
}

func (r *RLStrategy) NextIteration(ctx *strategies.Context) {
	r.interpreter.Reset()
	r.policy.NextIteration(ctx.CurIteration(), r.metrics.Trace)
	r.metrics.NextIteration()
	r.stepCountLock.Lock()
	r.stepCount = 0
	r.stepCountLock.Unlock()
	r.pendingMessages.RemoveAll()
	r.pendingActionsLock.Lock()
	for _, ch := range r.pendingActions {
		close(ch)
	}
	r.pendingActions = make(map[types.MessageID]chan struct{})
	r.pendingActionsLock.Unlock()
}

func (r *RLStrategy) Finalize(ctx *strategies.Context) {
}

func (r *RLStrategy) Start() error {
	r.BaseService.StartRunning()
	go r.agentLoop()
	return nil
}

func (r *RLStrategy) Stop() error {
	r.BaseService.StopRunning()
	return nil
}

// TODO: Abstract away the agent
func (r *RLStrategy) agentLoop() {
	for {
		select {
		case <-r.agentTicker.C:
		case <-r.QuitCh():
			return
		}
		state := r.wrapState(r.interpreter.CurState())
		possibleActions := state.Actions()
		if len(possibleActions) == 0 {
			continue
		}
		r.stepCountLock.Lock()
		step := r.stepCount
		r.stepCountLock.Unlock()
		action, ok := r.policy.NextAction(step, state, possibleActions)
		if ok {
			r.doAction(action)
			r.stepCountLock.Lock()
			r.stepCount += 1
			r.stepCountLock.Unlock()
			// TODO: move the select (waiting) here
			nextState := r.wrapState(r.interpreter.CurState())
			r.policy.Update(step, state, action, nextState)
			r.metrics.Update(step, state, action)
		}
	}
}

func (r *RLStrategy) doAction(a *Action) {
	switch a.Type {
	case TimeoutReplica:
		for _, m := range r.pendingMessages.IterValues() {
			if m.To == a.replica || m.From == a.replica {
				r.pendingMessages.Remove(m.ID)
			}
		}
	case DeliverMessage:
		ch := make(chan struct{})
		r.pendingActionsLock.Lock()
		if existingCh, ok := r.pendingActions[a.message.ID]; ok {
			close(ch)
			ch = existingCh
		} else {
			r.pendingActions[a.message.ID] = ch
		}
		r.pendingActionsLock.Unlock()
		r.actions.BlockingAdd(
			strategies.DeliverMessage(a.message),
		)
		<-ch
	}
}
