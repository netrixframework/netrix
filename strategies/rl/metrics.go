package rl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
)

type Metrics struct {
	storePath string

	iteration int
	Trace     *Trace

	interpreterStates map[string]int
	wrapperStates     map[string]int
	interpreterTraces map[string]int
	traces            map[string]int
	lock              *sync.Mutex
}

func NewMetrics(path string) (*Metrics, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a director", path)
	}
	return &Metrics{
		storePath:         path,
		iteration:         0,
		Trace:             NewTrace(),
		traces:            make(map[string]int),
		interpreterStates: make(map[string]int),
		wrapperStates:     make(map[string]int),
		interpreterTraces: make(map[string]int),
		lock:              new(sync.Mutex),
	}, nil
}

func (m *Metrics) Update(step int, state *wrappedState, action *Action) {
	m.Trace.Add(state, action)
	m.lock.Lock()
	defer m.lock.Unlock()
	interpreterStateHash := state.InterpreterState.Hash()
	stateHash := state.Hash()
	if _, ok := m.interpreterStates[interpreterStateHash]; !ok {
		m.interpreterStates[interpreterStateHash] = 0
	}
	if _, ok := m.wrapperStates[stateHash]; !ok {
		m.wrapperStates[stateHash] = 0
	}
	m.interpreterStates[interpreterStateHash] += 1
	m.wrapperStates[stateHash] += 1
}

func (m *Metrics) NextIteration() {
	wrappedTraceHash := m.Trace.Hash()
	traceHash := m.Trace.unwrappedHash()

	m.lock.Lock()
	if _, ok := m.interpreterTraces[traceHash]; !ok {
		m.interpreterTraces[traceHash] = 0
	}
	m.interpreterTraces[traceHash] += 1
	if _, ok := m.traces[wrappedTraceHash]; !ok {
		m.traces[wrappedTraceHash] = 0
	}
	m.traces[wrappedTraceHash] += 1

	interpreterStates := make(map[string]int)
	for k, v := range m.interpreterStates {
		interpreterStates[k] = v
	}
	toStore := &iterationMetrics{
		Iteration:               m.iteration,
		InterpreterStateVisits:  interpreterStates,
		UniqueInterpreterStates: len(m.interpreterStates),
		UniqueWrapperStates:     len(m.wrapperStates),
		UniqueInterpreterTraces: len(m.interpreterTraces),
		UniqueWrappedTraces:     len(m.traces),
		WrappedTrace:            m.Trace.Strings(),
		InterpreterTrace:        m.Trace.unwrappedStrings(),
	}
	m.iteration += 1
	m.lock.Unlock()
	m.Trace.Reset()
	path := path.Join(m.storePath, fmt.Sprintf("%d.json", toStore.Iteration))
	data, err := json.MarshalIndent(toStore, "", "\t")
	if err != nil {
		return
	}
	file, err := os.Create(path)
	if err != nil {
		return
	}
	writer := bufio.NewWriter(file)
	writer.Write(data)
	writer.Flush()
	file.Close()
}

func (m *Metrics) Finalize(ctx *strategies.Context) {
	ctx.Logger.With(log.LogParams{})
}

type iterationMetrics struct {
	Iteration               int
	InterpreterStateVisits  map[string]int
	UniqueInterpreterStates int
	UniqueWrapperStates     int
	UniqueInterpreterTraces int
	UniqueWrappedTraces     int
	WrappedTrace            []string
	InterpreterTrace        []string
}
