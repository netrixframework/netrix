package timeout

import (
	"sync"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
)

type records struct {
	r    map[int]*record
	lock *sync.Mutex
}

func newRecords() *records {
	r := &records{
		r:    make(map[int]*record),
		lock: new(sync.Mutex),
	}
	r.r[0] = newRecord()
	return r
}

func (r *records) newIteration(spurious bool, i int) {
	r.lock.Lock()
	defer r.lock.Unlock()
	cur, ok := r.r[i-1]
	if ok && spurious {
		cur.SetSpurious()
	}
	_, ok = r.r[i]
	if !ok {
		r.r[i] = newRecord()
	}
}

func (r *records) step(c *strategies.Context) {
	i := c.CurIteration()
	r.lock.Lock()
	record, ok := r.r[i]
	if !ok {
		record = newRecord()
		r.r[i] = record
	}
	r.lock.Unlock()
	record.IncrEvents()
}

func (r *records) summarize(ctx *strategies.Context) {
	sumEvents := 0
	spuriousCount := 0
	r.lock.Lock()
	for _, r := range r.r {
		sumEvents = sumEvents + r.Events()
		if r.IsSpurious() {
			spuriousCount += 1
		}
	}
	total := len(r.r)
	r.lock.Unlock()
	ctx.Logger.With(log.LogParams{
		"total_iterations":    total,
		"spurious_iterations": spuriousCount,
		"avg_events":          float64(sumEvents) / float64(total),
	}).Info("finalizing")
}

type record struct {
	events   int
	spurious bool
	lock     *sync.Mutex
}

func newRecord() *record {
	return &record{
		events:   0,
		spurious: false,
		lock:     new(sync.Mutex),
	}
}

func (r *record) Events() int {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.events
}

func (r *record) IsSpurious() bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.spurious
}

func (r *record) IncrEvents() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.events = r.events + 1
}

func (r *record) SetSpurious() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.spurious = true
}
