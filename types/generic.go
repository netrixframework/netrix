package types

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/exp/constraints"
)

// Map[T,V] is a generic thread safe map of key type [T] and value type [V]
type Map[T constraints.Ordered, V any] struct {
	m    map[T]V
	lock *sync.Mutex
}

// NewMap[T,V] creates an empty Map
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
	indexes := rand.Perm(len(s.m))
	i := 0
	for _, v := range s.m {
		vals[indexes[i]] = v
		i++
	}
	return vals
}

func (s *Map[T, V]) Keys() []T {
	s.lock.Lock()
	defer s.lock.Unlock()

	keys := make([]T, len(s.m))
	indexes := rand.Perm(len(s.m))
	i := 0
	for k := range s.m {
		keys[indexes[i]] = k
		i++
	}
	return keys
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

// Set[T] is thread safe generic set implementation
type Set[T constraints.Ordered] struct {
	m *Map[T, T]
}

// NewSet[T] creates an empty Set
func NewSet[T constraints.Ordered]() *Set[T] {
	return &Set[T]{
		m: NewMap[T, T](),
	}
}

func (s *Set[T]) Add(elem T) {
	s.m.Add(elem, elem)
}

func (s *Set[T]) Remove(elem T) {
	s.m.Remove(elem)
}

func (s *Set[T]) Contains(elem T) bool {
	return s.m.Exists(elem)
}

func (s *Set[T]) Iter() []T {
	return s.m.Keys()
}

func (s *Set[T]) Size() int {
	return s.m.Size()
}

// List[V] is a generic thread safe list
type List[V any] struct {
	elems []V
	size  int
	lock  *sync.Mutex
}

// NewEmptyList[V] creates an empty List
func NewEmptyList[V any]() *List[V] {
	return &List[V]{
		elems: make([]V, 0),
		size:  0,
		lock:  new(sync.Mutex),
	}
}

// NewList[V] copies the contents of the specified list onto a new List object
func NewList[V any](cur []V) *List[V] {
	elements := make([]V, len(cur))
	copy(elements, cur)
	return &List[V]{
		elems: elements,
		size:  len(cur),
		lock:  new(sync.Mutex),
	}
}

func (l *List[V]) Append(e V) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.elems = append(l.elems, e)
	l.size += 1
}

func (l *List[V]) Elem(index int) (V, bool) {
	l.lock.Lock()
	defer l.lock.Unlock()
	var res V
	if index < 0 || index >= l.size {
		return res, false
	}
	return l.elems[index], true
}

func (l *List[V]) Size() int {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.size
}

func (l *List[V]) Iter() []V {
	l.lock.Lock()
	defer l.lock.Unlock()

	res := make([]V, l.size)
	copy(res, l.elems)
	return res
}

func (l *List[V]) RemoveAll() []V {
	l.lock.Lock()
	defer l.lock.Unlock()
	result := make([]V, l.size)
	copy(result, l.elems)
	l.elems = make([]V, 0)
	l.size = 0
	return result
}

func (l *List[V]) MarshalJSON() ([]byte, error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	return json.Marshal(l.elems)
}

func (l *List[V]) Set(index int, elem V) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.size <= index {
		return false
	}
	l.elems[index] = elem
	return true
}

// Max[T] abstracts the max function for all ordered types T
func Max[T constraints.Ordered](one, two T) T {
	if one > two {
		return one
	}
	return two
}
