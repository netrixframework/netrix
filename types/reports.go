package types

import (
	"sync"
)

type reportLog struct {
	ID     uint64            `json:"id"`
	KeyVal map[string]string `json:"logs"`
}

func (r *reportLog) match(keyvals map[string]string) bool {
	for k, v := range keyvals {
		cV, ok := r.KeyVal[k]
		if ok && cV != v {
			return false
		}
	}
	return true
}

type ReportStore struct {
	logs []*reportLog
	size uint64
	lock *sync.Mutex
}

func NewReportStore() *ReportStore {
	return &ReportStore{
		logs: make([]*reportLog, 0),
		size: 0,
		lock: new(sync.Mutex),
	}
}

func (r *ReportStore) Log(keyvals map[string]string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.logs = append(r.logs, &reportLog{
		ID:     r.size,
		KeyVal: keyvals,
	})
	r.size += 1
}

func (r *ReportStore) GetLogs(keyvals map[string]string, count int, from int) []*reportLog {
	if count == -1 {
		count = 100
	}
	if from == -1 {
		from = 0
	}

	if len(keyvals) == 0 {
		r.lock.Lock()
		defer r.lock.Unlock()
		return r.logs[from : from+count]
	}
	return r.filter(keyvals, from, count)
}

func (r *ReportStore) filter(keyvals map[string]string, from int, count int) []*reportLog {
	resultSize := 0
	result := make([]*reportLog, 0)

	r.lock.Lock()
	for i := from; i < len(r.logs); i++ {
		if resultSize == count {
			break
		}
		if r.logs[i].match(keyvals) {
			result = append(result, r.logs[i])
			resultSize += 1
		}
	}
	r.lock.Unlock()
	return result
}
