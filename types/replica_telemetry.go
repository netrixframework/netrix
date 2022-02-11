package types

import (
	"sync"

	"github.com/netrixframework/netrix/log"
)

// ReplicaLog encapsulates a log message with the necessary attributes
type ReplicaLog struct {
	// Params is a marhsalled params of the log message
	Params map[string]string `json:"params"`
	// Message is the message that was logged
	Message string `json:"message"`
	// Timestamp of the log
	Timestamp int64 `json:"timestamp"`
	// Replica which posted the log
	Replica ReplicaID `json:"replica"`
}

// Clone implements Clonable
func (l *ReplicaLog) Clone() Clonable {
	return &ReplicaLog{
		Params:    l.Params,
		Message:   l.Message,
		Timestamp: l.Timestamp,
		Replica:   l.Replica,
	}
}

// ReplicaLogStore stores the logs as a map indexed by the replica ID
type ReplicaLogStore struct {
	logs map[ReplicaID][]*ReplicaLog
	lock *sync.Mutex
}

// NewReplicaLogStore instantiates a ReplicaLogStore
func NewReplicaLogStore() *ReplicaLogStore {
	return &ReplicaLogStore{
		logs: make(map[ReplicaID][]*ReplicaLog),
		lock: new(sync.Mutex),
	}
}

// Add adds to the log store
func (store *ReplicaLogStore) Add(log *ReplicaLog) {
	store.lock.Lock()
	defer store.lock.Unlock()

	logs, ok := store.logs[log.Replica]
	if !ok {
		logs = make([]*ReplicaLog, 1)
		logs[0] = log
		store.logs[log.Replica] = logs
		return
	}
	pos := len(logs) - 1
	for {
		if pos < 0 {
			break
		}
		if logs[pos].Timestamp < log.Timestamp {
			break
		}
		pos = pos - 1
	}
	store.logs[log.Replica] = append(store.logs[log.Replica], nil)
	copy(store.logs[log.Replica][pos+2:], store.logs[log.Replica][pos+1:])
	store.logs[log.Replica][pos+1] = log
}

// GetLogs returns the list of logs for a replica where from <=index<to
func (store *ReplicaLogStore) GetLogs(replica ReplicaID, from, to int) ([]*ReplicaLog, int) {
	store.lock.Lock()
	defer store.lock.Unlock()
	logs, ok := store.logs[replica]
	if !ok {
		return []*ReplicaLog{}, 0
	}
	return logs[from:to], len(logs)
}

func (store *ReplicaLogStore) Reset() {
	store.lock.Lock()
	defer store.lock.Unlock()

	for k := range store.logs {
		store.logs[k] = make([]*ReplicaLog, 0)
	}
}

// ReplicaLogQueue is the queue of log messages
type ReplicaLogQueue struct {
	logs        []*ReplicaLog
	subscribers map[string]chan *ReplicaLog
	lock        *sync.Mutex
	size        int
	dispatchWG  *sync.WaitGroup
	*BaseService
}

// NewReplicaLogQueue instantiates ReplicaLogQueue
func NewReplicaLogQueue(logger *log.Logger) *ReplicaLogQueue {
	return &ReplicaLogQueue{
		logs:        make([]*ReplicaLog, 0),
		subscribers: make(map[string]chan *ReplicaLog),
		lock:        new(sync.Mutex),
		size:        0,
		dispatchWG:  new(sync.WaitGroup),
		BaseService: NewBaseService("LogQueue", logger),
	}
}

// Start implements Service
func (q *ReplicaLogQueue) Start() {
	q.StartRunning()
	go q.dispatchloop()
}

// Stop implements Service
func (q *ReplicaLogQueue) Stop() {
	q.StopRunning()
	q.dispatchWG.Wait()
}

func (q *ReplicaLogQueue) dispatchloop() {
	for {
		q.lock.Lock()
		logs := q.logs
		size := q.size
		subscribers := q.subscribers
		q.lock.Unlock()

		if size > 0 {
			toAdd := logs[0]
			for _, s := range subscribers {
				q.dispatchWG.Add(1)
				go func(subs chan *ReplicaLog) {
					select {
					case subs <- toAdd.Clone().(*ReplicaLog):
					case <-q.QuitCh():
					}
					q.dispatchWG.Done()
				}(s)
			}

			q.lock.Lock()
			q.size = q.size - 1
			q.logs = q.logs[1:]
			q.lock.Unlock()
		}
	}
}

// Subscribe creates and returns a channel for the subscriber
func (q *ReplicaLogQueue) Subscribe(label string) chan *ReplicaLog {
	q.lock.Lock()
	defer q.lock.Unlock()
	ch, ok := q.subscribers[label]
	if ok {
		return ch
	}
	ch = make(chan *ReplicaLog, 10)
	q.subscribers[label] = ch
	return ch
}

// Add adds to the queue
func (q *ReplicaLogQueue) Add(log *ReplicaLog) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.logs = append(q.logs, log)
	q.size = q.size + 1
}

// Flush erases the contents of the queue
func (q *ReplicaLogQueue) Flush() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.logs = make([]*ReplicaLog, 0)
	q.size = 0
}

type ReplicaState struct {
	State     string    `json:"state"`
	Timestamp int64     `json:"timestamp"`
	Replica   ReplicaID `json:"replica"`
}

type ReplicaStateStore struct {
	stateUpdates map[ReplicaID]*ReplicaState
	lock         *sync.Mutex
}

func NewReplicaStateStore() *ReplicaStateStore {
	return &ReplicaStateStore{
		stateUpdates: make(map[ReplicaID]*ReplicaState),
		lock:         new(sync.Mutex),
	}
}
