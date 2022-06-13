package timeout

import (
	"fmt"
	"time"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/util/z3"
)

func (t *TimeoutStrategy) findRandomPendingEvent(ctx *strategies.Context) (*pendingEvent, bool) {
	if !t.config.SpuriousCheck {
		return t.pendingEvents.RandomValue()
	}
	if t.pendingEvents.Size() < t.config.PendingEventThreshold {
		return nil, false
	}
	messages := 0
	timeouts := 0
	solver := t.z3solver
	solver.Push()
	for _, p := range t.pendingEvents.IterValues() {
		if p.timeout != nil {
			timeouts = timeouts + 1
		} else {
			messages = messages + 1
		}
		for _, c := range p.constraints {
			solver.Assert(c)
		}
		latest, ok := ctx.EventDAG.GetLatestNode(p.replica)
		if !ok {
			continue
		}
		latestSymbol, ok := t.symbolMap.Get(fmt.Sprintf("e_%d", latest.ID))
		if !ok {
			continue
		}
		pSymbol, ok := t.symbolMap.Get(p.label)
		if !ok {
			continue
		}
		solver.Assert(latestSymbol.Sub(pSymbol).Le(t.z3context.Int(0)))
	}
	start := time.Now()
	var randomEvent *pendingEvent = nil
	shouldChooseMessage := t.dist.Rand() == 1
	for _, e := range t.pendingEvents.IterValues() {
		if shouldChooseMessage && e.message == nil {
			continue
		}
		solver.Push()
		eSymbol, ok := t.symbolMap.Get(e.label)
		if !ok {
			continue
		}
		for _, p := range t.pendingEvents.IterValues() {
			if p.label != e.label {
				pSymbol, ok := t.symbolMap.Get(p.label)
				if ok {
					solver.Assert(eSymbol.Lt(pSymbol))
				}
			}
		}
		sOk := solver.Check()
		solver.Pop(1)
		if sOk == z3.True {
			randomEvent = e
			break
		}
	}
	solver.Pop(1)
	duration := time.Since(start)
	if randomEvent != nil {
		ch := choice{
			timeouts: timeouts,
			messages: messages,
			time:     duration,
		}
		if randomEvent.timeout != nil {
			ch.choice = 1
		} else {
			ch.choice = 0
		}
		t.records.updateChoice(ctx, ch)
		return randomEvent, true
	}
	return nil, false
}
