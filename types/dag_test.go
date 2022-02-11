package types

import (
	"fmt"
	"testing"
	"time"
)

var (
	replicaOne = ReplicaID("one")
	replicaTwo = ReplicaID("two")
)

func TestDAGMarshal(t *testing.T) {
	message_send := NewMessageSendEventType("message_one")
	message_receive := NewMessageReceiveEventType("message_one")
	commit := NewGenericEventType(map[string]string{"type": "commit"}, "COMMIT")
	events := []*Event{
		{
			Replica:   replicaOne,
			Type:      message_send,
			TypeS:     message_send.String(),
			ID:        1,
			Timestamp: time.Now().Unix(),
		},
		{
			Replica:   replicaTwo,
			Type:      message_receive,
			TypeS:     message_receive.String(),
			ID:        2,
			Timestamp: time.Now().Unix(),
		},
		{
			Replica:   replicaOne,
			Type:      commit,
			TypeS:     commit.String(),
			ID:        3,
			Timestamp: time.Now().Unix(),
		},
	}
	dag := NewEventDag()
	dag.AddNode(events[0], []*Event{})
	dag.AddNode(events[1], []*Event{events[0]})
	dag.AddNode(events[2], []*Event{events[0]})

	b, err := dag.MarshalJSON()
	if err != nil {
		t.Errorf("error marshalling: %s", err)
	}
	fmt.Printf("Marshalled Dag: %s\n", string(b))
}
