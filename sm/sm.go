// Package sm defines the state machine used to specify properties to test for.
//
// The package also defines a builder interface that can be used to construct
// the state machine in an intuitive manner.
package sm

import (
	"encoding/json"
	"sync"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

const (
	// StartStateLabel is the start state label of a state machine
	StartStateLabel = "startState"
	// FailStateLabel is the failure state label
	FailStateLabel = "failState"
	// SuccessStateLabel is the state label of the success state
	SuccessStateLabel = "successState"
)

// StateMachineBuilder struct defines a builder pattern to create a state machine
type StateMachineBuilder struct {
	stateMachine *StateMachine
	curState     *State
}

// On can be used to create a transition relation between states based on the specified condition
func (s StateMachineBuilder) On(cond Condition, stateLabel string) StateMachineBuilder {
	next, ok := s.stateMachine.getState(stateLabel)
	if !ok {
		next = s.stateMachine.newState(stateLabel)
	}
	s.curState.Transitions[next.Label] = cond
	s.curState.transitionOrder = append(s.curState.transitionOrder, next.Label)
	return StateMachineBuilder{
		stateMachine: s.stateMachine,
		curState:     next,
	}
}

// MarkSuccess marks the current state of the builder as a success state
func (s StateMachineBuilder) MarkSuccess() StateMachineBuilder {
	s.curState.Success = true
	return s
}

// State of the testcase state machine
type State struct {
	Label           string               `json:"label"`
	Transitions     map[string]Condition `json:"-"`
	Success         bool                 `json:"success"`
	transitionOrder []string
}

// Is returns true if the label matches with the current state label
func (s *State) Is(l string) bool {
	return s.Label == l
}

// Eq returns true if the two state labels are the same
func (s *State) Eq(other *State) bool {
	return s.Label == other.Label
}

// For serializing the
func (s *State) MarshalJSON() ([]byte, error) {
	keyvals := make(map[string]interface{})
	keyvals["label"] = s.Label
	transitions := make([]string, len(s.Transitions))
	i := 0
	for to := range s.Transitions {
		transitions[i] = to
		i++
	}
	keyvals["transitions"] = transitions
	return json.Marshal(keyvals)
}

type run struct {
	start       *State
	curState    *State
	lock        *sync.Mutex
	transitions []string
}

func newRun(start *State) *run {
	return &run{
		start:       start,
		curState:    start,
		lock:        new(sync.Mutex),
		transitions: []string{start.Label},
	}
}

func (r *run) Transition(to *State) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.curState = to
	r.transitions = append(r.transitions, to.Label)
}

func (r *run) GetTransitions() []string {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.transitions
}

func (r *run) CurState() *State {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.curState
}

func (r *run) Reset() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.curState = r.start
	r.transitions = []string{r.start.Label}
}

// StateMachine is a deterministic transition system where the transitions are labelled by conditions
type StateMachine struct {
	states map[string]*State
	run    *run
}

// NewStateMachine instantiate a StateMachine
func NewStateMachine() *StateMachine {
	m := &StateMachine{
		states: make(map[string]*State),
	}
	startState := &State{
		Label:           StartStateLabel,
		Success:         false,
		Transitions:     make(map[string]Condition),
		transitionOrder: make([]string, 0),
	}
	m.states[StartStateLabel] = startState
	m.run = newRun(startState)
	m.states[FailStateLabel] = &State{
		Label:           FailStateLabel,
		Success:         false,
		Transitions:     make(map[string]Condition),
		transitionOrder: make([]string, 0),
	}
	m.states[SuccessStateLabel] = &State{
		Label:           SuccessStateLabel,
		Success:         true,
		Transitions:     make(map[string]Condition),
		transitionOrder: make([]string, 0),
	}
	return m
}

// Builder retruns a StateMachineBuilder instance which provides a builder patter to construct the state machine
func (s *StateMachine) Builder() StateMachineBuilder {
	return StateMachineBuilder{
		stateMachine: s,
		curState:     s.states[StartStateLabel],
	}
}

// CurState return the State that the StateMachine is currently in
func (s *StateMachine) CurState() *State {
	return s.run.CurState()
}

// Transition moves the current stat eof the StateMachine to the specified state
func (s *StateMachine) Transition(to string) {
	state, ok := s.getState(to)
	if ok {
		s.run.Transition(state)
	}
}

func (s *StateMachine) getState(label string) (*State, bool) {
	state, ok := s.states[label]
	return state, ok
}

func (s *StateMachine) newState(label string) *State {
	cur, ok := s.states[label]
	if ok {
		return cur
	}
	newState := &State{
		Label:           label,
		Transitions:     make(map[string]Condition),
		Success:         false,
		transitionOrder: make([]string, 0),
	}
	s.states[label] = newState
	return newState
}

// Step runs the next step of the state machine for the current event.
func (s *StateMachine) Step(e *types.Event, c *Context) {
	state := s.run.CurState()
	for _, to := range state.transitionOrder {
		cond := state.Transitions[to]
		if cond(e, c) {
			next, ok := s.states[to]
			if ok {
				c.Logger.With(log.LogParams{
					"state": to,
				}).Debug("State machine transition")
				s.run.Transition(next)
				c.Vars.Set("curState", to)
			}
			break
		}
	}
}

// InSuccessState returns true if the current state of the state machine is a success state
func (s *StateMachine) InSuccessState() bool {
	return s.run.CurState().Success
}

// InState returns a condition which is true if the StateMachine is in a specific state.
// This can be used to define handler that access the state
func (s *StateMachine) InState(state string) Condition {
	return func(e *types.Event, c *Context) bool {
		curState := s.CurState()
		return curState.Label == state
	}
}

// Reset the state machine to its initial state
func (s *StateMachine) Reset() {
	s.run.Reset()
}
