package types

import (
	"errors"
	"math/rand"
	"sync"
)

var (
	// ErrReplicaStoreFull is returned when more than the intended number of replicas register with the scheduler tool
	ErrReplicaStoreFull = errors.New("replica store is full")
)

// ReplicaID is an identifier for the replica encoded as a string
type ReplicaID string

// Replica immutable representation of the attributes of a replica
type Replica struct {
	ID    ReplicaID              `json:"id"`
	Ready bool                   `json:"ready"`
	Info  map[string]interface{} `json:"info"`
	Addr  string                 `json:"addr"`
	index int
}

// ReplicaStore to store all replica information, thread safe
type ReplicaStore struct {
	replicas map[ReplicaID]*Replica
	index    []ReplicaID
	lock     *sync.Mutex
	cap      int
	size     int
}

// NewReplicaStore creates an empty ReplicaStore
func NewReplicaStore(size int) *ReplicaStore {
	return &ReplicaStore{
		replicas: make(map[ReplicaID]*Replica),
		lock:     new(sync.Mutex),
		cap:      size,
		size:     0,
	}
}

// Add adds or updates a replica to the store
func (s *ReplicaStore) Add(p *Replica) {
	s.lock.Lock()
	defer s.lock.Unlock()
	existing, ok := s.replicas[p.ID]
	if !ok {
		s.index = append(s.index, p.ID)
		p.index = s.size
		s.size++
	} else {
		p.index = existing.index
	}
	s.replicas[p.ID] = p
}

// Get returns the replica and a bool indicating if it exists or not
func (s *ReplicaStore) Get(id ReplicaID) (p *Replica, ok bool) {
	s.lock.Lock()
	p, ok = s.replicas[id]
	s.lock.Unlock()
	return
}

// NumReady returns the number of replicas with Ready attribute set to true
func (s *ReplicaStore) NumReady() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	count := 0
	for _, r := range s.replicas {
		if r.Ready {
			count = count + 1
		}
	}
	return count
}

// Count returns the total number of replicas
func (s *ReplicaStore) Count() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.size
}

// Iter returns a list of the existing replicas
func (s *ReplicaStore) Iter() []*Replica {
	s.lock.Lock()
	defer s.lock.Unlock()
	replicas := make([]*Replica, s.cap)
	i := 0
	for _, p := range s.index {
		replicas[i] = s.replicas[p]
		i++
	}
	return replicas
}

// ResetReady sets the Ready attribute of all replicas to false
func (s *ReplicaStore) ResetReady() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, p := range s.replicas {
		p.Ready = false
	}
}

// Cap returns the set of replicas used for the test
func (s *ReplicaStore) Cap() int {
	return s.cap
}

func (s *ReplicaStore) GetRandom() (*Replica, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.replicas) == 0 {
		return nil, false
	}
	r := make([]ReplicaID, len(s.replicas))
	i := 0
	for id := range s.replicas {
		r[i] = id
		i++
	}
	randomID := r[rand.Intn(len(s.replicas))]
	return s.replicas[randomID], true
}
