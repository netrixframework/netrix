package rl

import (
	"fmt"

	"github.com/netrixframework/netrix/types"
)

type wrappedState struct {
	State
	pendingMessages []*types.Message
}

func (w *wrappedState) Hash() string {
	out := fmt.Sprintf("state:{%s},messages:{", w.State.Hash())
	for i, m := range w.pendingMessages {
		out += string(m.ID)
		if i < len(w.pendingMessages)-1 {
			out += ","
		}
	}
	out += "}"
	return out
}

func (w *wrappedState) Actions() []*Action {
	actions := make([]*Action, len(w.pendingMessages))
	for i, m := range w.pendingMessages {
		actions[i] = DeliverMessageAction(m)
	}
	return actions
}

func (r *RLStrategy) wrapState(state State) *wrappedState {
	return &wrappedState{
		State:           state,
		pendingMessages: r.pendingMessages.IterValues(),
	}
}
