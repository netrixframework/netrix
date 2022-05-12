package timeout

import (
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type TimeoutStrategy struct {
	pendingMessages map[string]*types.Message
	*types.BaseService
}

func NewTimeoutStrategy() *TimeoutStrategy {
	return &TimeoutStrategy{
		pendingMessages: make(map[string]*types.Message),
		BaseService:     types.NewBaseService("TimeoutStrategy", nil),
	}
}

func (t *TimeoutStrategy) Step(e *types.Event, ctx *strategies.Context) strategies.Action {
	// TODO: fill this up

	return strategies.DoNothing()
}
