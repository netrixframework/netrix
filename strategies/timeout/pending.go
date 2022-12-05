package timeout

import (
	"fmt"
	"time"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/netrix/util/z3"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

type pendingEvent struct {
	label       string
	replica     types.ReplicaID
	constraints []*z3.AST
	timeout     *types.ReplicaTimeout
	message     *types.Message
	delay       int
	dist        *distuv.Bernoulli
}

func (p *pendingEvent) assignDist() {
	var prob float64
	if p.delay < 10 {
		prob = 1.0
	} else {
		prob = 10.0 / float64(p.delay)
	}
	p.dist = &distuv.Bernoulli{
		P:   prob,
		Src: rand.NewSource(uint64(time.Now().UnixNano())),
	}
}

func (p *pendingEvent) fire() bool {
	return p.dist.Rand() == 1
}

func (t *TimeoutStrategy) updatePendingEvents(e *types.Event, ctx *strategies.Context) {
	var pEvent *pendingEvent = nil
	if e.IsMessageSend() {
		messageID, _ := e.MessageID()
		message, ok := ctx.MessagePool.Get(messageID)
		if !ok {
			// t.Logger.With(log.LogParams{"message_id": messageID}).Debug("no message found!")
			return
		}
		pEvent = &pendingEvent{
			label:       fmt.Sprintf("pe_%d", t.pendingEventCtr.Next()),
			replica:     message.To,
			constraints: make([]*z3.AST, 0),
			message:     message,
			delay:       0,
		}
		sendSymbol, ok := t.symbolMap.Get(fmt.Sprintf("e_%d", e.ID))
		if !ok {
			return
		}
		t.records.updateEvents(ctx, false)
		receiveSymbol := t.z3context.RealConst(pEvent.label)
		t.symbolMap.Add(pEvent.label, receiveSymbol)
		delayVal := t.config.delayValue()
		t.delayVals.Add(string(messageID), delayVal)
		t.records.updateDistVal(ctx, delayVal)
		if t.config.useDistribution() {
			pEvent.constraints = append(
				pEvent.constraints,
				sendSymbol.Mul(t.config.driftMin(t.z3context)).Add(t.z3context.Int(delayVal)).Le(receiveSymbol),
				receiveSymbol.Sub(sendSymbol.Mul(t.config.driftMax(t.z3context))).Eq(t.z3context.Int(delayVal)),
			)
		} else {
			pEvent.constraints = append(
				pEvent.constraints,
				sendSymbol.Mul(t.config.driftMin(t.z3context)).Sub(receiveSymbol).Le(t.z3context.Int(0)),
				receiveSymbol.Sub(sendSymbol.Mul(t.config.driftMax(t.z3context))).Le(t.z3context.Int(delayVal)),
			)
		}
		pEvent.delay = delayVal
		pEvent.assignDist()
	} else if e.IsTimeoutStart() {
		timeout, _ := e.Timeout()
		pEvent = &pendingEvent{
			label:       fmt.Sprintf("pe_%d", t.pendingEventCtr.Next()),
			replica:     timeout.Replica,
			constraints: make([]*z3.AST, 0),
			timeout:     timeout,
		}
		startSymbol, ok := t.symbolMap.Get(fmt.Sprintf("e_%d", e.ID))
		if !ok {
			return
		}
		t.records.updateEvents(ctx, true)
		endSymbol := t.z3context.RealConst(pEvent.label)
		t.symbolMap.Add(pEvent.label, endSymbol)
		pEvent.constraints = append(
			pEvent.constraints,
			endSymbol.Eq(startSymbol.Add(t.z3context.Int(int(timeout.Duration.Milliseconds())))),
		)
		pEvent.delay = int(timeout.Duration.Milliseconds())
		pEvent.assignDist()
	}
	if pEvent != nil {
		t.pendingEvents.Add(pEvent.label, pEvent)
	}
}
