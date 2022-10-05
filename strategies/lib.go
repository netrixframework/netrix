package strategies

import (
	"sync"

	"github.com/netrixframework/netrix/apiserver"
	"github.com/netrixframework/netrix/context"
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
	c.Vars.Reset()
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
	Do   func(*Context, *apiserver.APIServer) error
}

func DeliverMessage(m *types.Message) Action {
	return Action{
		Name: "DeliverMessage",
		Do: func(ctx *Context, d *apiserver.APIServer) error {
			if !ctx.MessagePool.Exists(m.ID) {
				ctx.MessagePool.Add(m.ID, m)
			}
			return d.SendMessage(m)
		},
	}
}

func DeliverMany(messages []*types.Message) Action {
	return Action{
		Name: "DeliverManyMessages",
		Do: func(ctx *Context, a *apiserver.APIServer) error {
			for _, message := range messages {
				if !ctx.MessagePool.Exists(message.ID) {
					ctx.MessagePool.Add(message.ID, message)
				}
				if err := a.SendMessage(message); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

var doNothingAction = "_nothing"

func DoNothing() Action {
	return Action{
		Name: doNothingAction,
		Do: func(ctx *Context, d *apiserver.APIServer) error {
			return nil
		},
	}
}

func ActionSequence(actions ...Action) Action {
	return Action{
		Name: "Sequence",
		Do: func(ctx *Context, d *apiserver.APIServer) error {
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
