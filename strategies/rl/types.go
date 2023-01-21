package rl

import (
	"crypto/sha256"
	"encoding/hex"
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

func (t *Trace) Hash() string {
	traceStr := "["
	for i := 0; i < t.stateSequence.Size(); i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		traceStr += fmt.Sprintf("(%s, %s),", state.Hash(), action.Name())
	}
	traceStr = traceStr[0 : len(traceStr)-1]
	traceStr += "]"

	hash := sha256.Sum256([]byte(traceStr))
	return hex.EncodeToString(hash[:])
}

func (t *Trace) unwrappedHash() string {
	traceStr := "["
	for i := 0; i < t.stateSequence.Size(); i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		traceStr += fmt.Sprintf("(%s, %s),", state.InterpreterState.Hash(), action.Name())
	}
	traceStr = traceStr[0 : len(traceStr)-1]
	traceStr += "]"

	hash := sha256.Sum256([]byte(traceStr))
	return hex.EncodeToString(hash[:])
}

func (t *Trace) Strings() []string {
	result := make([]string, t.stateSequence.Size())
	for i := 0; i < t.stateSequence.Size(); i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		result[i] = fmt.Sprintf("{state:%s, action: %s}", state.String(), action.Name())
	}
	return result
}

func (t *Trace) unwrappedStrings() []string {
	result := make([]string, t.stateSequence.Size())
	for i := 0; i < t.stateSequence.Size(); i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		result[i] = fmt.Sprintf("(%s, %s)", state.InterpreterState.Hash(), action.Name())
	}
	return result
}
