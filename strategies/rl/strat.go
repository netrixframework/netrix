package rl

import (
	"time"
)

type RLStrategyConfig struct {
	interpreter       Interpreter
	policy            Policy
	agentTickDuration time.Duration
}

// type RLStrategy struct {
// 	*types.BaseService
// 	actions *types.Channel[*strategies.Action]

// 	interpreter Interpreter
// 	policy      Policy
// 	trace       *Trace

// 	pendingMessages *types.Map[types.MessageID, *types.Message]
// 	pendingActions  *types.Map[types.MessageID, chan struct{}]

// 	agentTicker *time.Ticker
// 	config      *RLStrategyConfig

// 	stepCount int
// 	stepCountLock *sync.Mutex
// }

// func NewRLStrategy(config *RLStrategyConfig) *RLStrategy {
// 	return &RLStrategy{
// 		BaseService: types.NewBaseService("RLStrategy", nil),
// 		actions:     types.NewChannel[*strategies.Action](),
// 		interpreter: config.interpreter,
// 		policy:      config.policy,
// 		trace:       NewTrace(),

// 		pendingMessages: types.NewMap[types.MessageID, *types.Message](),
// 		pendingActions:  types.NewMap[types.MessageID, chan struct{}](),
// 		agentTicker:     time.NewTicker(config.agentTickDuration),
// 		config:          config,

// 		stepCount: 0,
// 		stepCountLock: new(sync.Mutex),
// 	}
// }

// var _ strategies.Strategy = &RLStrategy{}

// func (r *RLStrategy) ActionsCh() *types.Channel[*strategies.Action] {
// 	return r.actions
// }

// func (r *RLStrategy) Step(e *types.Event, ctx *strategies.Context) {
// 	if e.IsMessageSend() {
// 		if message, ok := ctx.GetMessage(e); ok {
// 			r.pendingMessages.Add(message.ID, message)
// 		}
// 	} else if e.IsMessageReceive() {
// 		if message, ok := ctx.GetMessage(e); ok {
// 			r.pendingMessages.Remove(message.ID)
// 			if ch, ok := r.pendingActions.Get(message.ID); ok {
// 				close(ch)
// 				r.pendingActions.Remove(message.ID)
// 			}
// 		}
// 	}
// 	r.interpreter.Update(e, ctx)
// }

// func (r *RLStrategy) EndCurIteration(ctx *strategies.Context) {

// }

// func (r *RLStrategy) NextIteration(ctx *strategies.Context) {

// }

// func (r *RLStrategy) Finalize(ctx *strategies.Context) {

// }

// func (r *RLStrategy) Start() error {
// 	r.BaseService.StartRunning()
// 	return nil
// }

// func (r *RLStrategy) Stop() error {
// 	r.BaseService.StopRunning()
// 	return nil
// }

// func (r *RLStrategy) actionLoop() {
// 	for {
// 		select {
// 		case <-r.agentTicker.C:
// 		case <-r.QuitCh():
// 			return
// 		}

// 		state := r.wrapState(r.interpreter.CurState())
// 		actions := state.Actions()
// 		r.stepCountLock.Lock()
// 		step := r.stepCount
// 		r.stepCountLock.Unlock()
// 		if len(actions) != 0 {
// 			action, ok := r.policy.NextAction(step, state, actions)
// 			if ok {
// 				r.doAction(action)

// 			}
// 		}
// 	}
// }

// func (r *RLStrategy) doAction(a *strategies.Action) {
// 	ch := make(chan struct{})
// 	r.pendingActions.Add(types.MessageID(a.Name), ch)

// 	r.actions.BlockingAdd(a)

// 	<-ch
// }

// func (r *RLStrategy) wrapState(s State) State {
// 	return &wrappedState{
// 		State:           s,
// 		pendingMessages: r.pendingMessages.IterValues(),
// 	}
// }
