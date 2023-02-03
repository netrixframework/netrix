package fuzzing

import (
	"github.com/netrixframework/netrix/types"
)

type Trace struct {
	*types.List[State]
}

func NewTrace() *Trace {
	return &Trace{
		List: types.NewEmptyList[State](),
	}
}

func (t *Trace) HaveNewState(known *types.Map[string, State]) bool {
	new := false
	for _, state := range t.Iter() {
		stateHash := state.Hash()
		if !known.Exists(stateHash) {
			new = true
			known.Add(stateHash, state)
		}
	}
	return new
}

func (t *Trace) Reset() {
	t.RemoveAll()
}
