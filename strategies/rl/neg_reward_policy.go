package rl

import (
	"math"
	"sync"

	"github.com/netrixframework/netrix/strategies"
	"gonum.org/v1/gonum/stat/sampleuv"
)

type ExplorationPolicyConfig struct {
	Horizon     int
	StateSpace  int
	Iterations  int
	ActionSpace int
	Probability float64
	C           float64
}

type ExplorationPolicy struct {
	config *ExplorationPolicyConfig
	state  *ExplorationState
}

var _ Policy = &ExplorationPolicy{}

func NewExplorationPolicy(config *ExplorationPolicyConfig) *ExplorationPolicy {
	return &ExplorationPolicy{
		config: config,
		state:  NewExplorationState(config),
	}
}

func (e *ExplorationPolicy) NextAction(step int, state State, actions []*strategies.Action) (*strategies.Action, bool) {
	a := e.state.NextAction(step, state, actions)
	return a, a != nil
}

func (e *ExplorationPolicy) Update(step int, state State, action *strategies.Action, nextState State) {
	e.state.Update(step, state, action, nextState)
}

func (e *ExplorationPolicy) NextIteration(iteration int, trace *Trace) {

}

type ExplorationState struct {
	config        *ExplorationPolicyConfig
	eta           float64
	defaultQValue float64

	// Q map of step, state and action
	qValues *qValues
	// N values for step, state and action
	visitCount  *visitCount
	stateValues *stateValues
}

func NewExplorationState(config *ExplorationPolicyConfig) *ExplorationState {
	eta := math.Log(
		float64(config.StateSpace) * float64(config.Horizon) * float64(config.ActionSpace) * float64(config.Iterations) / config.Probability,
	)
	return &ExplorationState{
		config:        config,
		eta:           eta,
		defaultQValue: float64(config.Horizon),
		qValues: &qValues{
			vals:    make(map[int]map[string]map[string]float64),
			horizon: config.Horizon,
			lock:    new(sync.Mutex),
		},
		visitCount: &visitCount{
			vals: make(map[int]map[string]map[string]int),
			lock: new(sync.Mutex),
		},
		stateValues: &stateValues{
			vals:    make(map[int]map[string]float64),
			horizon: config.Horizon,
			lock:    new(sync.Mutex),
		},
	}
}

func (e *ExplorationState) NextAction(step int, state State, actions []*strategies.Action) *strategies.Action {
	return e.qValues.NextAction(step, state, actions)
}

func (e *ExplorationState) Update(step int, curState State, nextAction *strategies.Action, nextState State) bool {
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

type NegativeRewardPolicy struct {
	Alpha float64
	Gamma float64
	qmap  map[string]map[string]float64
	lock  *sync.Mutex
}

func NewNegativeRewardPolicy(alpha, gamma float64) *NegativeRewardPolicy {
	return &NegativeRewardPolicy{
		Alpha: alpha,
		Gamma: gamma,
		qmap:  make(map[string]map[string]float64),
		lock:  new(sync.Mutex),
	}
}

var _ Policy = &NegativeRewardPolicy{}

func (n *NegativeRewardPolicy) NextAction(step int, state State, actions []*strategies.Action) (*strategies.Action, bool) {
	stateHash := state.Hash()

	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.qmap[stateHash]; !ok {
		n.qmap[stateHash] = make(map[string]float64)
	}

	for _, a := range actions {
		if _, ok := n.qmap[stateHash][a.Name]; !ok {
			n.qmap[stateHash][a.Name] = 0
		}
	}

	sum := float64(0)
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))

	for i, action := range actions {
		val, _ := n.qmap[stateHash][action.Name]
		exp := math.Exp(val)
		vals[i] = exp
		sum += exp
	}

	for i, v := range vals {
		weights[i] = v / sum
	}
	i, ok := sampleuv.NewWeighted(weights, nil).Take()
	if !ok {
		return nil, false
	}
	return actions[i], true
}

func (n *NegativeRewardPolicy) Update(_ int, _ State, _ *strategies.Action, _ State) {

}

func (n *NegativeRewardPolicy) NextIteration(iteration int, trace *Trace) {
	n.lock.Lock()
	defer n.lock.Unlock()
	traceLength := trace.Length()
	for i := 0; i < traceLength; i++ {
		state, action, ok := trace.Get(i)
		if !ok {
			continue
		}
		stateHash := state.Hash()
		if _, ok := n.qmap[stateHash]; !ok {
			continue
		}
		if _, ok := n.qmap[stateHash][action.Name]; !ok {
			continue
		}
		curVal := n.qmap[stateHash][action.Name]
		max := float64(0)
		for _, val := range n.qmap[stateHash] {
			if val > max {
				max = val
			}
		}
		nextVal := (1-n.Alpha)*curVal + n.Alpha*(-1+n.Gamma*max)
		n.qmap[stateHash][action.Name] = nextVal
	}
}
