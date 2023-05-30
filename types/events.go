// Package type defines the common data structures used internally and for defining tests/strategies in Netrix.
package types

// EventID is a unique identifier for each event
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
	// Additional params of the event
	Params map[string]string `json:"params"`
}

// NewEvent creates an [Event] with the specified values
func NewEvent(replica ReplicaID, t EventType, ts string, id EventID, time int64) *Event {
	return &Event{
		Replica:   replica,
		Type:      t,
		TypeS:     ts,
		ID:        id,
		Timestamp: time,
		Params:    make(map[string]string),
	}
}

func NewEventWithParams(replica ReplicaID, t EventType, ts string, id EventID, time int64, params map[string]string) *Event {
	e := &Event{
		Replica:   replica,
		Type:      t,
		TypeS:     ts,
		ID:        id,
		Timestamp: time,
		Params:    make(map[string]string),
	}
	for k, v := range params {
		e.Params[k] = v
	}
	return e
}

// MessageID returns the message ID of the corresponding message when the event type is either
// a message send or a message receive. The second return value is false otherwise
func (e *Event) MessageID() (MessageID, bool) {
	switch eType := e.Type.(type) {
	case *MessageReceiveEventType:
		return eType.MessageID, true
	case *MessageSendEventType:
		return eType.MessageID, true
	}
	return "", false
}

// IsMessageSend is true when the event is of type message send
func (e *Event) IsMessageSend() bool {
	switch e.Type.(type) {
	case *MessageSendEventType:
		return true
	}
	return false
}

// IsMessageReceive is true when the event is of type message receive
func (e *Event) IsMessageReceive() bool {
	switch e.Type.(type) {
	case *MessageReceiveEventType:
		return true
	}
	return false
}

// Timeout returns the corresponding timeout if the event is of type either
// timeout start or timeout end. The second return value is false otherwise.
func (e *Event) Timeout() (*ReplicaTimeout, bool) {
	switch eType := e.Type.(type) {
	case *TimeoutStartEventType:
		return eType.Timeout, true
	case *TimeoutEndEventType:
		return eType.Timeout, true
	}
	return nil, false
}

// IsTimeoutStart returns true when the event is of type timeout start.
func (e *Event) IsTimeoutStart() bool {
	switch e.Type.(type) {
	case *TimeoutStartEventType:
		return true
	}
	return false
}

// IsTimeoutEnd returns true when the event is of type timeout end.
func (e *Event) IsTimeoutEnd() bool {
	switch e.Type.(type) {
	case *TimeoutEndEventType:
		return true
	}
	return false
}

// IsGeneric is true when the event is of type [GenericEventType].
func (e *Event) IsGeneric() bool {
	switch e.Type.(type) {
	case *GenericEventType:
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
