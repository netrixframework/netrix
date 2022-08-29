package strategies

import "github.com/netrixframework/netrix/types"

type Monitor interface {
	Step(*types.Event, *Context)
	NextIteration(*Context)
	Finalize(*Context)
}

type StrategyWithMonitor struct {
	Strategy
	monit Monitor
}

func NewStrategyWithMonitor(strat Strategy, monit Monitor) Strategy {
	return &StrategyWithMonitor{
		Strategy: strat,
		monit:    monit,
	}
}

func (s *StrategyWithMonitor) Step(e *types.Event, c *Context) Action {
	s.monit.Step(e, c)
	return s.Strategy.Step(e, c)
}

func (s *StrategyWithMonitor) NextIteration(c *Context) {
	s.monit.NextIteration(c)
	s.Strategy.NextIteration(c)
}

func (s *StrategyWithMonitor) Finalize(c *Context) {
	s.monit.Finalize(c)
	s.Strategy.Finalize(c)
}
