package pct

import "github.com/netrixframework/netrix/types"

type Event struct {
	messageID types.MessageID
	replica   types.ReplicaID
}

func NewEvent(message *types.Message) *Event {
	return &Event{
		messageID: message.ID,
		replica:   message.To,
	}
}

type EventPartialOrder struct {
}
