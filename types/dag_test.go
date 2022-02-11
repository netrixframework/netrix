package types

import (
	"testing"
	"time"
)

var (
	replicaOne = ReplicaID("one")
	replicaTwo = ReplicaID("two")
)

func TestDAGMarshal(t *testing.T) {
	replicaStore := NewReplicaStore(2)
	replicaStore.Add(&Replica{
		ID: replicaOne,
	})
	replicaStore.Add(&Replica{
		ID: replicaTwo,
	})

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
	dag := NewEventDag(replicaStore)
	dag.AddNode(events[0], []*Event{})
	dag.AddNode(events[1], []*Event{events[0]})
	dag.AddNode(events[2], []*Event{events[0]})

	_, err := dag.MarshalJSON()
	if err != nil {
		t.Errorf("error marshalling: %s", err)
	}
	// fmt.Printf("Marshalled Dag: %s\n", string(b))
}

func TestVectorClocks(t *testing.T) {
	replicaStore := NewReplicaStore(2)
	replicaStore.Add(&Replica{
		ID: replicaOne,
	})
	replicaStore.Add(&Replica{
		ID: replicaTwo,
	})

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
	dag := NewEventDag(replicaStore)
	dag.AddNode(events[0], []*Event{})
	dag.AddNode(events[1], []*Event{events[0]})
	dag.AddNode(events[2], []*Event{events[0]})

	en0, _ := dag.GetNode(events[0].ID)
	en1, _ := dag.GetNode(events[1].ID)
	en2, _ := dag.GetNode(events[2].ID)

	// fmt.Printf("vectorclocks: %#v, %#v, %#v\n", en0.ClockValue, en1.ClockValue, en2.ClockValue)

	if !en0.Lt(en1) {
		t.Errorf("vector clocks are not as expected: %#v, %#v", en0.ClockValue, en1.ClockValue)
	}
	if !en0.Lt(en2) {
		t.Errorf("vector clocks are not as expected: %#v, %#v", en0.ClockValue, en2.ClockValue)
	}
	if en1.Lt(en2) {
		t.Errorf("vector clocks are not as expected: %#v, %#v", en1.ClockValue, en2.ClockValue)
	}
}
