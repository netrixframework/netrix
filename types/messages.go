package types

import (
	"errors"
	"fmt"
)

var (
	ErrDuplicateSubs = errors.New("duplicate subscriber")
	ErrNoSubs        = errors.New("subscriber does not exist")
	ErrNoData        = errors.New("no data in message")
	ErrBadParser     = errors.New("bad parser")

	DefaultSubsChSize = 10
)

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

// Clone to create a new Message object with the same attributes
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
