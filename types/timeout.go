package types

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/netrixframework/netrix/log"
)

type ReplicaTimeout struct {
	Replica  ReplicaID     `json:"replica"`
	Type     string        `json:"type"`
	Duration time.Duration `json:"duration"`
}

func (t *ReplicaTimeout) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"replica":  string(t.Replica),
		"type":     t.Type,
		"duration": t.Duration.String(),
	})
}

func (t *ReplicaTimeout) Eq(other *ReplicaTimeout) bool {
	return t.Replica == other.Replica && t.Type == other.Type && t.Duration == other.Duration
}

func TimeoutFromParams(replica ReplicaID, params map[string]string) (*ReplicaTimeout, bool) {
	t := &ReplicaTimeout{
		Replica: replica,
	}
	ttype, ok := params["type"]
	if !ok {
		return nil, false
	}
	t.Type = ttype
	durS, ok := params["duration"]
	dur, err := time.ParseDuration(durS)
	if !ok || err != nil {
		return nil, false
	}
	t.Duration = dur
	return t, true
}

type timeoutWrapper struct {
	timeout    *ReplicaTimeout
	startEvent *Event
	endEvent   *Event

	lock *sync.Mutex
}

func newTimeoutWrapper(timeout *ReplicaTimeout) *timeoutWrapper {
	return &timeoutWrapper{
		timeout:    timeout,
		startEvent: nil,
		endEvent:   nil,
		lock:       new(sync.Mutex),
	}
}

func (t *timeoutWrapper) SetStart(e *Event) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.startEvent = e
}
func (t *timeoutWrapper) SetEnd(e *Event) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.endEvent = e
}
func (t *timeoutWrapper) HasEnded() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.endEvent == nil
}

type TimeoutContext struct {
	PendingTimeouts map[string]*timeoutWrapper
	PendingReceives map[string]*Event

	eventDag *EventDAG
	lock     *sync.Mutex
}

func NewTimeoutContext(d *EventDAG) *TimeoutContext {
	return &TimeoutContext{
		PendingTimeouts: make(map[string]*timeoutWrapper),
		PendingReceives: make(map[string]*Event),

		eventDag: d,
		lock:     new(sync.Mutex),
	}
}

func (t *TimeoutContext) AddEvent(e *Event) {
	if e.IsTimeoutStart() {
		timeout, _ := e.Timeout()
		key := fmt.Sprintf("%s_%s", e.Replica, timeout.Type)
		t.lock.Lock()
		if _, ok := t.PendingTimeouts[key]; !ok {
			t.PendingTimeouts[key] = newTimeoutWrapper(timeout)
			t.PendingTimeouts[key].SetStart(e)
		} else {
			panic("Not expecting two timeouts of the same type in the same replica")
		}
		t.lock.Unlock()
	} else if e.IsTimeoutEnd() {
		timeout, _ := e.Timeout()
		key := fmt.Sprintf("%s_%s", e.Replica, timeout.Type)
		t.lock.Lock()
		timeoutW, ok := t.PendingTimeouts[key]
		if ok {
			timeoutW.SetEnd(e)
		}
		delete(t.PendingTimeouts, key)
		t.lock.Unlock()
	}
}

func (t *TimeoutContext) CanDeliverMessages(messages []*Message) {

}

type TimeoutStore struct {
	timeouts map[string]*ReplicaTimeout
	outChan  chan *ReplicaTimeout
	wg       *sync.WaitGroup
	ready    []*ReplicaTimeout
	*BaseService
}

var _ Service = &TimeoutStore{}

func NewTimeoutStore(logger *log.Logger) *TimeoutStore {
	return &TimeoutStore{
		timeouts:    make(map[string]*ReplicaTimeout),
		outChan:     make(chan *ReplicaTimeout, 10),
		wg:          new(sync.WaitGroup),
		ready:       make([]*ReplicaTimeout, 0),
		BaseService: NewBaseService("TimeoutStore", logger),
	}
}

func (s *TimeoutStore) AddTimeout(t *ReplicaTimeout) {
	s.lock.Lock()
	key := fmt.Sprintf("%s_%s", t.Replica, t.Type)
	if _, ok := s.timeouts[key]; ok {
		panic("Not expecting multiple timeouts of same type in the same replica")
	}
	s.timeouts[key] = t
	s.lock.Unlock()
	s.Logger.With(log.LogParams{
		"type":     t.Type,
		"duration": t.Duration.String(),
		"replica":  t.Replica,
	}).Debug("Scheduling timeout")
	s.wg.Add(1)
	go s.scheduleTimeout(t)
}

func (s *TimeoutStore) scheduleTimeout(t *ReplicaTimeout) {
	select {
	case <-time.After(t.Duration):
		s.outChan <- t
	case <-s.QuitCh():
	}
	s.wg.Done()
}

func (s *TimeoutStore) Start() error {
	s.StartRunning()
	go s.poll()
	return nil
}

func (s *TimeoutStore) Stop() error {
	s.StopRunning()
	s.flush()
	s.wg.Wait()
	return nil
}

func (s *TimeoutStore) flush() {
	for {
		len := len(s.outChan)
		if len == 0 {
			return
		}
		<-s.outChan
	}
}

func (s *TimeoutStore) Reset() {
	s.flush()
	s.lock.Lock()
	s.ready = make([]*ReplicaTimeout, 0)
	s.timeouts = make(map[string]*ReplicaTimeout)
	s.lock.Unlock()
}

func (s *TimeoutStore) ToDispatch() []*ReplicaTimeout {
	result := make([]*ReplicaTimeout, 0)
	s.lock.Lock()
	result = append(result, s.ready...)
	s.ready = make([]*ReplicaTimeout, 0)
	s.lock.Unlock()
	return result
}

func (s *TimeoutStore) poll() {
	for {
		select {
		case t := <-s.outChan:
			s.Logger.With(log.LogParams{
				"type":     t.Type,
				"duration": t.Duration.String(),
				"replica":  t.Replica,
			}).Debug("Ending timeout")
			s.lock.Lock()
			key := fmt.Sprintf("%s_%s", t.Replica, t.Type)
			_, ok := s.timeouts[key]
			if ok {
				s.ready = append(s.ready, t)
				delete(s.timeouts, key)
			}
			s.lock.Unlock()
		case <-s.QuitCh():
			return
		}
	}
}
