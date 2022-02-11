package types

import (
	"errors"
	"fmt"
	"sync"

	"github.com/netrixframework/netrix/log"
)

var (
	ErrDuplicateSubs = errors.New("duplicate subscriber")
	ErrNoSubs        = errors.New("subscriber does not exist")
	ErrNoData        = errors.New("no data in message")
	ErrBadParser     = errors.New("bad parser")

	DefaultSubsChSize = 10
)

// Message stores a message that has been interecepted between two replicas
type Message struct {
	From          ReplicaID     `json:"from"`
	To            ReplicaID     `json:"to"`
	Data          []byte        `json:"data"`
	Type          string        `json:"type"`
	ID            string        `json:"id"`
	Intercept     bool          `json:"intercept"`
	ParsedMessage ParsedMessage `json:"-"`
	Repr          string        `json:"repr"`
}

// Clone to create a new Message object with the same attributes
func (m *Message) Clone() *Message {
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

// MessageStore to store the messages. Thread safe
type MessageStore struct {
	messages map[string]*Message
	lock     *sync.Mutex
}

// NewMessageStore creates a empty MessageStore
func NewMessageStore() *MessageStore {
	return &MessageStore{
		messages: make(map[string]*Message),
		lock:     new(sync.Mutex),
	}
}

// Get returns a message and bool indicating if the message exists
func (s *MessageStore) Get(id string) (*Message, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	m, ok := s.messages[id]
	return m, ok
}

// Exists returns true if the message exists
func (s *MessageStore) Exists(id string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s.messages[id]
	return ok
}

// Add adds a message to the store
// Returns any old message with the same ID if it exists or nil if not
func (s *MessageStore) Add(m *Message) *Message {
	s.lock.Lock()
	defer s.lock.Unlock()

	old, ok := s.messages[m.ID]
	s.messages[m.ID] = m
	if ok {
		return old
	}
	return nil
}

// Iter returns a list of all the messages in the store
func (s *MessageStore) Iter() []*Message {
	s.lock.Lock()
	defer s.lock.Unlock()
	result := make([]*Message, len(s.messages))
	i := 0
	for _, m := range s.messages {
		result[i] = m
		i++
	}
	return result
}

// Remove returns and deleted the message from the store if it exists. Returns nil otherwise
func (s *MessageStore) Remove(id string) *Message {
	s.lock.Lock()
	defer s.lock.Unlock()
	m, ok := s.messages[id]
	if ok {
		delete(s.messages, id)
		return m
	}
	return nil
}

// Size returns the size of the message store
func (s *MessageStore) Size() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.messages)
}

// RemoveAll empties the message store
func (s *MessageStore) RemoveAll() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.messages = make(map[string]*Message)
}

// MessageQueue datastructure to store the messages in a FIFO queue
type MessageQueue struct {
	messages    []*Message
	subscribers map[string]chan *Message
	lock        *sync.Mutex
	size        int
	dispatchWG  *sync.WaitGroup
	enabled     bool
	*BaseService
}

var _ Service = &MessageQueue{}

// NewMessageQueue returns an empty MessageQueue
func NewMessageQueue(logger *log.Logger) *MessageQueue {
	return &MessageQueue{
		messages:    make([]*Message, 0),
		size:        0,
		subscribers: make(map[string]chan *Message),
		lock:        new(sync.Mutex),
		dispatchWG:  new(sync.WaitGroup),
		enabled:     true,
		BaseService: NewBaseService("MessageQueue", logger),
	}
}

// Start implements Service
func (q *MessageQueue) Start() error {
	q.StartRunning()
	go q.dispatchloop()
	return nil
}

func (q *MessageQueue) dispatchloop() {
	for {
		q.lock.Lock()
		size := q.size
		messages := q.messages

		subscribers := q.subscribers
		q.lock.Unlock()

		if size > 0 {
			toAdd := messages[0]

			for _, s := range subscribers {
				q.dispatchWG.Add(1)
				go func(subs chan *Message) {
					select {
					case subs <- toAdd.Clone():
					case <-q.QuitCh():
					}
					q.dispatchWG.Done()
				}(s)
			}
			q.Pop()
		}
	}
}

func (q *MessageQueue) Pop() (*Message, bool) {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.size < 1 {
		return nil, false
	}
	front := q.messages[0]
	q.size = q.size - 1
	q.messages = q.messages[1:]
	return front, true
}

// Stop implements Service
func (q *MessageQueue) Stop() error {
	q.StopRunning()
	q.dispatchWG.Wait()
	return nil
}

// Add adds a message to the queue
func (q *MessageQueue) Add(m *Message) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if !q.enabled {
		return
	}

	q.messages = append(q.messages, m)
	q.size = q.size + 1
}

// Flush clears the queue of all messages
func (q *MessageQueue) Flush() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.messages = make([]*Message, 0)
	q.size = 0
}

// Restart implements Service
func (q *MessageQueue) Restart() error {
	q.Flush()
	return nil
}

// Disable closes the queue and drops all incoming messages
// Use with caution
func (q *MessageQueue) Disable() {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.enabled = false
}

// Enable enqueues the messages and feeds it to the subscribers if any
func (q *MessageQueue) Enable() {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.enabled = true
}

// Subscribe create and returns a channel for the subscriber with the specified label
func (q *MessageQueue) Subscribe(label string) chan *Message {
	q.lock.Lock()
	defer q.lock.Unlock()
	ch, ok := q.subscribers[label]
	if ok {
		return ch
	}
	newChan := make(chan *Message, DefaultSubsChSize)
	q.subscribers[label] = newChan
	return newChan
}
