package strategies

import (
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/types"
)

type StrategyChain struct {
	ctx        *context.RootContext
	strategies []Strategy
	*types.BaseService
}

var _ Strategy = &StrategyChain{}

func NewStrategyChain(ctx *context.RootContext, strategies ...Strategy) *StrategyChain {
	return &StrategyChain{
		ctx:         ctx,
		strategies:  strategies,
		BaseService: types.NewBaseService("StrategyChain", ctx.Logger),
	}
}

func (c *StrategyChain) AddStrategy(s Strategy) {
	c.strategies = append(c.strategies, s)
}

func (c *StrategyChain) Step(e *types.Event, in []*types.Message) []*types.Message {
	next := in
	for _, s := range c.strategies {
		next = s.Step(e, next)
	}
	return next
}

func (c *StrategyChain) Start() error {
	for _, s := range c.strategies {
		if err := s.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (c *StrategyChain) Stop() error {
	var err error = nil
	for _, s := range c.strategies {
		if e := s.Stop(); e != nil {
			err = e
		}
	}
	return err
}
