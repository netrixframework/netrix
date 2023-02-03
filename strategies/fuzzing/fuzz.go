package fuzzing

import (
	"math/rand"
	"sync"
	"time"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type FuzzStrategyConfig struct {
	Mutator      Mutator
	Interpreter  Interpreter
	TickDuration time.Duration
	Steps        int
	Seed         rand.Source
}

// Coverage guided fuzzing strategy
// TODO: goals
// 1. Allow plugging in different mutation strategies
// 2. Allow measureing state coverage using an interpreter
// 3. Maintain corpus of successful mutations

// Main idea - the input is a sequence of actions.
// An action can be one of - Deliver to Process p, Drop to process p
// Generate a random input and use quick mutations to improve coverage
type FuzzStrategy struct {
	*types.BaseService
	actions *types.Channel[*strategies.Action]

	mailBoxes    *types.Map[types.ReplicaID, []*types.Message]
	corpus       *types.Map[string, *Input]
	uniqueStates *types.Map[string, State]

	trace     *Trace
	curInput  *Input
	iteration int
	lock      *sync.Mutex

	mutator     Mutator
	interpreter Interpreter
	config      *FuzzStrategyConfig
}

var _ strategies.Strategy = &FuzzStrategy{}

func NewFuzzStrategy(config *FuzzStrategyConfig) *FuzzStrategy {
	return &FuzzStrategy{
		BaseService: types.NewBaseService("FuzzStrategy", nil),
		actions:     types.NewChannel[*strategies.Action](),

		mailBoxes:    types.NewMap[types.ReplicaID, []*types.Message](),
		corpus:       types.NewMap[string, *Input](),
		uniqueStates: types.NewMap[string, State](),
		mutator:      config.Mutator,
		interpreter:  config.Interpreter,
		config:       config,
		trace:        NewTrace(),
		iteration:    0,
		lock:         new(sync.Mutex),
	}
}

func (f *FuzzStrategy) ActionsCh() *types.Channel[*strategies.Action] {
	return f.actions
}

func (f *FuzzStrategy) EndCurIteration(ctx *strategies.Context) {

}

func (f *FuzzStrategy) NextIteration(ctx *strategies.Context) {

}

func (f *FuzzStrategy) Finalize(ctx *strategies.Context) {

}

func (f *FuzzStrategy) Start() error {
	f.BaseService.StartRunning()
	return nil
}

func (f *FuzzStrategy) Stop() error {
	f.BaseService.StopRunning()
	return nil
}

func (f *FuzzStrategy) Step(e *types.Event, ctx *strategies.Context) {

}
