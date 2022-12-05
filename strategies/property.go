package strategies

import (
	"sync"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
)

// StrategyWithProperty encapsulates a strategy with a property
// The property is specified as a state machine
type StrategyWithProperty struct {
	Strategy
	Property *sm.StateMachine

	success int
	failed  int
	lock    *sync.Mutex
}

// Creates a new [StrategyWithProperty]
func NewStrategyWithProperty(strategy Strategy, prop *sm.StateMachine) Strategy {
	return &StrategyWithProperty{
		Strategy: strategy,
		Property: prop,
		success:  0,
		failed:   0,
		lock:     new(sync.Mutex),
	}
}

func (s *StrategyWithProperty) Step(e *types.Event, ctx *Context) {
	s.Property.Step(e, ctx.Context)
	if s.Property.CurState().Is(sm.FailStateLabel) {
		s.lock.Lock()
		s.failed += 1
		s.lock.Unlock()
	}
	s.Strategy.Step(e, ctx)
}

func (s *StrategyWithProperty) EndCurIteration(ctx *Context) {
	if s.Property.InSuccessState() {
		s.lock.Lock()
		s.success += 1
		s.lock.Unlock()
	} else {
		s.lock.Lock()
		s.failed += 1
		s.lock.Unlock()
	}
	s.Strategy.EndCurIteration(ctx)
}

func (s *StrategyWithProperty) NextIteration(ctx *Context) {
	s.Property.Reset()
	s.Strategy.NextIteration(ctx)

	s.lock.Lock()
	ctx.Logger.With(log.LogParams{
		"success": s.success,
		"failed":  s.failed,
	}).Info("Current outcomes")
	s.lock.Unlock()
}

func (s *StrategyWithProperty) Finalize(ctx *Context) {
	s.lock.Lock()
	ctx.Logger.With(log.LogParams{
		"success": s.success,
		"failed":  s.failed,
	}).Info("Property outcomes")
	s.lock.Unlock()
	s.Strategy.Finalize(ctx)
}
