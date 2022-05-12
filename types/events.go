package types

type EventID uint64

// EventType abstract type for representing different types of events
type EventType interface {
	// Clone copies the event type
	Clone() EventType
	// Type is a unique key for that event type
	Type() string
	// String should return a string representation of the event type
	String() string
}

// Event is a generic event that occurs at a replica
type Event struct {
	// Replica at which the event occurs
	Replica ReplicaID `json:"replica"`
	// Type of the event
	Type EventType `json:"-"`
	// TypeS is the string representation of the event
	TypeS string `json:"type"`
	// ID unique identifier assigned for every new event
	ID EventID `json:"id"`
	// Timestamp of the event
	Timestamp int64 `json:"timestamp"`
}

func NewEvent(replica ReplicaID, t EventType, ts string, id EventID, time int64) *Event {
	return &Event{
		Replica:   replica,
		Type:      t,
		TypeS:     ts,
		ID:        id,
		Timestamp: time,
	}
}

func (e *Event) MessageID() (MessageID, bool) {
	switch eType := e.Type.(type) {
	case *MessageReceiveEventType:
		return eType.MessageID, true
	case *MessageSendEventType:
		return eType.MessageID, true
	}
	return "", false
}

func (e *Event) IsMessageSend() bool {
	switch e.Type.(type) {
	case *MessageSendEventType:
		return true
	}
	return false
}

func (e *Event) IsMessageReceive() bool {
	switch e.Type.(type) {
	case *MessageReceiveEventType:
		return true
	}
	return false
}

func (e *Event) Timeout() (*ReplicaTimeout, bool) {
	switch eType := e.Type.(type) {
	case *TimeoutStartEventType:
		return eType.Timeout, true
	case *TimeoutEndEventType:
		return eType.Timeout, true
	}
	return nil, false
}

func (e *Event) IsTimeoutStart() bool {
	switch e.Type.(type) {
	case *TimeoutStartEventType:
		return true
	}
	return false
}

func (e *Event) IsTimeoutEnd() bool {
	switch e.Type.(type) {
	case *TimeoutEndEventType:
		return true
	}
	return false
}

// Clone implements Clonable
func (e *Event) Clone() Clonable {
	return &Event{
		Replica:   e.Replica,
		Type:      e.Type.Clone(),
		TypeS:     e.TypeS,
		ID:        e.ID,
		Timestamp: e.Timestamp,
	}
}
