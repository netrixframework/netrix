package strategies

import (
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

type Filter interface {
	Step(*types.Event, *Context) []*types.Event
}

type FilteredStrategy struct {
	strategy Strategy
	filter   Filter

	*types.BaseService
}

var _ Strategy = &FilteredStrategy{}

func NewFilteredStrategy(filter Filter, strategy Strategy, logger *log.Logger) *FilteredStrategy {
	return &FilteredStrategy{
		filter:      filter,
		strategy:    strategy,
		BaseService: types.NewBaseService("filtered strategy", logger),
	}
}

func (f *FilteredStrategy) Start() error {
	return nil
}

func (f *FilteredStrategy) Stop() error {
	return nil
}

func (f *FilteredStrategy) Step(e *types.Event, c *Context) Action {
	// TODO: fill this up
	return DoNothing()
}

func (f *FilteredStrategy) NextIteration() {

}

func (f *FilteredStrategy) Finalize() {

}
