package sm

import (
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

// Context used to step the state machine.
// The Context is passed to the [Condition].
type Context struct {
	MessagePool  *types.Map[types.MessageID, *types.Message]
	ReplicaStore *types.ReplicaStore
	EventDAG     *types.EventDAG
	Vars         *types.VarSet
	Logger       *log.Logger
}

// NewContext creates a new context from the global context object.
func NewContext(ctx *context.RootContext, logger *log.Logger) *Context {
	return &Context{
		MessagePool:  ctx.MessageStore,
		ReplicaStore: ctx.Replicas,
		EventDAG:     types.NewEventDag(ctx.Replicas),
		Vars:         types.NewVarSet(),
		Logger:       logger,
	}
}

// GetMessage helper function used to fetch the message for the corresponding event
// This returns the message only when the event is a send or a receive.
func (c *Context) GetMessage(e *types.Event) (*types.Message, bool) {
	if !e.IsMessageSend() && !e.IsMessageReceive() {
		return nil, false
	}
	mID, _ := e.MessageID()
	return c.MessagePool.Get(mID)
}
