package rl

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type RLStrategyConfig struct {
	Interpreter       Interpreter
	Policy            Policy
	AgentTickDuration time.Duration
	MetricsPath       string
	// TimeoutEpsilon configures how often we want to transition to a state with
	// timeouts enabled (between 0 and 1). 0 indicates never and 1 indicates always.
	TimeoutEpsilon float64
}

type RLStrategy struct {
	*types.BaseService
	config      *RLStrategyConfig
	interpreter Interpreter
	actions     *types.Channel[*strategies.Action]
	policy      Policy
	replicas    map[types.ReplicaID]bool

	pendingMessages     *types.Map[types.MessageID, *types.Message]
	pendingActions      map[types.MessageID]chan struct{}
	pendingActionsLock  *sync.Mutex
	agentTicker         *time.Ticker
	metrics             *Metrics
	stepCount           int
	stepCountLock       *sync.Mutex
	agentEnabled        bool
	agentEnabledLock    *sync.Mutex
	timeoutRand         *rand.Rand
	curTimeoutState     bool
	curTimeoutStateLock *sync.Mutex
}

func NewRLStrategy(config *RLStrategyConfig) (*RLStrategy, error) {
	metrics, err := NewMetrics(config.MetricsPath)
	if err != nil {
		return nil, err
	}
	if config.TimeoutEpsilon < 0 || config.TimeoutEpsilon > 1 {
		return nil, fmt.Errorf("invalid time epsilon params")
	}
	return &RLStrategy{
		BaseService: types.NewBaseService("RLExploration", nil),
		config:      config,
		interpreter: config.Interpreter,
		policy:      config.Policy,
		actions:     types.NewChannel[*strategies.Action](),
		replicas:    make(map[types.ReplicaID]bool),

		pendingMessages:     types.NewMap[types.MessageID, *types.Message](),
		pendingActions:      make(map[types.MessageID]chan struct{}),
		pendingActionsLock:  new(sync.Mutex),
		agentTicker:         time.NewTicker(config.AgentTickDuration),
		metrics:             metrics,
		stepCount:           0,
		stepCountLock:       new(sync.Mutex),
		agentEnabled:        false,
		agentEnabledLock:    new(sync.Mutex),
		curTimeoutState:     false,
		curTimeoutStateLock: new(sync.Mutex),
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
	r.agentEnabledLock.Lock()
	r.agentEnabled = false
	r.agentEnabledLock.Unlock()
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
	r.agentTicker.Reset(r.config.AgentTickDuration)

	r.agentEnabledLock.Lock()
	r.agentEnabled = true
	r.agentEnabledLock.Unlock()
}

func (r *RLStrategy) Finalize(ctx *strategies.Context) {
}

func (r *RLStrategy) isAgentEnabled() bool {
	r.agentEnabledLock.Lock()
	defer r.agentEnabledLock.Unlock()
	return r.agentEnabled
}

func (r *RLStrategy) Start() error {
	r.BaseService.StartRunning()
	r.Logger.With(log.LogParams{
		"timeout_epsilon": r.config.TimeoutEpsilon,
		"agent_tick_dur":  r.config.AgentTickDuration.String(),
	}).Info("Starting RL strategy!")
	go r.agentLoop()
	return nil
}

func (r *RLStrategy) Stop() error {
	r.BaseService.StopRunning()
	r.agentTicker.Stop()
	return nil
}

// TODO: Abstract away the agent
func (r *RLStrategy) agentLoop() {
	for {
		if !r.isAgentEnabled() {
			continue
		}
		state := r.wrapState(r.interpreter.CurState(), false)
		possibleActions := state.Actions()
		if len(possibleActions) == 0 {
			continue
		}
		haveMessageAction := false
		for _, a := range possibleActions {
			if a.Type == DeliverMessage {
				haveMessageAction = true
				break
			}
		}
		if haveMessageAction {
			r.Logger.With(log.LogParams{
				"state": fmt.Sprintf("%v", state.InterpreterState),
				"hash":  state.InterpreterState.Hash(),
			}).Debug("Stepping RL Agent, have message action")
		}
		r.stepCountLock.Lock()
		step := r.stepCount
		r.stepCountLock.Unlock()
		action, ok := r.policy.NextAction(step, state, possibleActions)
		if ok {
			r.doAction(action)
			movedOn := false
			r.stepCountLock.Lock()
			if r.stepCount == step {
				r.stepCount += 1
			} else {
				movedOn = true
			}
			r.stepCountLock.Unlock()
			if movedOn {
				// The Agent has moved on to the next episode/iteration
				// We do not update the policy or metrics

				// Relying on the assumption that number of steps is >= 1
				// in each episode/iteration
				continue
			}
			select {
			case <-r.agentTicker.C:
			case <-r.QuitCh():
				return
			}
			nextState := r.wrapState(r.interpreter.CurState(), true)
			r.policy.Update(step, state, action, nextState)
			r.metrics.Update(step, state, action)
		}
	}
}

func (r *RLStrategy) doAction(a *Action) {
	switch a.Type {
	case TimeoutReplica:
		for _, m := range r.pendingMessages.IterValues() {
			if m.To == a.replica { //|| m.From == a.replica
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
