package rl

import (
	"math/rand"
	"time"
)

type RandomPolicy struct {
	rand *rand.Rand
}

var _ Policy = &RandomPolicy{}

func NewRandomPolicy() *RandomPolicy {
	return &RandomPolicy{
		rand: rand.New(rand.NewSource(time.Now().UnixMilli())),
	}
}

func (r *RandomPolicy) NextAction(step int, state State, actions []*Action) (*Action, bool) {
	if len(actions) > 0 {
		i := r.rand.Intn(len(actions))
		return actions[i], true
	}
	return nil, false
}

func (r *RandomPolicy) Update(_ int, _ State, _ *Action, _ State) {

}

func (r *RandomPolicy) NextIteration(_ int, _ *Trace) {

}
