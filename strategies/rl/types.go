package rl

import (
	"fmt"
	"hash/crc32"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type State interface {
	Hash() string
}

type Action struct {
	Type    string
	message *types.Message
	replica types.ReplicaID
}

func (a *Action) Name() string {
	switch a.Type {
	case "DeliverMessage":
		return fmt.Sprintf("%s_%s_%s", a.message.From, a.message.To, a.message.Repr)
	case "TimeoutReplica":
		return fmt.Sprintf("Timeout_%s", a.replica)
	default:
		return ""
	}
}

func DeliverMessageAction(message *types.Message) *Action {
	return &Action{
		Type:    "DeliverMessage",
		message: message,
	}
}

func TimeoutReplica(replica types.ReplicaID) *Action {
	return &Action{
		Type:    "TimeoutReplica",
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
	hasher := crc32.New(crc32.MakeTable(crc32.IEEE))

	traceStr := "["
	for i := 0; i < t.stateSequence.Size(); i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		traceStr += fmt.Sprintf("(%s, %s),", state.Hash(), action.Name())
	}
	traceStr = traceStr[0 : len(traceStr)-1]
	traceStr += "]"

	return string(hasher.Sum([]byte(traceStr)))
}

func (t *Trace) unwrappedHash() string {
	hasher := crc32.New(crc32.MakeTable(crc32.IEEE))

	traceStr := "["
	for i := 0; i < t.stateSequence.Size(); i++ {
		state, _ := t.stateSequence.Elem(i)
		action, _ := t.actionSequence.Elem(i)
		traceStr += fmt.Sprintf("(%s, %s),", state.State.Hash(), action.Name())
	}
	traceStr = traceStr[0 : len(traceStr)-1]
	traceStr += "]"

	return string(hasher.Sum([]byte(traceStr)))
}
