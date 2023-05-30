package rl

import (
	"sync"

	"gonum.org/v1/gonum/stat/distuv"
)

type ThomsonSamplingPolicy struct {
	sMap map[string]map[string]*distuv.Beta
	lock *sync.Mutex
}

var _ Policy = &ThomsonSamplingPolicy{}

func NewThomsonSamplingPolicy() *ThomsonSamplingPolicy {
	return &ThomsonSamplingPolicy{
		sMap: make(map[string]map[string]*distuv.Beta),
		lock: new(sync.Mutex),
	}
}

func (t *ThomsonSamplingPolicy) NextAction(step int, state State, actions []*Action) (*Action, bool) {
	if len(actions) == 0 {
		return nil, false
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	stateKey := state.Hash()
	if _, ok := t.sMap[stateKey]; !ok {
		t.sMap[stateKey] = make(map[string]*distuv.Beta)
	}
	var maxVal float64 = 0
	var maxAction *Action = nil
	for _, a := range actions {
		aName := a.Name()
		if _, ok := t.sMap[stateKey][aName]; !ok {
			t.sMap[stateKey][aName] = &distuv.Beta{Alpha: 1, Beta: 1}
		}
		val := t.sMap[stateKey][aName].Rand()
		if val > maxVal {
			maxVal = val
			maxAction = a
		}
	}
	return maxAction, maxAction != nil
}

func (t *ThomsonSamplingPolicy) Update(step int, curState State, action *Action, nextState State) {
	nextStateKey := nextState.Hash()
	stateKey := curState.Hash()
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, ok := t.sMap[stateKey]; !ok {
		t.sMap[stateKey][action.Name()] = &distuv.Beta{Alpha: 1, Beta: 1}
	}
	if _, ok := t.sMap[stateKey][action.Name()]; !ok {
		t.sMap[stateKey][action.Name()] = &distuv.Beta{Alpha: 1, Beta: 1}
	}

	reward := 0
	if _, ok := t.sMap[nextStateKey]; !ok {
		reward = 1
	}
	curDist := t.sMap[stateKey][action.Name()]
	t.sMap[stateKey][action.Name()] = &distuv.Beta{
		Alpha: curDist.Alpha + float64(reward),
		Beta:  curDist.Beta + float64(1-reward),
	}
}

func (t *ThomsonSamplingPolicy) NextIteration(iteration int, trace *Trace) {

}
