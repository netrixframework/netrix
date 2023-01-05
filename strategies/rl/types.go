package rl

import (
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type State interface {
	Hash() string
	Actions() []*strategies.Action
}

type Policy interface {
	NextAction(int, State, []*strategies.Action) (*strategies.Action, bool)
	Update(int, State, *strategies.Action, State)
	NextIteration(int, *Trace)
}

type Interpreter interface {
	Update(*types.Event, *strategies.Context)
	CurState() State
	Reset()
}

type Trace struct {
	StateSequence  *types.List[State]
	ActionSequence *types.List[*strategies.Action]
}

func NewTrace() *Trace {
	return &Trace{
		StateSequence:  types.NewEmptyList[State](),
		ActionSequence: types.NewEmptyList[*strategies.Action](),
	}
}

func (t *Trace) Add(state State, action *strategies.Action) {
	t.StateSequence.Append(state)
	t.ActionSequence.Append(action)
}

func (t *Trace) Get(i int) (State, *strategies.Action, bool) {
	state, ok := t.StateSequence.Elem(i)
	if !ok {
		return nil, nil, false
	}
	action, ok := t.ActionSequence.Elem(i)
	if !ok {
		return nil, nil, false
	}
	return state, action, true
}

func (t *Trace) Length() int {
	return t.StateSequence.Size()
}

func (t *Trace) Reset() {
	t.StateSequence.RemoveAll()
	t.ActionSequence.RemoveAll()
}
