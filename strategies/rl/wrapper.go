package rl

import (
	"crypto/sha256"
	"fmt"

	"github.com/netrixframework/netrix/types"
)

type wrappedState struct {
	InterpreterState State
	pendingMessages  []*types.Message
}

func (w *wrappedState) Hash() string {
	out := fmt.Sprintf("state:{%s},messages:{", w.InterpreterState.Hash())
	for i, m := range w.pendingMessages {
		out += m.Name()
		if i < len(w.pendingMessages)-1 {
			out += ","
		}
	}
	out += "}"
	hash := sha256.Sum256([]byte(out))
	return string(hash[:])
}

func (w *wrappedState) Actions() []*Action {
	uniqueActions := make(map[string]*Action)
	for _, m := range w.pendingMessages {
		aName := m.Name()
		if _, ok := uniqueActions[aName]; !ok {
			uniqueActions[aName] = DeliverMessageAction(m)
		}
	}
	actions := make([]*Action, len(uniqueActions))
	i := 0
	for _, a := range uniqueActions {
		actions[i] = a
		i++
	}
	return actions
}

func (r *RLStrategy) wrapState(state State) *wrappedState {
	return &wrappedState{
		InterpreterState: state,
		pendingMessages:  r.pendingMessages.IterValues(),
	}
}
