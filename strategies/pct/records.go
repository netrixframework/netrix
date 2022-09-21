package pct

import (
	"encoding/json"
	"os"
	"path"
	"sync"
)

type record struct {
	changePoints []int
	chains       int
	events       int
	lock         *sync.Mutex
}

type records struct {
	records  map[int]*record
	lock     *sync.Mutex
	filePath string
}

func newRecords(filePath string) *records {
	if _, err := os.Stat(filePath); err != nil {
		os.MkdirAll(filePath, 0750)
	}
	return &records{
		records:  make(map[int]*record),
		lock:     new(sync.Mutex),
		filePath: filePath,
	}
}

func (r *records) NextIteration(i int, changePts []int) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.records[i] = newRecord(changePts)
}

func (r *records) IncrChains(i int) {
	r.lock.Lock()
	defer r.lock.Unlock()
	record, ok := r.records[i]
	if ok {
		record.IncrChains()
	}

}

func (r *records) IncrEvents(i int) {
	r.lock.Lock()
	defer r.lock.Unlock()

	record, ok := r.records[i]
	if ok {
		record.IncrEvents()
	}

}

func (r *records) Summarize() {
	r.lock.Lock()
	defer r.lock.Unlock()

	data := make(map[string]interface{})
	d := make([][]int, len(r.records))
	changePts := make([][]int, len(r.records))
	for i, record := range r.records {
		d[i] = []int{record.Chains(), record.Events()}
		changePts[i] = record.changePoints
	}

	data["data"] = d
	data["changePoints"] = changePts
	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return
	}
	os.WriteFile(path.Join(r.filePath, "data.json"), b, 0644)
}

func newRecord(changePts []int) *record {
	return &record{
		changePoints: changePts,
		chains:       0,
		events:       0,
		lock:         new(sync.Mutex),
	}
}

func (r *record) Chains() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.chains
}

func (r *record) Events() int {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.events
}

func (r *record) IncrChains() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.chains += 1
}

func (r *record) IncrEvents() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.events += 1
}
