package pct

import (
	"math/rand"
	"sync"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type PCTStrategyConfig struct {
	RandSrc   rand.Source
	MaxEvents int
	Depth     int
}

type PCTStrategy struct {
	*types.BaseService

	priorityMap       *types.Map[int, int]
	totalEvents       int
	priorityChangePts []int
	totalChains       int
	config            *PCTStrategyConfig
	rand              *rand.Rand
	chainPartition    *ChainPartition
	lock              *sync.Mutex
}

var _ strategies.Strategy = &PCTStrategy{}

func NewPCTStrategy(config *PCTStrategyConfig) *PCTStrategy {
	strategy := &PCTStrategy{
		BaseService:       types.NewBaseService("pctStrategy", nil),
		rand:              rand.New(config.RandSrc),
		priorityMap:       types.NewMap[int, int](),
		totalEvents:       0,
		priorityChangePts: make([]int, config.Depth-1),
		totalChains:       0,
		config:            config,
		chainPartition:    NewChainPartition(),
		lock:              new(sync.Mutex),
	}
	strategy.setup()
	return strategy
}

func (p *PCTStrategy) setup() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.rand = rand.New(p.config.RandSrc)
	p.priorityMap.RemoveAll()
	p.totalEvents = 0
	p.totalChains = 0
	p.priorityChangePts = make([]int, p.config.Depth-1)

	changePoints := make(map[int]bool)
	changePointsCount := 0
	for changePointsCount < p.config.Depth-1 {
		next := p.rand.Intn(p.config.MaxEvents)
		if _, ok := changePoints[next]; !ok {
			p.priorityChangePts[changePointsCount] = next
			changePointsCount = changePointsCount + 1
		}
	}
	p.chainPartition.Reset()
}

func (p *PCTStrategy) Step(e *types.Event, ctx *strategies.Context) strategies.Action {

	// TODO: Need to update the event partial order when message send and receive events are observed.

	if e.IsMessageSend() {
		// PCTCP addNewEvent method
		messageID, _ := e.MessageID()
		message, ok := ctx.Messages.Get(messageID)
		if !ok {
			return strategies.DoNothing()
		}
		pctEvent := NewEvent(message)

		chainID, new := p.chainPartition.AddEvent(pctEvent)
		if new {
			p.lock.Lock()
			newPriority := p.rand.Intn(p.totalChains) + p.config.Depth
			p.priorityMap.Add(chainID, newPriority)
			p.totalChains = p.totalChains + 1
			p.lock.Unlock()
		}

		p.lock.Lock()
		p.totalEvents = p.totalEvents + 1
		for i, changePt := range p.priorityChangePts {
			if p.totalEvents == changePt {
				pctEvent.label = i
			}
		}
		p.lock.Unlock()
	}

	// PCTCP scheduleEvent method
	enabledChains := p.chainPartition.EnabledChains()
	highestPriority := 0
	var theEvent *Event = nil
	for _, chainID := range enabledChains {
		priority, ok := p.priorityMap.Get(chainID)
		if !ok {
			continue
		}
		enabledEvent, _ := p.chainPartition.GetEnabledEvent(chainID)
		if enabledEvent.HasLabel() && enabledEvent.Label() != priority {
			newPriority := enabledEvent.Label()
			p.priorityMap.Add(chainID, newPriority)
			priority = newPriority
		}

		if priority > highestPriority {
			highestPriority = priority
			theEvent = enabledEvent
		}
	}

	if theEvent != nil {
		message, ok := ctx.Messages.Get(theEvent.messageID)
		if ok {
			return strategies.DeliverMessage(message)
		}
	}
	return strategies.DoNothing()
}

func (p *PCTStrategy) EndCurIteration(_ *strategies.Context) {
}

func (p *PCTStrategy) NextIteration(_ *strategies.Context) {
	p.setup()
}

func (p *PCTStrategy) Finalize(_ *strategies.Context) {
}

func (p *PCTStrategy) Start() error {
	p.BaseService.StartRunning()
	return nil
}

func (p *PCTStrategy) Stop() error {
	p.BaseService.StopRunning()
	return nil
}
