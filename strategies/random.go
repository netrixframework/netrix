package strategies

import (
	"time"

	"github.com/netrixframework/netrix/types"
)

type RandomStrategy struct {
	*types.BaseService
	pendingMessages *types.Map[types.MessageID, *types.Message]
	actions         *types.Channel[*Action]
	tickDuration    time.Duration
}

func NewRandomStrategy(dur time.Duration) *RandomStrategy {
	return &RandomStrategy{
		BaseService:     types.NewBaseService("RandomStrategy", nil),
		pendingMessages: types.NewMap[types.MessageID, *types.Message](),
		actions:         types.NewChannel[*Action](),
		tickDuration:    dur,
	}
}

var _ Strategy = &RandomStrategy{}

func (r *RandomStrategy) ActionsCh() *types.Channel[*Action] {
	return r.actions
}

func (r *RandomStrategy) EndCurIteration(*Context) {

}

func (r *RandomStrategy) NextIteration(*Context) {
	r.pendingMessages.RemoveAll()
}

func (r *RandomStrategy) Finalize(*Context) {

}

func (r *RandomStrategy) Start() error {
	r.BaseService.StartRunning()
	go r.deliveryLoop()
	return nil
}

func (r *RandomStrategy) Stop() error {
	r.BaseService.StopRunning()
	return nil
}

func (r *RandomStrategy) Step(e *types.Event, ctx *Context) {
	if e.IsMessageSend() {
		message, ok := ctx.GetMessage(e)
		if ok {
			r.pendingMessages.Add(message.ID, message)
		}
	}
}
func (r *RandomStrategy) deliveryLoop() {
	ticker := time.NewTicker(r.tickDuration)
	defer ticker.Stop()
	for {
		select {
		case <-r.QuitCh():
			return
		case <-ticker.C:
		}

		message, ok := r.pendingMessages.RandomValue()
		if ok {
			r.actions.BlockingAdd(DeliverMessage(message))
			r.pendingMessages.Remove(message.ID)
		}
	}
}
