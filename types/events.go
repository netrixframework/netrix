package types

import (
	"sync"

	"github.com/netrixframework/netrix/log"
)

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
	ID uint64 `json:"id"`
	// Timestamp of the event
	Timestamp int64 `json:"timestamp"`
	// Vector clock value of the event
	ClockValue ClockValue
}

func NewEvent(replica ReplicaID, t EventType, ts string, id uint64, time int64) *Event {
	return &Event{
		Replica:    replica,
		Type:       t,
		TypeS:      ts,
		ID:         id,
		Timestamp:  time,
		ClockValue: nil,
	}
}

// Returns true if the current event is less than `other`
func (e *Event) Lt(other *Event) bool {
	if e.ClockValue == nil {
		return true
	}
	if other.ClockValue == nil {
		return false
	}
	return e.ClockValue.Lt(other.ClockValue)
}

func (e *Event) MessageID() (string, bool) {
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

// EventQueue datastructure to store the messages in a FIFO queue
type EventQueue struct {
	events      []*Event
	subscribers map[string]chan *Event
	lock        *sync.Mutex
	size        int
	dispatchWG  *sync.WaitGroup
	*BaseService
}

// NewEventQueue returns an empty EventQueue
func NewEventQueue(logger *log.Logger) *EventQueue {
	return &EventQueue{
		events:      make([]*Event, 0),
		size:        0,
		subscribers: make(map[string]chan *Event),
		lock:        new(sync.Mutex),
		dispatchWG:  new(sync.WaitGroup),
		BaseService: NewBaseService("EventQueue", logger),
	}
}

// Start implements Service
func (q *EventQueue) Start() error {
	q.StartRunning()
	go q.dispatchloop()
	return nil
}

func (q *EventQueue) dispatchloop() {
	for {
		q.lock.Lock()
		size := q.size
		messages := q.events
		subcribers := q.subscribers
		q.lock.Unlock()

		if size > 0 {
			toAdd := messages[0]

			for _, s := range subcribers {
				q.dispatchWG.Add(1)
				go func(subs chan *Event) {
					select {
					case subs <- toAdd.Clone().(*Event):
					case <-q.QuitCh():
					}
					q.dispatchWG.Done()
				}(s)
			}

			q.lock.Lock()
			q.size = q.size - 1
			q.events = q.events[1:]
			q.lock.Unlock()
		}
	}
}

// Stop implements Service
func (q *EventQueue) Stop() error {
	q.StopRunning()
	q.dispatchWG.Wait()
	return nil
}

// Add adds a message to the queue
func (q *EventQueue) Add(m *Event) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.events = append(q.events, m)
	q.size = q.size + 1
}

// Flush clears the queue of all messages
func (q *EventQueue) Flush() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.events = make([]*Event, 0)
	q.size = 0
}

// Restart implements Service
func (q *EventQueue) Restart() error {
	q.Flush()
	return nil
}

// Subscribe creates and returns a channel for the subscriber with the given label
func (q *EventQueue) Subscribe(label string) chan *Event {
	q.lock.Lock()
	defer q.lock.Unlock()
	ch, ok := q.subscribers[label]
	if ok {
		return ch
	}
	newChan := make(chan *Event, 10)
	q.subscribers[label] = newChan
	return newChan
}
