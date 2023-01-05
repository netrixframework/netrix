package rl

import (
	"math"
	"sync"

	"github.com/netrixframework/netrix/strategies"
	"gonum.org/v1/gonum/stat/sampleuv"
)

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
		val := n.qmap[stateHash][action.Name]
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
