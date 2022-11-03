package types

import (
	"math/rand"
	"sync"
	"time"

	"github.com/netrixframework/netrix/log"
)

// Clonable is any type which returns a copy of itself on Clone()
type Clonable interface {
	Clone() Clonable
}

type Queue[V Clonable] struct {
	vals        []V
	subscribers map[string]*Channel[V]
	lock        *sync.Mutex
	size        int
	dispatchWG  *sync.WaitGroup
	discard     bool
	*BaseService
}

// NewQueue[V] returns an empty Queue[V]
func NewQueue[V Clonable](logger *log.Logger) *Queue[V] {
	return &Queue[V]{
		vals:        make([]V, 0),
		size:        0,
		subscribers: make(map[string]*Channel[V]),
		lock:        new(sync.Mutex),
		dispatchWG:  new(sync.WaitGroup),
		discard:     false,
		BaseService: NewBaseService("Queue[V]", logger),
	}
}

// Start implements Service
func (q *Queue[V]) Start() error {
	q.StartRunning()
	// go q.dispatchLoop()
	return nil
}

func (q *Queue[V]) dispatchLoop() {
	for {
		q.lock.Lock()
		size := q.size
		vals := q.vals
		subscribers := q.subscribers
		discard := q.discard
		q.lock.Unlock()

		if size > 0 {
			toAdd := vals[0]
			if !discard {
				for _, s := range subscribers {
					q.dispatchWG.Add(1)
					go func(subs *Channel[V]) {
						select {
						case <-q.QuitCh():
						default:
							subs.BlockingAdd(toAdd.Clone().(V))
						}
						q.dispatchWG.Done()
					}(s)
				}
			}

			q.lock.Lock()
			q.size = q.size - 1
			q.vals = q.vals[1:]
			q.lock.Unlock()
		}

		select {
		case <-q.QuitCh():
			return
		default:
		}

	}
}

// Stop implements Service
func (q *Queue[V]) Stop() error {
	q.StopRunning()
	q.dispatchWG.Wait()
	return nil
}

// Add adds a message to the queue
func (q *Queue[V]) Add(m V) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.discard {
		return
	}

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

// Flush clears the queue of all messages
func (q *Queue[V]) Flush() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.vals = make([]V, 0)
	q.size = 0
}

// Restart implements Service
func (q *Queue[V]) Restart() error {
	q.Flush()
	q.lock.Lock()
	defer q.lock.Unlock()

	for _, s := range q.subscribers {
		s.Reset()
	}
	return nil
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

// Subscribe creates and returns a channel for the subscriber with the given label
func (q *Queue[V]) Subscribe(label string) *Channel[V] {
	q.lock.Lock()
	defer q.lock.Unlock()
	ch, ok := q.subscribers[label]
	if ok {
		return ch
	}
	newChan := NewChannel[V]()
	q.subscribers[label] = newChan
	return newChan
}

type Channel[V any] struct {
	curChan chan V
	open    bool
	lock    *sync.Mutex
}

func NewChannel[V any]() *Channel[V] {
	return &Channel[V]{
		curChan: make(chan V, 20),
		open:    true,
		lock:    new(sync.Mutex),
	}
}

func (c *Channel[V]) Open() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.open {
		c.curChan = make(chan V, 20)
		c.open = true
	}
}

func (c *Channel[V]) IsOpen() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.open
}

func (c *Channel[V]) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.open {
		close(c.curChan)
		c.open = false
	}
}

func (c *Channel[V]) Ch() chan V {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.curChan
}

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

func (c *Channel[V]) BlockingAdd(element V) {
	for !c.NonBlockingAdd(element) {
	}
}

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
