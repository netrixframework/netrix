package rl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type ActionType string

var (
	DeliverMessage ActionType = "DeliverMessage"
	TimeoutReplica ActionType = "TimeoutReplica"
)

type State interface {
	Hash() string
}

type Action struct {
	Type    ActionType
	message *types.Message
	replica types.ReplicaID
}

func (a *Action) Name() string {
	switch a.Type {
	case DeliverMessage:
		return a.message.Name()
	case TimeoutReplica:
		return fmt.Sprintf("Timeout_%s", a.replica)
	default:
		return ""
	}
}

func DeliverMessageAction(message *types.Message) *Action {
	return &Action{
		Type:    DeliverMessage,
		message: message,
	}
}

func TimeoutReplicaAction(replica types.ReplicaID) *Action {
	return &Action{
		Type:    TimeoutReplica,
		replica: replica,
	}
}

type Policy interface {
	NextAction(int, State, []*Action) (*Action, bool)
	Update(int, State, *Action, State)
	NextIteration(int, *Trace)
}

type Interpreter interface {
	Update(*types.Event, *strategies.Context)
	CurState() State
	Reset()
}

type Trace struct {
	stateSequence  *types.List[*wrappedState]
	actionSequence *types.List[*Action]
}

func NewTrace() *Trace {
	return &Trace{
		stateSequence:  types.NewEmptyList[*wrappedState](),
		actionSequence: types.NewEmptyList[*Action](),
	}
}

func (t *Trace) Add(state *wrappedState, action *Action) {
	t.stateSequence.Append(state)
	t.actionSequence.Append(action)
}

func (t *Trace) Get(i int) (State, *Action, bool) {
	state, ok := t.stateSequence.Elem(i)
	if !ok {
		return nil, nil, false
	}
	action, ok := t.actionSequence.Elem(i)
	if !ok {
		return nil, nil, false
	}
	return state, action, true
}

func (t *Trace) Length() int {
	return t.stateSequence.Size()
}

func (t *Trace) Reset() {
	t.stateSequence.RemoveAll()
	t.actionSequence.RemoveAll()
}

type traceEvent struct {
	State  string
	Action string
}

func (t *Trace) Hash() string {
	len := t.stateSequence.Size()
	events := make([]*traceEvent, len)
	for i := 0; i < len; i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		events[i] = &traceEvent{
			State:  state.Hash(),
			Action: action.Name(),
		}
	}
	b, _ := json.Marshal(events)

	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

func (t *Trace) unwrappedHash() string {
	len := t.stateSequence.Size()
	events := make([]*traceEvent, len)
	for i := 0; i < len; i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		events[i] = &traceEvent{
			State:  state.InterpreterState.Hash(),
			Action: action.Name(),
		}
	}
	b, _ := json.Marshal(events)

	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

func (t *Trace) Strings() []string {
	len := t.stateSequence.Size()
	result := make([]string, len)
	for i := 0; i < len; i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		b, _ := json.Marshal(&traceEvent{
			State:  state.String(),
			Action: action.Name(),
		})
		result[i] = string(b)
	}
	return result
}

func (t *Trace) unwrappedStrings() []string {
	len := t.stateSequence.Size()
	result := make([]string, len)
	for i := 0; i < len; i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		b, _ := json.Marshal(&traceEvent{
			State:  state.InterpreterState.Hash(),
			Action: action.Name(),
		})
		result[i] = string(b)
	}
	return result
}
