package timeout

import (
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/types"
)

type TimeoutStrategy struct {
	ctx             *context.RootContext
	pendingMessages map[string]*types.Message
	eventDAG        *types.EventDAG
	*types.BaseService
}

func NewTimeoutStrategy(ctx *context.RootContext) *TimeoutStrategy {
	return &TimeoutStrategy{
		ctx:             ctx,
		pendingMessages: make(map[string]*types.Message),
		BaseService:     types.NewBaseService("TimeoutStrategy", ctx.Logger),
	}
}

func (t *TimeoutStrategy) Step(e *types.Event, in []*types.Message) *types.Message {
	for _, m := range in {
		t.pendingMessages[m.ID] = m
	}

	return nil
}
