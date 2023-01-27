// Package strategies provides the framework to define testing strategies.
//
// Test strategy can incorporate any exploration logic and run for a specified number of iterations.
// The package provides some inbuilt strategies like PCTCP and unittest (running a unit test for many iterations)
package strategies

import (
	"sync"

	"github.com/netrixframework/netrix/apiserver"
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
)

// Context here references to the execution context of a strategy
type Context struct {
	*sm.Context
	// ActionSequence stores the sequence of actions performed in the current iteration
	ActionSequence *types.List[*Action]
	curIteration   int
	abortCh        chan struct{}
	once           *sync.Once
	lock           *sync.Mutex
}

func newContext(ctx *context.RootContext) *Context {
	return &Context{
		Context:        sm.NewContext(ctx, ctx.Logger),
		ActionSequence: types.NewEmptyList[*Action](),
		curIteration:   0,
		abortCh:        make(chan struct{}),
		once:           new(sync.Once),
		lock:           new(sync.Mutex),
	}
}

// CurIteration returns the current iteration number
func (c *Context) CurIteration() int {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.curIteration
}

func (c *Context) nextIteration() {
	c.ActionSequence.RemoveAll()
	c.EventDAG.Reset()
	c.Vars.Reset()

	c.lock.Lock()
	defer c.lock.Unlock()
	c.curIteration++
}

// Abort the current iteration
func (c *Context) Abort() {
	c.once.Do(func() {
		close(c.abortCh)
	})
}

// Action that the strategy can perform
type Action struct {
	// Unique name for the action
	Name string
	// Function that performs the action
	Do func(*Context, *apiserver.APIServer) error
}

// DeliverMessage action that delivered the given message using [apiserver.APIServer]
func DeliverMessage(m *types.Message) *Action {
	return &Action{
		Name: "DeliverMessage_" + string(m.ID),
		Do: func(ctx *Context, d *apiserver.APIServer) error {
			if !ctx.MessagePool.Exists(m.ID) {
				ctx.MessagePool.Add(m.ID, m)
			}
			return d.SendMessage(m)
		},
	}
}

// DeliverMany action delivered all the specified messages
func DeliverMany(messages []*types.Message) *Action {
	return &Action{
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

// DoNothing is a no-op action
func DoNothing() *Action {
	return &Action{
		Name: doNothingAction,
		Do: func(ctx *Context, d *apiserver.APIServer) error {
			return nil
		},
	}
}

// ActionSequence actions executes actions in a sequence
func ActionSequence(actions ...Action) *Action {
	return &Action{
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

// Strategy interface to define an exploration strategy
// An exploration strategy should be encoded in this strategy
type Strategy interface {
	types.Service
	// ActionsCh should return a channel that is used to
	// communicate the actions that needs to be performed
	ActionsCh() *types.Channel[*Action]
	// Step is called for each event, actions to be performed can be decided within the step call
	Step(*types.Event, *Context)
	// EndCurIteration is called at the end of every iteration
	EndCurIteration(*Context)
	// NextIteration is called at the beginning of every iteration
	NextIteration(*Context)
	// Finalize is called after executing all the iteration.
	// This function can be used to record/log telemetry or other stats
	Finalize(*Context)
}
