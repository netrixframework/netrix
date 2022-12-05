package sm

import "github.com/netrixframework/netrix/types"

// Action is a generic function that can be used to define side effects in the context
type Action func(*types.Event, *Context)

// ConditionWithAction creates a condition with the side effect given by Action
func ConditionWithAction(cond Condition, action Action) Condition {
	return func(e *types.Event, c *Context) bool {
		if cond(e, c) {
			action(e, c)
			return true
		}
		return false
	}
}

// Incr is an action that increments the counter
func (c *CountWrapper) Incr() Action {
	return func(e *types.Event, ctx *Context) {
		counter, ok := c.CounterFunc(e, ctx)
		if ok {
			counter.Incr()
		}
	}
}

// Add is an action that adds the corresponding message (if any) to the context
func (s *SetWrapper) Add() Action {
	return func(e *types.Event, ctx *Context) {
		message, ok := ctx.GetMessage(e)
		if !ok {
			return
		}
		set, ok := s.SetFunc(e, ctx)
		if ok {
			set.Add(message.ID, message)
		}
	}
}
