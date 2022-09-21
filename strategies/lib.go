package strategies

import (
	"sync"

	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
)

type Context struct {
	*sm.Context
	curIteration int
	abortCh      chan struct{}
	once         *sync.Once
	lock         *sync.Mutex
}

func newContext(ctx *context.RootContext) *Context {
	return &Context{
		Context:      sm.NewContext(ctx, ctx.Logger),
		curIteration: 0,
		abortCh:      make(chan struct{}),
		once:         new(sync.Once),
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
	c.EventDAG.Reset()
}

func (c *Context) AbortCh() <-chan struct{} {
	return c.abortCh
}

func (c *Context) Abort() {
	c.once.Do(func() {
		close(c.abortCh)
	})
}

type Action struct {
	Name string
	Do   func(*Context, *dispatcher.Dispatcher) error
}

func DeliverMessage(m *types.Message) Action {
	return Action{
		Name: "DeliverMessage",
		Do: func(ctx *Context, d *dispatcher.Dispatcher) error {
			if !ctx.MessagePool.Exists(m.ID) {
				ctx.MessagePool.Add(m.ID, m)
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

func ActionSequence(actions ...Action) Action {
	return Action{
		Name: "Sequence",
		Do: func(ctx *Context, d *dispatcher.Dispatcher) error {
			for _, action := range actions {
				ctx.Logger.With(log.LogParams{"action": action.Name}).Debug("Calling action")
				if err := action.Do(ctx, d); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

type Strategy interface {
	types.Service
	Step(*types.Event, *Context) Action
	EndCurIteration(*Context)
	NextIteration(*Context)
	Finalize(*Context)
}
