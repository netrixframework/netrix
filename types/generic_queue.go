package types

import (
	"math/rand"
	"sync"
	"time"
)

// Clonable is any type which returns a copy of itself on Clone()
type Clonable interface {
	Clone() Clonable
}

// Queue is a generic pub-sub queue that is thread safe
type Queue[V any] struct {
	vals    []V
	lock    *sync.Mutex
	size    int
	discard bool
}

// NewQueue[V] returns an empty Queue[V]
func NewQueue[V any]() *Queue[V] {
	return &Queue[V]{
		vals:    make([]V, 0),
		size:    0,
		lock:    new(sync.Mutex),
		discard: false,
	}
}

// Add adds a message to the queue
func (q *Queue[V]) Add(m V) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.vals = append(q.vals, m)
	q.size = q.size + 1
}

// Pause stops accepting queue elements (discards them instead)
func (q *Queue[V]) Pause() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.discard = true
}

// Resume stops discarding and starts accepting elements
func (q *Queue[V]) Resume() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.discard = false
}

// Reset clears the queue of all messages
func (q *Queue[V]) Reset() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.vals = make([]V, 0)
	q.size = 0
}

// Pop returns an element at the head of the queue
func (q *Queue[V]) Pop() (V, bool) {
	q.lock.Lock()
	defer q.lock.Unlock()

	var result V
	if q.size == 0 {
		return result, false
	}
	result = q.vals[0]
	q.size = q.size - 1
	q.vals = q.vals[1:]
	return result, true
}

// Returns the size of the queue
func (q *Queue[V]) Size() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	return q.size
}

// Channel[V] is a synchronized channel that can be reset (closed and opened) multiple times.
/*
The data structure is useful when we need to close the channel and re-open a new one in a multi threaded
environment. The underlying channel is encapsulated with a Mutex lock.

Example:
	ch := NewChannel[int]

	go func() {
		for {
			select {
			case <-ch.Ch():
				//...
			}
		}
	}()

	ch.BlockingAdd(1)
	ch.BlockingAdd(2)
	ch.Close()
	ch.Open()
	//...
*/
type Channel[V any] struct {
	curChan chan V
	open    bool
	lock    *sync.Mutex
}

// NewChannel[V] creates a Channel[V] object.
func NewChannel[V any]() *Channel[V] {
	return &Channel[V]{
		curChan: make(chan V, 20),
		open:    true,
		lock:    new(sync.Mutex),
	}
}

// Open clears the previous channel and unlocks it
func (c *Channel[V]) Open() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.open {
		c.curChan = make(chan V, 20)
		c.open = true
	}
}

// IsOpen returns true if the channel is open
func (c *Channel[V]) IsOpen() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.open
}

// Close closes the channel
func (c *Channel[V]) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.open {
		close(c.curChan)
		c.open = false
	}
}

// Ch returns the underlying channel that can be used to poll
func (c *Channel[V]) Ch() chan V {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.curChan
}

// NonBlockingAdd adds the element if the underlying channel is not full,
// results in a no-op otherwise.
func (c *Channel[V]) NonBlockingAdd(element V) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.open {
		if len(c.curChan) == cap(c.curChan) {
			return false
		}
		c.curChan <- element
	}
	return true
}

// BlockingAdd waits until the channel is not full to add.
func (c *Channel[V]) BlockingAdd(element V) {
	for !c.NonBlockingAdd(element) {
	}
}

// Reset closes the underlying channel and creates a new one.
func (c *Channel[V]) Reset() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.open {
		close(c.curChan)
	}
	c.curChan = make(chan V, 20)
	c.open = true
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
