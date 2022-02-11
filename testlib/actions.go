package testlib

import (
	"github.com/netrixframework/netrix/types"
)

// Action is used to specify the consequence in the `If().Then()` handler
type Action func(*types.Event, *Context) []*types.Message

// IfThenHandler struct is used to wrap the attributes of the `If().Then()` handler
type IfThenHandler struct {
	cond    Condition
	actions []Action
}

// If creates a IfThenHandler with the specified condition
func If(cond Condition) *IfThenHandler {
	return &IfThenHandler{
		cond:    cond,
		actions: make([]Action, 0),
	}
}

// Then returns a HandlerFunc which encodes the `If().Then()` semantics.
// Accepts actions as arguments
func (i *IfThenHandler) Then(action Action, rest ...Action) FilterFunc {
	i.actions = append(i.actions, action)
	i.actions = append(i.actions, rest...)
	return func(e *types.Event, c *Context) ([]*types.Message, bool) {
		if i.cond(e, c) {
			result := make([]*types.Message, 0)
			for _, h := range i.actions {
				result = append(result, h(e, c)...)
			}
			return result, true
		}
		return []*types.Message{}, false
	}
}

// DeliverMessage returns the message if the event is a message send event
func DeliverMessage() Action {
	return func(e *types.Event, c *Context) []*types.Message {
		if !e.IsMessageSend() {
			return []*types.Message{}
		}
		messageID, _ := e.MessageID()
		message, ok := c.MessagePool.Get(messageID)
		if ok {
			return []*types.Message{message}
		}
		return []*types.Message{}
	}
}

// DropMessage returns an empty list of messages
func DropMessage() Action {
	return func(e *types.Event, c *Context) []*types.Message {
		return []*types.Message{}
	}
}

// Incr returns an action which increments the counter value
func (c *CountWrapper) Incr() Action {
	return func(e *types.Event, ctx *Context) []*types.Message {
		counter, ok := c.CounterFunc(e, ctx)
		if !ok {
			return []*types.Message{}
		}
		counter.Incr()
		return []*types.Message{}
	}
}

// Store returns an action. If the event is a message send or receive,
// the action adds the message to the message set
func (s *SetWrapper) Store() Action {
	return func(e *types.Event, c *Context) []*types.Message {
		set, ok := s.SetFunc(e, c)
		if !ok {
			return []*types.Message{}
		}
		message, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		set.Add(message)
		return []*types.Message{}
	}
}

// DeliverAll returns an action which inturn returns all the messages in the
// message set and removes the messages from the set.
func (s *SetWrapper) DeliverAll() Action {
	return func(e *types.Event, c *Context) []*types.Message {
		set, ok := s.SetFunc(e, c)
		if !ok {
			return []*types.Message{}
		}
		result := set.Iter()
		set.RemoveAll()
		return result
	}
}

// RecordMessageAs returns an action. If the event is a message send or receive,
// the message is recorded in context with the label as reference
func RecordMessageAs(label string) Action {
	return func(e *types.Event, c *Context) []*types.Message {
		message, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		c.Vars.Set(label, message)
		return []*types.Message{}
	}
}

type ReplicaFunc func(*types.Event, *Context) (types.ReplicaID, bool)

// StopReplica is the action to stop a replica. The replica details are fetched dynamically through the specified function
func StopReplica(replicaFunc ReplicaFunc) Action {
	return func(e *types.Event, c *Context) (messages []*types.Message) {
		replica, ok := replicaFunc(e, c)
		if !ok {
			return
		}
		c.dispatcher.StopReplica(replica)
		return
	}

}

// StartReplica is the action to start a replica. The replica details are fetched dynamically through the specified function
func StartReplica(replicaFunc ReplicaFunc) Action {
	return func(e *types.Event, c *Context) (messages []*types.Message) {
		replica, ok := replicaFunc(e, c)
		if !ok {
			return
		}
		c.dispatcher.StartReplica(replica)
		return
	}
}

// RestartReplica is the action to restart a replica. The replica details are fetched dynamically through the specified function
func RestartReplica(replicaFunc ReplicaFunc) Action {
	return func(e *types.Event, c *Context) (messages []*types.Message) {
		replica, ok := replicaFunc(e, c)
		if !ok {
			return
		}
		c.dispatcher.RestartReplica(replica)
		return
	}
}

// MessageFrom ReplicaFunc returns the message from replica, if the event is a message send/receive
func MessageFrom() ReplicaFunc {
	return func(e *types.Event, c *Context) (types.ReplicaID, bool) {
		m, ok := c.GetMessage(e)
		if !ok {
			return "", false
		}
		return m.From, true
	}
}

// MessageTo ReplicaFunc returns the message to replica, if the event is a message send/receive
func MessageTo() ReplicaFunc {
	return func(e *types.Event, c *Context) (types.ReplicaID, bool) {
		m, ok := c.GetMessage(e)
		if !ok {
			return "", false
		}
		return m.To, true
	}
}

// EventReplica ReplicaFunc returns the replica of the current event.
func EventReplica() ReplicaFunc {
	return func(e *types.Event, c *Context) (types.ReplicaID, bool) {
		return e.Replica, true
	}
}
