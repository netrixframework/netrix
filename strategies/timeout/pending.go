package timeout

import (
	"fmt"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/netrix/util/z3"
)

type pendingEvent struct {
	label       string
	replica     types.ReplicaID
	constraints []*z3.AST
	timeout     *types.ReplicaTimeout
	message     *types.Message
}

func (t *TimeoutStrategy) updatePendingEvents(e *types.Event, ctx *strategies.Context) {
	var pEvent *pendingEvent = nil
	if e.IsMessageSend() {
		messageID, _ := e.MessageID()
		message, ok := ctx.Messages.Get(messageID)
		if !ok {
			t.Logger.With(log.LogParams{"message_id": messageID}).Info("no message found!")
			return
		}
		pEvent = &pendingEvent{
			label:       fmt.Sprintf("pe_%d", t.pendingEventCtr.Next()),
			replica:     message.To,
			constraints: make([]*z3.AST, 0),
			message:     message,
		}
		sendSymbol, _ := t.symbolMap.Get(fmt.Sprintf("e_%d", e.ID))
		receiveSymbol := t.z3context.RealConst(pEvent.label)
		t.symbolMap.Add(pEvent.label, receiveSymbol)
		pEvent.constraints = append(
			pEvent.constraints,
			sendSymbol.Mul(t.config.driftMin(t.z3context)).Sub(receiveSymbol).Le(t.z3context.Int(0)),
		)
		if t.config.UseDistribution() {
			delayVal := t.config.DelayDistribution.Rand()
			pEvent.constraints = append(
				pEvent.constraints,
				receiveSymbol.Sub(sendSymbol.Mul(t.config.driftMax(t.z3context))).Eq(t.z3context.Int(delayVal)),
			)
		} else {
			delayVal := int(t.config.MaxMessageDelay.Milliseconds())
			pEvent.constraints = append(
				pEvent.constraints,
				receiveSymbol.Sub(sendSymbol.Mul(t.config.driftMax(t.z3context))).Le(t.z3context.Int(delayVal)),
			)
		}
	} else if e.IsTimeoutStart() {
		timeout, _ := e.Timeout()
		pEvent = &pendingEvent{
			label:       fmt.Sprintf("pe_%d", t.pendingEventCtr.Next()),
			replica:     timeout.Replica,
			constraints: make([]*z3.AST, 0),
			timeout:     timeout,
		}
		startSymbol, _ := t.symbolMap.Get(fmt.Sprintf("e_%d", e.ID))
		endSymbol := t.z3context.RealConst(pEvent.label)
		t.symbolMap.Add(pEvent.label, endSymbol)
		pEvent.constraints = append(
			pEvent.constraints,
			endSymbol.Eq(startSymbol.Add(t.z3context.Int(int(timeout.Duration.Milliseconds())))),
		)
	}
	if pEvent != nil {
		t.pendingEvents.Add(pEvent.label, pEvent)
	}
}
