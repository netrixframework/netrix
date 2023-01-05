package rl

import (
	"sync"
	"time"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type RLStrategyConfig struct {
	interpreter       Interpreter
	policy            Policy
	agentTickDuration time.Duration
}

type RLStrategy struct {
	*types.BaseService
	config      *RLStrategyConfig
	interpreter Interpreter
	actions     *types.Channel[*strategies.Action]
	policy      Policy

	pendingMessages *types.Map[types.MessageID, *types.Message]
	pendingActions  *types.Map[types.MessageID, chan struct{}]
	agentTicker     *time.Ticker

	trace         *Trace
	stepCount     int
	stepCountLock *sync.Mutex
}

func NewRLStrategy(config *RLStrategyConfig) *RLStrategy {
	return &RLStrategy{
		BaseService: types.NewBaseService("RLExploration", nil),
		config:      config,
		interpreter: config.interpreter,
		policy:      config.policy,
		actions:     types.NewChannel[*strategies.Action](),

		pendingMessages: types.NewMap[types.MessageID, *types.Message](),
		pendingActions:  types.NewMap[types.MessageID, chan struct{}](),
		agentTicker:     time.NewTicker(config.agentTickDuration),
		trace:           NewTrace(),
		stepCount:       0,
		stepCountLock:   new(sync.Mutex),
	}
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
			if ch, ok := r.pendingActions.Get(message.ID); ok {
				close(ch)
				r.pendingActions.Remove(message.ID)
			}
		}
	}
	r.interpreter.Update(e, ctx)
}

func (r *RLStrategy) EndCurIteration(ctx *strategies.Context) {
}

func (r *RLStrategy) NextIteration(ctx *strategies.Context) {
	r.interpreter.Reset()
	r.policy.NextIteration(ctx.CurIteration(), r.trace)
	r.trace.Reset()
	r.stepCountLock.Lock()
	r.stepCount = 0
	r.stepCountLock.Unlock()
	r.pendingMessages.RemoveAll()
	r.pendingActions.RemoveAll()
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
			r.trace.Add(state, action)
		}
	}
}

func (r *RLStrategy) doAction(a *strategies.Action) {
	ch := make(chan struct{})
	r.pendingActions.Add(types.MessageID(a.Name), ch)
	r.actions.BlockingAdd(a)

	<-ch
}
