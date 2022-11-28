package rl

import (
	"errors"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type RLStrategyConfig struct {
	Alpha       float64
	Gamma       float64
	Interpreter Interpreter
	Policy      Policy
}

type RLStrategy struct {
	*types.BaseService
	config        *RLStrategyConfig
	interpreter   Interpreter
	policy        Policy
	saMap         *StateActionMap
	stateSequence *types.List[State]
	actions       *types.Channel[*strategies.Action]
}

var _ strategies.Strategy = &RLStrategy{}

func NewRLStrategy(config *RLStrategyConfig) (*RLStrategy, error) {
	if config.Alpha < 0 || config.Alpha > 1 {
		return nil, errors.New("invalid value for Alpha")
	} else if config.Gamma < 0 || config.Gamma > 1 {
		return nil, errors.New("invalid value for Gamma")
	}
	return &RLStrategy{
		config:        config,
		saMap:         newStateActionMap(),
		interpreter:   config.Interpreter,
		policy:        config.Policy,
		stateSequence: types.NewEmptyList[State](),
		BaseService:   types.NewBaseService("RLStrategy", nil),
		actions:       types.NewChannel[*strategies.Action](),
	}, nil
}

func (r *RLStrategy) ActionsCh() *types.Channel[*strategies.Action] {
	return r.actions
}

func (r *RLStrategy) Step(e *types.Event, ctx *strategies.Context) {
	state := r.interpreter.Interpret(e, ctx)
	r.stateSequence.Append(state)
	if _, ok := r.saMap.GetState(state.Hash()); !ok {
		r.saMap.AddState(state)
	}
	actions := r.interpreter.Actions(state, ctx)
	for _, a := range actions {
		if !r.saMap.ExistsValue(state.Hash(), a.Name) {
			r.saMap.Update(state.Hash(), a.Name, 0)
		}
	}
	action, ok := r.policy.NextAction(r.saMap, state, actions)
	if ok {
		r.actions.BlockingAdd(action)
	}
}

func (r *RLStrategy) EndCurIteration(ctx *strategies.Context) {
	states := r.stateSequence.Iter()
	actions := ctx.ActionSequence

	for i, state := range states {
		action, ok := actions.Elem(i)
		if !ok {
			continue
		}
		curVal, _ := r.saMap.Get(state.Hash(), action.Name)
		maxQ, _ := r.saMap.MaxQ(state.Hash())
		reward := r.interpreter.RewardFunc(state, action)
		new := (1-r.config.Alpha)*curVal + r.config.Alpha*(reward+r.config.Gamma*maxQ)
		r.saMap.Update(state.Hash(), action.Name, new)
	}
}

func (r *RLStrategy) NextIteration(_ *strategies.Context) {
	r.stateSequence.RemoveAll()
}

func (r *RLStrategy) Finalize(_ *strategies.Context) {

}

func (r *RLStrategy) Start() error {
	r.BaseService.StartRunning()
	return nil
}

func (r *RLStrategy) Stop() error {
	r.BaseService.StopRunning()
	return nil
}
