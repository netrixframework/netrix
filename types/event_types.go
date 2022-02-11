package types

import (
	"fmt"
	"strings"
)

// MessageSendEventType is the event type where a message is sent from the replica
type MessageSendEventType struct {
	// MessageID of the message that was sent
	MessageID string
}

// NewMessageSendEventType instantiates MessageSendEventType
func NewMessageSendEventType(messageID string) *MessageSendEventType {
	return &MessageSendEventType{
		MessageID: messageID,
	}
}

// Clone returns a copy of the current MessageSendEventType
func (s *MessageSendEventType) Clone() EventType {
	return &MessageSendEventType{
		MessageID: s.MessageID,
	}
}

// Type returns a unique key for MessageSendEventType
func (s *MessageSendEventType) Type() string {
	return "MessageSendEventType"
}

// String returns a string representation of the event type
func (s *MessageSendEventType) String() string {
	return fmt.Sprintf("MessageSend { %s }", s.MessageID)
}

// MessageReceiveEventType is the event type when a replica receives a message
type MessageReceiveEventType struct {
	// MessageID is the ID of the message received
	MessageID string
}

// NewMessageReceiveEventType instantiates MessageReceiveEventType
func NewMessageReceiveEventType(messageID string) *MessageReceiveEventType {
	return &MessageReceiveEventType{
		MessageID: messageID,
	}
}

// Clone returns a copy of the current MessageReceiveEventType
func (r *MessageReceiveEventType) Clone() EventType {
	return &MessageReceiveEventType{
		MessageID: r.MessageID,
	}
}

// Type returns a unique key for MessageReceiveEventType
func (r *MessageReceiveEventType) Type() string {
	return "MessageReceiveEventType"
}

// String returns a string representation of the event type
func (r *MessageReceiveEventType) String() string {
	return fmt.Sprintf("MessageReceive { %s }", r.MessageID)
}

type TimeoutStartEventType struct {
	Timeout *ReplicaTimeout
}

func NewTimeoutStartEventType(timeout *ReplicaTimeout) *TimeoutStartEventType {
	return &TimeoutStartEventType{
		Timeout: timeout,
	}
}

func (ts *TimeoutStartEventType) Clone() EventType {
	return &TimeoutStartEventType{
		Timeout: ts.Timeout,
	}
}

func (ts *TimeoutStartEventType) Type() string {
	return "TimeoutStartEventType"
}

func (ts *TimeoutStartEventType) String() string {
	return fmt.Sprintf("TimeoutStart { %s, %s }", ts.Timeout.Replica, ts.Timeout.Type)
}

type TimeoutEndEventType struct {
	Timeout *ReplicaTimeout
}

func NewTimeoutEndEventType(timeout *ReplicaTimeout) *TimeoutEndEventType {
	return &TimeoutEndEventType{
		Timeout: timeout,
	}
}

func (te *TimeoutEndEventType) Clone() EventType {
	return &TimeoutEndEventType{
		Timeout: te.Timeout,
	}
}

func (te *TimeoutEndEventType) Type() string {
	return "TimeoutEndEventType"
}

func (te *TimeoutEndEventType) String() string {
	return fmt.Sprintf("TimeoutEnd { %s, %s }", te.Timeout.Replica, te.Timeout.Type)
}

// GenericEventType is the event type published by a replica
// It can be specific to the algorithm that is implemented
type GenericEventType struct {
	// Marshalled parameters
	Params map[string]string `json:"params"`
	// Type of event for reference
	// Eg: Commit
	T string `json:"type"`
}

// NewGenericEventType instantiates GenericEventType
func NewGenericEventType(params map[string]string, t string) *GenericEventType {
	return &GenericEventType{
		Params: params,
		T:      t,
	}
}

// Clone returns a copy of the current GenericEventType
func (g *GenericEventType) Clone() EventType {
	return &GenericEventType{
		Params: g.Params,
		T:      g.T,
	}
}

// Type returns a unique key for GenericEventType
func (g *GenericEventType) Type() string {
	return "GenericEvent"
}

// String returns a string representation of the event type
func (g *GenericEventType) String() string {
	str := g.T + " { "
	paramS := make([]string, len(g.Params))
	i := 0
	for k, v := range g.Params {
		paramS[i] = fmt.Sprintf("%s = %s", k, v)
		i++
	}
	str += strings.Join(paramS, " , ")
	str += " }"
	return str
}
