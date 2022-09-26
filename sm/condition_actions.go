package sm

import "github.com/netrixframework/netrix/types"

type Action func(*types.Event, *Context)

func ConditionWithAction(cond Condition, action Action) Condition {
	return func(e *types.Event, c *Context) bool {
		if cond(e, c) {
			action(e, c)
			return true
		}
		return false
	}
}

func (c *CountWrapper) Incr() Action {
	return func(e *types.Event, ctx *Context) {
		counter, ok := c.CounterFunc(e, ctx)
		if ok {
			counter.Incr()
		}
	}
}

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
