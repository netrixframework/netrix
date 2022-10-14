package pct

import (
	"math/rand"
	"sync"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type PCTStrategyConfig struct {
	RandSrc        rand.Source
	MaxEvents      int
	Depth          int
	MessageOrder   MessageOrder
	RecordFilePath string
}

type PCTStrategy struct {
	*types.BaseService

	priorityMap       *types.Map[int, int]
	totalEvents       int
	priorityChangePts []int
	totalChains       int
	config            *PCTStrategyConfig
	rand              *rand.Rand
	mo                MessageOrder
	chainPartition    *ChainPartition
	lock              *sync.Mutex
	records           *records
}

var _ strategies.Strategy = &PCTStrategy{}

// TODO: handle for 0 d
func NewPCTStrategy(config *PCTStrategyConfig) *PCTStrategy {
	var messageOrder MessageOrder = NewDefaultMessageOrder()
	if config.MessageOrder != nil {
		messageOrder = config.MessageOrder
	}
	strategy := &PCTStrategy{
		BaseService:       types.NewBaseService("pctStrategy", nil),
		rand:              rand.New(config.RandSrc),
		priorityMap:       types.NewMap[int, int](),
		totalEvents:       0,
		priorityChangePts: make([]int, config.Depth-1),
		totalChains:       0,
		config:            config,
		mo:                messageOrder,
		chainPartition:    NewChainPartition(messageOrder),
		lock:              new(sync.Mutex),
		records:           newRecords(config.RecordFilePath),
	}
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
	p.mo.Reset()
}

func (p *PCTStrategy) AddMessage(m *Message, ctx *strategies.Context) {
	chainID, new := p.chainPartition.AddEvent(m)
	if new {
		p.records.IncrChains(ctx.CurIteration())
		p.lock.Lock()
		p.totalChains = p.totalChains + 1
		newPriority := p.rand.Intn(p.totalChains) + p.config.Depth
		p.lock.Unlock()
		p.priorityMap.Add(chainID, newPriority)
	}

	p.lock.Lock()
	p.totalEvents = p.totalEvents + 1
	p.records.IncrEvents(ctx.CurIteration())
	for i, changePt := range p.priorityChangePts {
		if p.totalEvents == changePt {
			m.label = i
		}
	}
	p.lock.Unlock()
	p.Logger.With(log.LogParams{
		"message": m.messageID,
		"chainID": chainID,
		"new":     new,
	}).Debug("Added message to chain partition")
}

func (p *PCTStrategy) Schedule() (*Message, bool) {
	highestPriority := 0
	var theEvent *Message = nil
	var eventChainID int
	for _, chainID := range p.chainPartition.EnabledChains() {
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
			eventChainID = chainID
		}
	}
	if theEvent != nil {
		p.chainPartition.MarkScheduled(eventChainID)
		return theEvent, true
	}
	return nil, false
}

func (p *PCTStrategy) Step(e *types.Event, ctx *strategies.Context) strategies.Action {

	if e.IsMessageSend() {
		// PCTCP addNewEvent method
		message, ok := ctx.GetMessage(e)
		if !ok {
			return strategies.DoNothing()
		}
		p.mo.AddSendEvent(message)
		p.AddMessage(NewMessage(message), ctx)
	} else if e.IsMessageReceive() {
		message, ok := ctx.GetMessage(e)
		if ok {
			p.mo.AddRecvEvent(message)
		}
	}

	// PCTCP scheduleEvent method
	theEvent, ok := p.Schedule()
	if ok {
		message, ok := ctx.MessagePool.Get(theEvent.messageID)
		if ok {
			return strategies.DeliverMessage(message)
		}
	}
	return strategies.DoNothing()
}

func (p *PCTStrategy) EndCurIteration(_ *strategies.Context) {

}

func (p *PCTStrategy) NextIteration(ctx *strategies.Context) {
	p.setup()
	p.lock.Lock()
	p.records.NextIteration(ctx.CurIteration(), p.priorityChangePts)
	p.lock.Unlock()
}

func (p *PCTStrategy) Finalize(_ *strategies.Context) {
	p.records.Summarize()
}

func (p *PCTStrategy) Start() error {
	p.BaseService.StartRunning()
	return nil
}

func (p *PCTStrategy) Stop() error {
	p.BaseService.StopRunning()
	return nil
}
