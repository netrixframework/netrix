package rl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/netrixframework/netrix/types"
)

type wrappedState struct {
	InterpreterState State
	pendingMessages  []*types.Message
}

func (w *wrappedState) Hash() string {
	out := w.String()
	hash := sha256.Sum256([]byte(out))
	return hex.EncodeToString(hash[:])
}

func (w *wrappedState) String() string {
	actionsMap := make(map[string]bool)
	actions := make([]string, 0)

	for _, m := range w.pendingMessages {
		mName := m.Name()
		if _, ok := actionsMap[mName]; !ok {
			actionsMap[mName] = true
			actions = append(actions, mName)
		}
	}
	sort.Strings(actions)
	b, _ := json.Marshal(struct {
		State    string
		Messages []string
	}{
		State:    w.InterpreterState.Hash(),
		Messages: actions,
	})
	return string(b)
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
