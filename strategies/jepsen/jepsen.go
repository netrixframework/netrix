package jepsen

import (
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type JepsenStrategy struct {
	*types.BaseService
	Actions *types.Channel[*strategies.Action]
}

type Generator interface {
	Start(*strategies.Context) error
	Done() chan struct{}
	Stop(*strategies.Context) error
	Actions() *types.Channel[*strategies.Action]
	Reset(*strategies.Context) error
}

type Nemesis interface {
	Start(*strategies.Context) error
	Setup(*strategies.Context) error
	Teardown(*strategies.Context) error
}
