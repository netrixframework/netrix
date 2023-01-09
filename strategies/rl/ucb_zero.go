package rl

import (
	"math"
	"sync"

	"github.com/netrixframework/netrix/strategies"
)

type UCBZeroPolicyConfig struct {
	Horizon     int
	StateSpace  int
	Iterations  int
	ActionSpace int
	Probability float64
	C           float64
}

type UCBZeroPolicy struct {
	config *UCBZeroPolicyConfig
	state  *UCBZeroState
}

var _ Policy = &UCBZeroPolicy{}

func NewUCBZeroPolicy(config *UCBZeroPolicyConfig) *UCBZeroPolicy {
	return &UCBZeroPolicy{
		config: config,
		state:  NewUCBZeroState(config),
	}
}

func (e *UCBZeroPolicy) NextAction(step int, state State, actions []*strategies.Action) (*strategies.Action, bool) {
	a := e.state.NextAction(step, state, actions)
	return a, a != nil
}

func (e *UCBZeroPolicy) Update(step int, state State, action *strategies.Action, nextState State) {
	e.state.Update(step, state, action, nextState)
}

func (e *UCBZeroPolicy) NextIteration(iteration int, trace *Trace) {

}

type UCBZeroState struct {
	config        *UCBZeroPolicyConfig
	eta           float64
	defaultQValue float64

	// Q map of step, state and action
	qValues *qValues
	// N values for step, state and action
	visitCount  *visitCount
	stateValues *stateValues
}

func NewUCBZeroState(config *UCBZeroPolicyConfig) *UCBZeroState {
	eta := math.Log(
		float64(config.StateSpace) * float64(config.Horizon) * float64(config.ActionSpace) * float64(config.Iterations) / config.Probability,
	)
	return &UCBZeroState{
		config:        config,
		eta:           eta,
		defaultQValue: float64(config.Horizon),
		qValues: &qValues{
			vals:          make(map[int]map[string]map[string]float64),
			defaultQValue: float64(config.Horizon),
			lock:          new(sync.Mutex),
		},
		visitCount: &visitCount{
			vals: make(map[int]map[string]map[string]int),
			lock: new(sync.Mutex),
		},
		stateValues: &stateValues{
			vals: make(map[int]map[string]float64),
			lock: new(sync.Mutex),
		},
	}
}

func (e *UCBZeroState) NextAction(step int, state State, actions []*strategies.Action) *strategies.Action {
	return e.qValues.NextAction(step, state, actions)
}

func (e *UCBZeroState) Update(step int, curState State, nextAction *strategies.Action, nextState State) bool {
	curVisits := e.visitCount.Get(step, curState, nextAction)
	e.visitCount.Incr(step, curState, nextAction)

	nextStateValue := e.qValues.MaxValue(step+1, nextState)
	if nextStateValue > float64(e.config.Horizon) {
		nextStateValue = float64(e.config.Horizon)
	}
	new := e.stateValues.Update(step+1, nextState, nextStateValue)

	t := float64(curVisits + 1)
	alphaT := float64(e.config.Horizon+1) / (float64(e.config.Horizon) + t)
	betaT := e.config.C * math.Sqrt(math.Pow(float64(e.config.Horizon), 3)*(e.eta)/t)

	curQValue := e.qValues.Get(step, curState, nextAction)
	nextQValue := (1-alphaT)*float64(curQValue) + alphaT*(nextStateValue+2*betaT)
	e.qValues.Update(step, curState, nextAction, nextQValue)
	return new
}

type qValues struct {
	vals          map[int]map[string]map[string]float64
	defaultQValue float64
	lock          *sync.Mutex
}

func (q *qValues) init(step int, state State, action *strategies.Action) {
	q.lock.Lock()
	defer q.lock.Unlock()
	stateKey := state.Hash()
	_, ok := q.vals[step]
	if !ok {
		q.vals[step] = make(map[string]map[string]float64)
	}

	_, ok = q.vals[step][stateKey]
	if !ok {
		q.vals[step][stateKey] = make(map[string]float64)
	}

	_, ok = q.vals[step][stateKey][action.Name]
	if !ok {
		q.vals[step][stateKey][action.Name] = float64(q.defaultQValue)
	}
}

func (q *qValues) NextAction(step int, state State, actions []*strategies.Action) *strategies.Action {
	q.lock.Lock()
	defer q.lock.Unlock()

	stateKey := state.Hash()

	if _, ok := q.vals[step]; !ok {
		q.vals[step] = make(map[string]map[string]float64)
	}

	if _, ok := q.vals[step][stateKey]; !ok {
		q.vals[step][stateKey] = make(map[string]float64)
	}

	var max float64
	var maxAction string
	actionsMap := make(map[string]*strategies.Action)
	for i, a := range actions {
		if _, ok := q.vals[step][stateKey][a.Name]; !ok {
			q.vals[step][stateKey][a.Name] = float64(q.defaultQValue)
			max = float64(q.defaultQValue)
			maxAction = a.Name
		}
		if i == 0 {
			max = q.vals[step][stateKey][a.Name]
			maxAction = a.Name
		}
		actionsMap[a.Name] = a
	}

	for action, v := range q.vals[step][stateKey] {
		if v > max {
			max = v
			maxAction = action
		}
	}
	return actionsMap[maxAction]
}

func (q *qValues) Get(step int, state State, action *strategies.Action) float64 {
	q.init(step, state, action)
	q.lock.Lock()
	defer q.lock.Unlock()
	return q.vals[step][state.Hash()][action.Name]
}

func (q *qValues) Update(step int, state State, action *strategies.Action, val float64) {
	q.init(step, state, action)
	q.lock.Lock()
	defer q.lock.Unlock()
	q.vals[step][state.Hash()][action.Name] = val
}

func (q *qValues) MaxValue(step int, state State) float64 {
	q.lock.Lock()
	defer q.lock.Unlock()

	stateKey := state.Hash()

	_, ok := q.vals[step]
	if !ok {
		q.vals[step] = make(map[string]map[string]float64)
		q.vals[step][stateKey] = make(map[string]float64)
		return float64(q.defaultQValue)
	}
	_, ok = q.vals[step][stateKey]
	if !ok {
		q.vals[step][stateKey] = make(map[string]float64)
		return float64(q.defaultQValue)
	}

	if len(q.vals[step][stateKey]) == 0 {
		return float64(q.defaultQValue)
	}

	max := float64(math.MinInt)
	for _, v := range q.vals[step][stateKey] {
		if v > max {
			max = v
		}
	}
	return max
}

type visitCount struct {
	vals map[int]map[string]map[string]int
	lock *sync.Mutex
}

func (v *visitCount) Get(step int, state State, action *strategies.Action) int {
	v.lock.Lock()
	defer v.lock.Unlock()

	stateKey := state.Hash()

	if _, ok := v.vals[step]; !ok {
		v.vals[step] = make(map[string]map[string]int)
	}
	if _, ok := v.vals[step][stateKey]; !ok {
		v.vals[step][stateKey] = make(map[string]int)
	}
	val, ok := v.vals[step][stateKey][action.Name]
	if !ok {
		v.vals[step][stateKey][action.Name] = 0
		return 0
	}
	return val
}

func (v *visitCount) Incr(step int, state State, action *strategies.Action) {
	if _, ok := v.vals[step]; !ok {
		v.vals[step] = make(map[string]map[string]int)
	}
	stateKey := state.Hash()
	if _, ok := v.vals[step][stateKey]; !ok {
		v.vals[step][stateKey] = make(map[string]int)
	}
	cur, ok := v.vals[step][stateKey][action.Name]
	if !ok {
		v.vals[step][stateKey][action.Name] = 1
		return
	}
	v.vals[step][stateKey][action.Name] = cur + 1
}

type stateValues struct {
	vals map[int]map[string]float64
	lock *sync.Mutex
}

func (s *stateValues) Update(step int, state State, val float64) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	new := false
	if _, ok := s.vals[step]; !ok {
		s.vals[step] = make(map[string]float64)
		new = true
	}
	if _, ok := s.vals[step][state.Hash()]; !ok {
		new = true
	}
	s.vals[step][state.Hash()] = val
	return new
}

func (s *stateValues) NumStates() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	count := 0
	for _, states := range s.vals {
		count += len(states)
	}
	return count
}
