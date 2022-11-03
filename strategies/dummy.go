package strategies

import (
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

type DummyStrategy struct {
	*types.BaseService
	Actions *types.Channel[*Action]
}

var _ Strategy = &DummyStrategy{}

func NewDummyStrategy() *DummyStrategy {
	return &DummyStrategy{
		BaseService: types.NewBaseService("dummyStrategy", nil),
		Actions:     types.NewChannel[*Action](),
	}
}

func (d *DummyStrategy) ActionsCh() *types.Channel[*Action] {
	return d.Actions
}

func (d *DummyStrategy) Start() error {
	d.BaseService.StartRunning()
	return nil
}

func (d *DummyStrategy) Stop() error {
	d.BaseService.StopRunning()
	return nil
}

func (d *DummyStrategy) Step(event *types.Event, c *Context) {
	if !event.IsMessageSend() {
		return
	}
	messageID, _ := event.MessageID()
	message, ok := c.MessagePool.Get(messageID)
	if ok {
		d.Logger.With(log.LogParams{
			"message_id": messageID,
			"from":       message.From,
			"to":         message.To,
		}).Debug("Delivering message")
		d.Actions.BlockingAdd(DeliverMessage(message))
	}
}

func (f *DummyStrategy) EndCurIteration(*Context) {

}

func (d *DummyStrategy) NextIteration(*Context) {

}

func (d *DummyStrategy) Finalize(*Context) {

}
