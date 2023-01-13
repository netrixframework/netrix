package rl

import (
	"errors"
	"math/rand"
	"time"
)

var ErrInvalidEpsilon = errors.New("invalid epsilon value")

type UCBZeroEGreedyPolicyConfig struct {
	*UCBZeroPolicyConfig
	Epsilon float64
}

type UCBZeroEGreedyPolicy struct {
	*UCBZeroPolicy
	Epsilon float64
	rand    *rand.Rand
}

var _ Policy = &UCBZeroEGreedyPolicy{}

func NewUCBZeroEGreedyPolicy(config *UCBZeroEGreedyPolicyConfig) (*UCBZeroEGreedyPolicy, error) {
	if config.Epsilon < 0 || config.Epsilon > 1 {
		return nil, ErrInvalidEpsilon
	}
	return &UCBZeroEGreedyPolicy{
		UCBZeroPolicy: NewUCBZeroPolicy(config.UCBZeroPolicyConfig),
		Epsilon:       config.Epsilon,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

func (u *UCBZeroEGreedyPolicy) NextAction(step int, state State, actions []*Action) (*Action, bool) {
	if u.rand.Float64() > u.Epsilon {
		return u.UCBZeroPolicy.NextAction(step, state, actions)
	}
	i := rand.Intn(len(actions))
	return actions[i], true
}
