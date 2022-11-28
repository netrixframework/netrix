package rl

import (
	"math"
	"sync"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type State interface {
	Hash() string
}

type Interpreter interface {
	RewardFunc(State, *strategies.Action) float64
	Interpret(*types.Event, *strategies.Context) State
	Actions(State, *strategies.Context) []*strategies.Action
}

type Policy interface {
	NextAction(*StateActionMap, State, []*strategies.Action) (*strategies.Action, bool)
}

type StateActionMap struct {
	states    map[string]State
	stateLock *sync.Mutex
	m         map[string]map[string]float64
	lock      *sync.Mutex
}

func newStateActionMap() *StateActionMap {
	return &StateActionMap{
		m:         make(map[string]map[string]float64),
		lock:      new(sync.Mutex),
		states:    make(map[string]State),
		stateLock: new(sync.Mutex),
	}
}

func (s *StateActionMap) AddState(state State) {
	s.stateLock.Lock()
	s.states[state.Hash()] = state
	s.stateLock.Unlock()

	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.m[state.Hash()]
	if !ok {
		s.m[state.Hash()] = make(map[string]float64)
	}
}

func (s *StateActionMap) GetState(key string) (State, bool) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	state, ok := s.states[key]
	return state, ok
}

func (s *StateActionMap) MaxQ(state string) (float64, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	actions, ok := s.m[state]
	if !ok {
		return 0, false
	}
	var max float64 = math.MinInt64
	for _, v := range actions {
		if v > max {
			max = v
		}
	}
	return max, true
}

func (s *StateActionMap) Update(state, action string, val float64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.m[state]
	if !ok {
		s.m[state] = make(map[string]float64)
	}
	s.m[state][action] = val
}

func (s *StateActionMap) Get(state, action string) (float64, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	actions, ok := s.m[state]
	if !ok {
		return 0, false
	}
	val, ok := actions[action]
	return val, ok
}

func (s *StateActionMap) ExistsValue(state, action string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	actions, ok := s.m[state]
	if !ok {
		return false
	}
	_, ok = actions[action]
	return ok
}
