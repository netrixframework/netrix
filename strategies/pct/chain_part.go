package pct

import (
	"sync"

	"github.com/netrixframework/netrix/types"
)

type Chain struct {
	events []*Event
	lock   *sync.Mutex

	enabled           bool
	enabledEventIndex int
	size              int
}

func NewChain(event *Event) *Chain {
	chain := &Chain{
		events:            make([]*Event, 0),
		lock:              new(sync.Mutex),
		enabled:           true,
		enabledEventIndex: 0,
		size:              1,
	}
	chain.events = append(chain.events, event)
	return chain
}

type ChainPartition struct {
	Chains     *types.Map[int, *Chain]
	Partitions []*types.Set[int]
	lock       *sync.Mutex
}

func NewChainPartition() *ChainPartition {
	return &ChainPartition{
		Chains:     types.NewMap[int, *Chain](),
		Partitions: make([]*types.Set[int], 0),
		lock:       new(sync.Mutex),
	}
}
