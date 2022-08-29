package pct

import (
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

// Step(*types.Event, *Context) Action
// EndCurIteration(*Context)
// NextIteration(*Context)
// Finalize(*Context)

type PCTStrategy struct {
	*types.BaseService
}

var _ strategies.Strategy = &PCTStrategy{}

func NewPCTStrategy() *PCTStrategy {
	return &PCTStrategy{
		BaseService: types.NewBaseService("pctStrategy", nil),
	}
}

func (p *PCTStrategy) Step(_ *types.Event, _ *strategies.Context) strategies.Action {
	return strategies.DoNothing()
}

func (p *PCTStrategy) EndCurIteration(_ *strategies.Context) {

}

func (p *PCTStrategy) NextIteration(_ *strategies.Context) {

}

func (p *PCTStrategy) Finalize(_ *strategies.Context) {

}

func (p *PCTStrategy) Start() error {
	return nil
}

func (p *PCTStrategy) Stop() error {
	return nil
}
