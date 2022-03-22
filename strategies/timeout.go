package strategies

import (
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/types"
)

type TimeoutStrategy struct {
	ctx           *context.RootContext
	pendingEvents map[types.EventID]*types.Event
	*types.BaseService
}

func NewTimeoutStrategy(ctx *context.RootContext) *TimeoutStrategy {
	return &TimeoutStrategy{
		ctx:           ctx,
		pendingEvents: make(map[types.EventID]*types.Event),
		BaseService:   types.NewBaseService("TimeoutStrategy", ctx.Logger),
	}
}

func (t *TimeoutStrategy) Step(e *types.Event, in []*types.Message) []*types.Message {
	return []*types.Message{}
}
