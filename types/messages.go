package types

import (
	"errors"
	"fmt"
)

// Errors that can occur when parsing a message
var (
	// ErrNoData occurs when [Message.Data] is empty
	ErrNoData = errors.New("no data in message")
	// ErrBadParse occurs when [MessageParser] is nil
	ErrBadParser = errors.New("bad parser")
)

// MessageID is a unique identifier for each message
type MessageID string

// Message stores a message that has been intercepted between two replicas
type Message struct {
	From          ReplicaID     `json:"from"`
	To            ReplicaID     `json:"to"`
	Data          []byte        `json:"data"`
	Type          string        `json:"type"`
	ID            MessageID     `json:"id"`
	Intercept     bool          `json:"intercept"`
	ParsedMessage ParsedMessage `json:"-"`
	Repr          string        `json:"repr"`
}

// GetParsedMessage[V] type casts [Message.ParsedMessage] to the specified type V.
func GetParsedMessage[V any](m *Message) (V, bool) {
	var result V
	if m.ParsedMessage == nil {
		return result, false
	}
	result, ok := m.ParsedMessage.(V)
	return result, ok
}

// Clone to create a new Message object with the same attributes.
func (m *Message) Clone() Clonable {
	return &Message{
		From:          m.From,
		To:            m.To,
		Data:          m.Data,
		Type:          m.Type,
		ID:            m.ID,
		Intercept:     m.Intercept,
		Repr:          m.Repr,
		ParsedMessage: nil,
	}
}

// Parse invokes [MessageParser.Parse] with [Message.Data] and the result is stored in [Message.ParsedMessage].
func (m *Message) Parse(parser MessageParser) error {
	if parser == nil {
		return ErrBadParser
	}
	if m.Data == nil {
		return ErrNoData
	}
	p, err := parser.Parse(m.Data)
	if err != nil {
		return fmt.Errorf("parse error: %s", err)
	}
	m.ParsedMessage = p
	m.Repr = p.String()
	return nil
}

// Name returns a formatted string containing from, to and repr
// Name is not unique for every message, use ID for that purpose
func (m *Message) Name() string {
	return fmt.Sprintf("%s_%s_%s", m.From, m.To, m.Repr)
}
