package strategies

import (
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

type DummyStrategy struct {
	*types.BaseService
}

var _ Strategy = &DummyStrategy{}

func NewDummyStrategy() *DummyStrategy {
	return &DummyStrategy{
		BaseService: types.NewBaseService("dummyStrategy", nil),
	}
}

func (d *DummyStrategy) Start() error {
	d.BaseService.StartRunning()
	return nil
}

func (d *DummyStrategy) Stop() error {
	d.BaseService.StopRunning()
	return nil
}

func (d *DummyStrategy) Step(event *types.Event, c *Context) Action {
	if !event.IsMessageSend() {
		return DoNothing()
	}
	messageID, _ := event.MessageID()
	message, ok := c.Messages.Get(messageID)
	if ok {
		d.Logger.With(log.LogParams{
			"message_id": messageID,
			"from":       message.From,
			"to":         message.To,
		}).Debug("Delivering message")
		return DeliverMessage(message)
	}
	return DoNothing()
}

func (d *DummyStrategy) NextIteration() {

}
