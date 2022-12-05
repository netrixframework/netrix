package timeout

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
)

type distVal struct {
	iteration int
	val       int
}

func saveDistValues(p string, vals []distVal) {
	filePath := path.Join(p, "distvals.json")
	data := make(map[string]interface{})
	valsData := make([]map[string]int, 0)
	for _, val := range vals {
		valsData = append(valsData, map[string]int{
			"iteration": val.iteration,
			"value":     val.val,
		})
	}
	data["values"] = valsData
	dataB, err := json.Marshal(data)
	if err != nil {
		return
	}
	err = os.WriteFile(filePath, dataB, 0644)
	if err != nil {
		println("Error:", err.Error())
	}
}

type records struct {
	r                  map[int]*record
	lock               *sync.Mutex
	choices            map[int]*choices
	distributionValues []distVal
	filePath           string
}

func newRecords(filePath string) *records {
	if _, err := os.Stat(filePath); err != nil {
		os.MkdirAll(filePath, 0750)
	}
	choicesPath := path.Join(filePath, "choices")
	if _, err := os.Stat(choicesPath); err != nil {
		os.MkdirAll(choicesPath, 0750)
	}
	r := &records{
		r:                  make(map[int]*record),
		lock:               new(sync.Mutex),
		choices:            make(map[int]*choices),
		distributionValues: make([]distVal, 0),
		filePath:           filePath,
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
	choices, ok := r.choices[i-1]
	if ok {
		choices.Save(r.filePath)
		delete(r.choices, i-1)
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
	i := ctx.CurIteration()
	choices, ok := r.choices[i-1]
	if ok {
		choices.Save(r.filePath)
	}
	distVals := r.distributionValues
	r.lock.Unlock()
	saveDistValues(r.filePath, distVals)
	ctx.Logger.With(log.LogParams{
		"total_iterations":    total,
		"spurious_iterations": spuriousCount,
		"avg_events":          float64(sumEvents) / float64(total),
	}).Info("finalizing")
}

func (r *records) updateChoice(ctx *strategies.Context, c choice) {
	i := ctx.CurIteration()
	r.lock.Lock()
	defer r.lock.Unlock()
	_, ok := r.choices[i]
	if !ok {
		r.choices[i] = &choices{
			iteration: i,
			choices:   make([]choice, 0),
		}
	}
	r.choices[i].updateChoice(c)
}

func (r *records) updateEvents(ctx *strategies.Context, isTimeout bool) {
	i := ctx.CurIteration()
	r.lock.Lock()
	defer r.lock.Unlock()
	_, ok := r.choices[i]
	if !ok {
		r.choices[i] = &choices{
			iteration: i,
			choices:   make([]choice, 0),
		}
	}
	r.choices[i].addCount(isTimeout)
}

func (r *records) updateDistVal(ctx *strategies.Context, val int) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.distributionValues = append(r.distributionValues, distVal{
		iteration: ctx.CurIteration(),
		val:       val,
	})
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

type choices struct {
	iteration int
	timeouts  int
	messages  int
	choices   []choice
}

func (c *choices) updateChoice(ch choice) {
	c.choices = append(c.choices, ch)
}

func (c *choices) addCount(isTimeout bool) {
	if isTimeout {
		c.timeouts = c.timeouts + 1
	} else {
		c.messages = c.messages + 1
	}
}

func (c *choices) Save(p string) {
	data := make(map[string]interface{})
	summary := make(map[string]interface{})
	summary["timeouts"] = c.timeouts
	summary["messages"] = c.messages
	summary["choices"] = len(c.choices)
	data["summary"] = summary

	choices := make([]map[string]interface{}, 0)

	for _, choice := range c.choices {
		choiceD := make(map[string]interface{})
		choiceD["timeouts"] = choice.timeouts
		choiceD["messages"] = choice.messages
		choiceD["duration"] = choice.time.Nanoseconds()
		if choice.choice == 1 {
			choiceD["choice"] = "timeout"
		} else {
			choiceD["choice"] = "message"
		}
		choices = append(choices, choiceD)
	}
	data["choices"] = choices

	dataB, err := json.Marshal(data)
	if err != nil {
		return
	}
	filePath := path.Join(p, "choices", fmt.Sprintf("%d.json", c.iteration))
	if err := os.WriteFile(filePath, dataB, 0644); err != nil {
		println("Error:", err.Error())
	}
}

type choice struct {
	timeouts int
	messages int
	choice   int
	time     time.Duration
}
