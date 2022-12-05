package pct

import (
	"sync"

	"github.com/netrixframework/netrix/types"
)

type chain struct {
	ID     int
	events []*pctMessage
	lock   *sync.Mutex

	enabled           bool
	enabledEventIndex int
	size              int
}

func newChain(id int, event *pctMessage) *chain {
	chain := &chain{
		ID:                id,
		events:            make([]*pctMessage, 0),
		lock:              new(sync.Mutex),
		enabled:           true,
		enabledEventIndex: 0,
		size:              1,
	}
	chain.events = append(chain.events, event)
	return chain
}

func (c *chain) LastEvent() *pctMessage {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.events[c.size-1]
}

func (c *chain) AddEvent(e *pctMessage) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.events = append(c.events, e)
	c.size = c.size + 1
	if !c.enabled {
		c.enabled = true
		c.enabledEventIndex = c.size - 1
	}
}

func (c *chain) IsEnabled() bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.enabled
}

func (c *chain) EnabledEvent() (*pctMessage, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.enabled {
		return nil, false
	}
	return c.events[c.enabledEventIndex], true
}

func (c *chain) IncrEnabledEvent() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.enabledEventIndex+1 == c.size {
		c.enabled = false
		c.enabledEventIndex = c.enabledEventIndex + 1
	} else {
		c.enabledEventIndex = c.enabledEventIndex + 1
	}
}

type chainPartition struct {
	Chains     *types.Map[int, *chain]
	Partitions []*types.Set[int]
	mo         MessageOrder
	lock       *sync.Mutex
}

func newChainPartition(mo MessageOrder) *chainPartition {
	return &chainPartition{
		Chains:     types.NewMap[int, *chain](),
		Partitions: make([]*types.Set[int], 0),
		lock:       new(sync.Mutex),
		mo:         mo,
	}
}

// AddEvent adds the event to the chain partition
// Returns the chain ID and a boolean indicating if the chain is newly created
func (p *chainPartition) AddEvent(e *pctMessage) (int, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	newChainCreated := false
	addedChainID := -1
	addedPartition := -1

	for i, partition := range p.Partitions {
		var compatibleChain *chain = nil
		for _, chainID := range partition.Iter() {
			chain, _ := p.Chains.Get(chainID)
			last := chain.LastEvent()
			if last.Lt(p.mo, e) {
				compatibleChain = chain
				break
			}
		}

		if compatibleChain != nil {
			compatibleChain.AddEvent(e)
			addedChainID = compatibleChain.ID

			addedPartition = i
			break
		} else if partition.Size() < i+1 {
			newChain := newChain(p.Chains.Size(), e)
			p.Chains.Add(newChain.ID, newChain)
			partition.Add(newChain.ID)
			addedChainID = newChain.ID
			newChainCreated = true

			addedPartition = i
			break
		}
	}

	if addedChainID == -1 {
		newPartition := types.NewSet[int]()
		p.Partitions = append(p.Partitions, newPartition)

		newChain := newChain(p.Chains.Size(), e)
		p.Chains.Add(newChain.ID, newChain)
		newPartition.Add(newChain.ID)

		addedPartition = len(p.Partitions) - 1
		addedChainID = newChain.ID
		newChainCreated = true
	}

	if addedPartition > 0 {
		curPartition := p.Partitions[addedPartition]
		curPartition.Remove(addedChainID)

		precedingPartition := p.Partitions[addedPartition-1]
		precedingPartition.Add(addedChainID)

		p.Partitions[addedPartition-1], p.Partitions[addedPartition] = curPartition, precedingPartition
	}

	return addedChainID, newChainCreated
}

func (p *chainPartition) EnabledChains() []int {
	result := make([]int, 0)
	for chainID, chain := range p.Chains.ToMap() {
		if chain.IsEnabled() {
			result = append(result, chainID)
		}
	}

	return result
}

func (p *chainPartition) GetEnabledEvent(chainID int) (*pctMessage, bool) {
	chain, ok := p.Chains.Get(chainID)
	if !ok {
		return nil, false
	}

	return chain.EnabledEvent()
}

func (p *chainPartition) MarkScheduled(chainID int) {
	chain, ok := p.Chains.Get(chainID)
	if ok {
		chain.IncrEnabledEvent()
	}
}

func (p *chainPartition) Reset() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.Chains.RemoveAll()
	p.Partitions = make([]*types.Set[int], 0)
}
