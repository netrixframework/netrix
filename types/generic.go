package types

import (
	"math/rand"
	"sync"
	"time"

	"github.com/netrixframework/netrix/log"
	"golang.org/x/exp/constraints"
)

type Map[T constraints.Ordered, V any] struct {
	m    map[T]V
	lock *sync.Mutex
}

func NewMap[T constraints.Ordered, V any]() *Map[T, V] {
	return &Map[T, V]{
		m:    make(map[T]V),
		lock: new(sync.Mutex),
	}
}

func (s *Map[T, V]) Get(key T) (V, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	val, ok := s.m[key]
	return val, ok
}

func (s *Map[T, V]) Add(key T, val V) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.m[key] = val
}

func (s *Map[T, V]) Remove(key T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.m, key)
}

func (s *Map[T, V]) Exists(key T) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.m[key]
	return ok
}

func (s *Map[T, V]) Size() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.m)
}

func (s *Map[T, V]) RemoveAll() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.m = make(map[T]V)
}

func (s *Map[T, V]) IterValues() []V {
	s.lock.Lock()
	defer s.lock.Unlock()
	vals := make([]V, len(s.m))
	i := 0
	for _, v := range s.m {
		vals[i] = v
		i++
	}
	return vals
}

func (s *Map[T, V]) ToMap() map[T]V {
	s.lock.Lock()
	defer s.lock.Unlock()
	m := make(map[T]V)
	for k, v := range s.m {
		m[k] = v
	}
	return m
}

func (s *Map[T, V]) RandomValue() (V, bool) {
	return s.RandomValueWithSource(rand.NewSource(time.Now().UnixMilli()))
}

func (s *Map[T, V]) RandomValueWithSource(src rand.Source) (V, bool) {
	r := rand.New(src)

	s.lock.Lock()
	keys := make([]T, len(s.m))
	i := 0
	for k := range s.m {
		keys[i] = k
		i++
	}
	s.lock.Unlock()
	if len(keys) == 0 {
		var r V
		return r, false
	}

	rID := keys[r.Intn(len(keys))]
	return s.Get(rID)
}

// Clonable is any type which returns a copy of itself on Clone()
type Clonable interface {
	Clone() Clonable
}

type Queue[V Clonable] struct {
	vals        []V
	subscribers map[string]chan V
	lock        *sync.Mutex
	size        int
	dispatchWG  *sync.WaitGroup
	*BaseService
}

// NewQueue[V] returns an empty Queue[V]
func NewQueue[V Clonable](logger *log.Logger) *Queue[V] {
	return &Queue[V]{
		vals:        make([]V, 0),
		size:        0,
		subscribers: make(map[string]chan V),
		lock:        new(sync.Mutex),
		dispatchWG:  new(sync.WaitGroup),
		BaseService: NewBaseService("Queue[V]", logger),
	}
}

// Start implements Service
func (q *Queue[V]) Start() error {
	q.StartRunning()
	go q.dispatchLoop()
	return nil
}

func (q *Queue[V]) dispatchLoop() {
	for {
		q.lock.Lock()
		size := q.size
		vals := q.vals
		subscribers := q.subscribers
		q.lock.Unlock()

		if size > 0 {
			toAdd := vals[0]

			for _, s := range subscribers {
				q.dispatchWG.Add(1)
				go func(subs chan V) {
					select {
					case subs <- toAdd.Clone().(V):
					case <-q.QuitCh():
					}
					q.dispatchWG.Done()
				}(s)
			}

			q.lock.Lock()
			q.size = q.size - 1
			q.vals = q.vals[1:]
			q.lock.Unlock()
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

	q.vals = append(q.vals, m)
	q.size = q.size + 1
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
	return nil
}

// Subscribe creates and returns a channel for the subscriber with the given label
func (q *Queue[V]) Subscribe(label string) chan V {
	q.lock.Lock()
	defer q.lock.Unlock()
	ch, ok := q.subscribers[label]
	if ok {
		return ch
	}
	newChan := make(chan V, 10)
	q.subscribers[label] = newChan
	return newChan
}
