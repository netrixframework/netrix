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
	withTimeouts     bool
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
	possibleTimeouts := make(map[types.ReplicaID]bool)
	for _, m := range w.pendingMessages {
		aName := m.Name()
		if _, ok := uniqueActions[aName]; !ok {
			uniqueActions[aName] = DeliverMessageAction(m)
		}
		if _, ok := possibleTimeouts[m.To]; !ok {
			possibleTimeouts[m.To] = true
		}
	}
	actions := make([]*Action, 0)
	for _, a := range uniqueActions {
		actions = append(actions, a)
	}
	if w.withTimeouts {
		for to := range possibleTimeouts {
			actions = append(actions, TimeoutReplicaAction(to))
		}
	}
	return actions
}

func (r *RLStrategy) wrapState(state State, new bool) *wrappedState {
	withTimeouts := false
	if new {
		// Toss a coin to decide and store the new state
		if r.timeoutRand.Float64() < r.config.TimeoutEpsilon {
			withTimeouts = true
		}
		r.curTimeoutStateLock.Lock()
		r.curTimeoutState = withTimeouts
		r.curTimeoutStateLock.Unlock()
	} else {
		// Use existing state
		r.curTimeoutStateLock.Lock()
		withTimeouts = r.curTimeoutState
		r.curTimeoutStateLock.Unlock()
	}
	return &wrappedState{
		InterpreterState: state,
		pendingMessages:  r.pendingMessages.IterValues(),
		withTimeouts:     withTimeouts,
	}
}
