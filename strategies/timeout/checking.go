package timeout

import (
	"fmt"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/util/z3"
)

func (t *TimeoutStrategy) findRandomPendingEvent(ctx *strategies.Context) *pendingEvent {
	solver := t.z3solver
	solver.Push()
	for _, p := range t.pendingEvents.IterValues() {
		for _, c := range p.constraints {
			solver.Assert(c)
		}
		latest, ok := ctx.EventDAG.GetLatestNode(p.replica)
		if !ok {
			continue
		}
		latestSymbol, _ := t.symbolMap.Get(fmt.Sprintf("e_%d", latest.ID))
		pSymbol, _ := t.symbolMap.Get(p.label)
		solver.Assert(latestSymbol.Sub(pSymbol).Le(t.z3context.Int(0)))
	}

	var randomEvent *pendingEvent = nil
	for _, e := range t.pendingEvents.IterValues() {
		solver.Push()
		eSymbol, _ := t.symbolMap.Get(e.label)
		for _, p := range t.pendingEvents.IterValues() {
			if p.label != e.label {
				pSymbol, _ := t.symbolMap.Get(p.label)
				solver.Assert(eSymbol.Lt(pSymbol))
			}
		}
		ok := solver.Check()
		solver.Pop(1)
		if ok == z3.True {
			randomEvent = e
			break
		}
	}
	solver.Pop(1)
	return randomEvent
}
