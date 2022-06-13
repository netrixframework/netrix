package timeout

import (
	"sync"
	"time"

	"github.com/netrixframework/netrix/types"
)

type timeout struct {
	*types.ReplicaTimeout
	DoneCh chan struct{}
	Once   *sync.Once
}

func (t *timeout) Stop() {
	t.Once.Do(func() {
		close(t.DoneCh)
	})
}

func newTimeout(t *types.ReplicaTimeout) *timeout {
	return &timeout{
		ReplicaTimeout: t,
		DoneCh:         make(chan struct{}),
		Once:           new(sync.Once),
	}
}

type timer struct {
	timeouts     map[string]*timeout
	doneTimeouts []*types.ReplicaTimeout

	lock *sync.Mutex
}

func newTimer() *timer {
	return &timer{
		timeouts:     make(map[string]*timeout),
		doneTimeouts: make([]*types.ReplicaTimeout, 0),
		lock:         new(sync.Mutex),
	}
}

func (t *timer) spawn(to *timeout) {
	select {
	case <-time.After(to.Duration):
		t.lock.Lock()
		_, ok := t.timeouts[to.Key()]
		if ok {
			t.doneTimeouts = append(t.doneTimeouts, to.ReplicaTimeout)
		}
		t.lock.Unlock()
	case <-to.DoneCh:
		return
	}
}

func (t *timer) Add(to *types.ReplicaTimeout) {
	timeout := newTimeout(to)
	t.lock.Lock()
	t.timeouts[to.Key()] = timeout
	t.lock.Unlock()
	go t.spawn(timeout)
}

func (t *timer) Ready() []*types.ReplicaTimeout {
	t.lock.Lock()
	timeouts := t.doneTimeouts
	t.doneTimeouts = make([]*types.ReplicaTimeout, 0)
	t.lock.Unlock()
	return timeouts
}

func (t *timer) EndAll() {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, to := range t.timeouts {
		to.Stop()
	}
	t.timeouts = make(map[string]*timeout)
}
