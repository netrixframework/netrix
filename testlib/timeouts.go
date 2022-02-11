package testlib

import "github.com/netrixframework/netrix/types"

// Flow of execution
// 	1. Update the timeout context with the current event that is observed
// 		a. Keep track of pending timeouts
// 		b. Keep track of messages that have been delivered but not acknowledged
// 	2. Check delivery of messages that are currently intended to be delivered
// 		a. Easy to modify graph (mark nodes as clean and dirty)
// 		b. Check spuriousness (this can be configured explicitly)
// 		c. Easy algorithm to check spuriousness
// 		d. algorithm to order the timeouts that need to be fired.
// 	3. Update with additional timers that need to be fired
// 		a. Parameterized strategy to pick timeouts
// 		b. Ability for filters to mark certain timeouts to be fired earlier
type TimeoutDriver struct {
	store *types.TimeoutStore
}

func NewTimeoutDriver(store *types.TimeoutStore) *TimeoutDriver {
	return &TimeoutDriver{store}
}

func (t *TimeoutDriver) NewEvent(e *types.Event) {
	if e.IsTimeoutStart() {
		timeout, _ := e.Timeout()
		t.store.AddTimeout(timeout)
	}
}

func (t *TimeoutDriver) ToDispatch() []*types.ReplicaTimeout {
	return t.store.ToDispatch()
}
