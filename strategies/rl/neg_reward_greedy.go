package rl

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"gonum.org/v1/gonum/stat/sampleuv"
)

type NegRewardGreedyPolicy struct {
	// Learning rate
	Alpha float64

	Gamma float64
	// Epsilon-greedy parameter
	Epsilon float64

	qMap map[string]map[string]float64
	lock *sync.Mutex
	rand *rand.Rand
}

func NewNegRewardGreedyProlicy(alpha, gamma, epsilon float64) *NegRewardGreedyPolicy {
	return &NegRewardGreedyPolicy{
		Alpha:   alpha,
		Gamma:   gamma,
		Epsilon: epsilon,
		qMap:    make(map[string]map[string]float64),
		lock:    new(sync.Mutex),
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

var _ Policy = &NegRewardGreedyPolicy{}

func (n *NegRewardGreedyPolicy) NextAction(step int, state State, actions []*Action) (*Action, bool) {
	if len(actions) == 0 {
		return nil, false
	}

	stateHash := state.Hash()
	n.lock.Lock()
	defer n.lock.Unlock()

	if _, ok := n.qMap[stateHash]; !ok {
		n.qMap[stateHash] = make(map[string]float64)
	}

	max := float64(0)
	for _, a := range actions {
		aName := a.Name()
		if _, ok := n.qMap[stateHash][aName]; !ok {
			n.qMap[stateHash][aName] = 0
		} else {
			max = n.qMap[stateHash][aName]
		}
	}

	if n.rand.Float64() > n.Epsilon {
		// Greedy option
		var maxAction *Action = nil
		for _, a := range actions {
			aName := a.Name()
			val := n.qMap[stateHash][aName]
			if val >= max {
				max = val
				maxAction = a
			}
		}
		return maxAction, maxAction != nil
	} else {
		// Softmax option
		sum := float64(0)
		weights := make([]float64, len(actions))
		vals := make([]float64, len(actions))

		for i, action := range actions {
			val := n.qMap[stateHash][action.Name()]
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
}

func (n *NegRewardGreedyPolicy) Update(_ int, _ State, _ *Action, _ State) {

}

func (n *NegRewardGreedyPolicy) NextIteration(iteration int, trace *Trace) {
	n.lock.Lock()
	defer n.lock.Unlock()
	traceLength := trace.Length()
	for i := 0; i < traceLength; i++ {
		state, action, ok := trace.Get(i)
		if !ok {
			continue
		}
		stateHash := state.Hash()
		nextState, _, ok := trace.Get(i + 1)
		if !ok {
			continue
		}
		nextStateHash := nextState.Hash()
		actionKey := action.Name()
		if _, ok := n.qMap[stateHash]; !ok {
			continue
		}
		if _, ok := n.qMap[stateHash][actionKey]; !ok {
			continue
		}
		curVal := n.qMap[stateHash][actionKey]
		max := float64(0)
		if _, ok := n.qMap[nextStateHash]; ok {
			for _, val := range n.qMap[nextStateHash] {
				if val > max {
					max = val
				}
			}
		}
		nextVal := (1-n.Alpha)*curVal + n.Alpha*(-1+n.Gamma*max)
		n.qMap[stateHash][actionKey] = nextVal
	}
}
