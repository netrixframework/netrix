package strategies

import (
	"sync"

	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/types"
)

type Context struct {
	Replicas *types.ReplicaStore
	Messages *types.Map[types.MessageID, *types.Message]
	EventDAG *types.EventDAG

	curIteration int
	lock         *sync.Mutex
}

func newContext(ctx *context.RootContext) *Context {
	return &Context{
		Replicas:     ctx.Replicas,
		Messages:     ctx.MessageStore,
		EventDAG:     types.NewEventDag(ctx.Replicas),
		curIteration: 0,
		lock:         new(sync.Mutex),
	}
}

func (c *Context) CurIteration() int {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.curIteration
}

func (c *Context) NextIteration() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.curIteration++
}

type Action struct {
	Name string
	Do   func(*Context, *dispatcher.Dispatcher) error
}

func DeliverMessage(m *types.Message) Action {
	return Action{
		Name: "DeliverMessage",
		Do: func(ctx *Context, d *dispatcher.Dispatcher) error {
			if !ctx.Messages.Exists(m.ID) {
				ctx.Messages.Add(m.ID, m)
			}
			return d.DispatchMessage(m)
		},
	}
}

var doNothingAction = "_nothing"

func DoNothing() Action {
	return Action{
		Name: doNothingAction,
		Do: func(ctx *Context, d *dispatcher.Dispatcher) error {
			return nil
		},
	}
}

type Strategy interface {
	types.Service
	Step(*types.Event, *Context) Action
	NextIteration()
}
