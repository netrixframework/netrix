package testlib

import (
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
)

// NewStateMachineHandler returns a HandlerFunc that encodes the execution logic of the StateMachine
// For every invocation of the handler, internal a state machine step is executed which may or may not transition.
// If the StateMachine transitions to FailureState, the handler aborts the test case
func NewStateMachineHandler(stateMachine *sm.StateMachine) FilterFunc {
	return func(e *types.Event, c *Context) ([]*types.Message, bool) {
		if stateMachine == nil {
			return []*types.Message{}, false
		}
		c.Logger.With(log.LogParams{
			"event_id":   e.ID,
			"event_type": e.TypeS,
		}).Debug("Async state machine handler step")
		stateMachine.Step(e, c.Context)
		newState := stateMachine.CurState()
		if newState.Is(sm.FailStateLabel) {
			c.Abort()
		}

		return []*types.Message{}, false
	}
}
