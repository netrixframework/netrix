package sm

import (
	"fmt"

	"github.com/netrixframework/netrix/types"
)

// Condition a generic function that used to transition the [StateMachine] from one state to the next.
type Condition func(e *types.Event, c *Context) bool

// And to create boolean conditional expressions
func (c Condition) And(other Condition) Condition {
	return func(e *types.Event, ctx *Context) bool {
		return c(e, ctx) && other(e, ctx)
	}
}

// Or to create boolean conditional expressions
func (c Condition) Or(other Condition) Condition {
	return func(e *types.Event, ctx *Context) bool {
		return c(e, ctx) || other(e, ctx)
	}
}

// Not to create boolean conditional expressions
func (c Condition) Not() Condition {
	return func(e *types.Event, ctx *Context) bool {
		return !c(e, ctx)
	}
}

// IsEventOf returns true if the event if of the specified replica
func IsEventOf(replica types.ReplicaID) Condition {
	return func(e *types.Event, c *Context) bool {
		return e.Replica == replica
	}
}

// IsEventOfF accepts a function to specify the replica and returns true if the event is of that corresponding replica.
func IsEventOfF(replicaFunc func(*types.Event, *Context) (types.ReplicaID, bool)) Condition {
	return func(e *types.Event, c *Context) bool {
		rID, ok := replicaFunc(e, c)
		if !ok {
			return false
		}
		return e.Replica == rID
	}
}

// IsMessageSend condition returns true if the event is a message send event
func IsMessageSend() Condition {
	return func(e *types.Event, ctx *Context) bool {
		return e.IsMessageSend()
	}
}

// IsMessageReceive condition returns true if the event is a message receive event
func IsMessageReceive() Condition {
	return func(e *types.Event, ctx *Context) bool {
		return e.IsMessageReceive()
	}
}

// IsEventType condition returns true if the event is of type [types.GenericEventType] with T == t
func IsEventType(t string) Condition {
	return func(e *types.Event, c *Context) bool {
		eType, ok := e.Type.(*types.GenericEventType)
		if !ok {
			return false
		}
		return eType.T == t
	}
}

// IsMessageType condition returns true if the event is a message send or receive and the type of message is `t`
func IsMessageType(t string) Condition {
	return func(e *types.Event, c *Context) bool {
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		return message.Type == t
	}
}

// IsMessageTo condition returns true if the event is a message send or receive with message.To == to
func IsMessageTo(to types.ReplicaID) Condition {
	return func(e *types.Event, c *Context) bool {
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		return message.To == to
	}
}

// IsMessageFrom condition returns true if the event is a message send or receive with message.From == from
func IsMessageFrom(from types.ReplicaID) Condition {
	return func(e *types.Event, c *Context) bool {
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		return message.From == from
	}
}

// IsMessageFromF works the same as IsMessageFrom but the replica is fetched from the event and context
func IsMessageFromF(replicaF func(*types.Event, *Context) (types.ReplicaID, bool)) Condition {
	return func(e *types.Event, c *Context) bool {
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		replica, ok := replicaF(e, c)
		return ok && message.From == replica
	}
}

// IsMessageToF works the same as IsMessageTo but the replica is fetched from the event and context
func IsMessageToF(replicaF func(*types.Event, *Context) (types.ReplicaID, bool)) Condition {
	return func(e *types.Event, c *Context) bool {
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		replica, ok := replicaF(e, c)
		return ok && message.To == replica
	}
}

// IsMessage between condition is true when the underlying event is a message between the two specified processes.
func IsMessageBetween(one, two types.ReplicaID) Condition {
	return func(e *types.Event, c *Context) bool {
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		return (message.From == one && message.To == two) ||
			(message.To == one && message.From == two)
	}
}

// LtF condition that returns true if the counter value is less than the specified value.
// The input is a function that obtains the value dynamically based on the event and context.
func (c *CountWrapper) LtF(valF func(*types.Event, *Context) (int, bool)) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		v, ok := valF(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() < v
	}
}

// GtF condition that returns true if the counter value is greater than the specified value.
// The input is a function that obtains the value dynamically based on the event and context.
func (c *CountWrapper) GtF(val func(*types.Event, *Context) (int, bool)) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		v, ok := val(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() > v
	}
}

// EqF condition that returns true if the counter value is equal to the specified value.
// The input is a function that obtains the value dynamically based on the event and context.
func (c *CountWrapper) EqF(val func(*types.Event, *Context) (int, bool)) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		v, ok := val(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() == v
	}
}

// LeqF condition that returns true if the counter value is less than or equal to the specified value.
// The input is a function that obtains the value dynamically based on the event and context.
func (c *CountWrapper) LeqF(val func(*types.Event, *Context) (int, bool)) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		v, ok := val(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() <= v
	}
}

// GeqF condition that returns true if the counter value is greater than or equal to the specified value.
// The input is a function that obtains the value dynamically based on the event and context.
func (c *CountWrapper) GeqF(val func(*types.Event, *Context) (int, bool)) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		v, ok := val(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() >= v
	}
}

// Lt condition that returns true if the counter value is less than the specified value.
func (c *CountWrapper) Lt(val int) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() < val
	}
}

// Gt condition that returns true if the counter value is greater than the specified value.
func (c *CountWrapper) Gt(val int) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() > val
	}
}

// Eq condition that returns true if the counter value is equal to the specified value.
func (c *CountWrapper) Eq(val int) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() == val
	}
}

// Leq condition that returns true if the counter value is less than or equal to the specified value.
func (c *CountWrapper) Leq(val int) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() <= val
	}
}

// Geq condition that returns true if the counter value is greater than or equal to the specified value.
func (c *CountWrapper) Geq(val int) Condition {
	return func(e *types.Event, ctx *Context) bool {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return false
		}
		return counter.Value() >= val
	}
}

// Contains condition returns true if the event is a message send or receive and the message is apart of the message set.
func (s *SetWrapper) Contains() Condition {
	return func(e *types.Event, c *Context) bool {
		set, ok := s.SetFunc(e, c)
		if !ok {
			return false
		}
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		return set.Exists(message.ID)
	}
}

// Once is a meta condition that allows the inner condition to be true only once
func OnceCondition(name string, c Condition) Condition {
	return func(e *types.Event, ctx *Context) bool {
		key := fmt.Sprintf("%s_cond_once", name)
		if !ctx.Vars.Exists(key) {
			if c(e, ctx) {
				ctx.Vars.Set(key, true)
				return true
			}
		}
		return false
	}
}
