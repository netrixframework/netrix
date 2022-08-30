package pct

import "github.com/netrixframework/netrix/types"

type Event struct {
	messageID types.MessageID
	from      types.ReplicaID
	to        types.ReplicaID
	label     int
}

func (e *Event) Lt(other *Event) bool {
	// use vector clocks for determining partial orders
	// Delegate this to the protocol.
	return false
}

func (e *Event) HasLabel() bool {
	return e.label != -1
}

func (e *Event) Label() int {
	return e.label
}

func NewEvent(message *types.Message) *Event {
	return &Event{
		messageID: message.ID,
		to:        message.To,
		from:      message.From,
		label:     -1,
	}
}

type EventOrder struct {
}
