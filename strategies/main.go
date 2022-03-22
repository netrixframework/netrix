package strategies

import (
	"errors"

	"github.com/netrixframework/netrix/apiserver"
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

var (
	ErrNoStrategy = errors.New("strategy does not exist")
)

type Strategy interface {
	types.Service
	// Step returns the messages to be delivered to the replicas,
	// Arguments are
	// 1. The new event received from the replicas
	// 2. Context containing a set of possible messages that can be delivered
	Step(*types.Event, []*types.Message) []*types.Message
}

type Filter interface {
	Step(*types.Event) []*types.Message
}

type DefaultFilter struct {
	ctx *context.RootContext
}

func NewDefaultFilter(ctx *context.RootContext) *DefaultFilter {
	return &DefaultFilter{
		ctx: ctx,
	}
}

func (d *DefaultFilter) Step(e *types.Event) []*types.Message {
	if e.IsMessageSend() {
		messageID, _ := e.MessageID()
		message, ok := d.ctx.MessageStore.Get(messageID)
		if ok {
			return []*types.Message{message}
		}
	}
	return []*types.Message{}
}

func GetStrategy(ctx *context.RootContext, s string) (*Driver, error) {
	var strategy Strategy = nil
	switch s {
	case "dummy":
		strategy = NewDummyStrategy(ctx)
	}
	if strategy == nil {
		return nil, ErrNoStrategy
	}
	return NewDriver(ctx, strategy), nil
}

type Driver struct {
	strategy   Strategy
	apiserver  *apiserver.APIServer
	dispatcher *dispatcher.Dispatcher
	eventCh    chan *types.Event
	ctx        *context.RootContext
	filter     Filter
	*types.BaseService
}

func NewDriver(ctx *context.RootContext, strategy Strategy) *Driver {
	ctx.Logger.With(log.LogParams{"strategy": strategy.Name()}).Debug("Creating driver")
	d := &Driver{
		strategy:    strategy,
		dispatcher:  dispatcher.NewDispatcher(ctx),
		eventCh:     ctx.EventQueue.Subscribe("StrategyDriver"),
		ctx:         ctx,
		filter:      NewDefaultFilter(ctx),
		BaseService: types.NewBaseService("StrategyDriver", ctx.Logger),
	}
	d.apiserver = apiserver.NewAPIServer(ctx, nil, d)
	return d
}

func (d *Driver) Start() error {
	d.StartRunning()
	d.apiserver.Start()
	go d.poll()
	d.strategy.Start()
	return nil
}

func (d *Driver) Stop() error {
	d.StopRunning()
	d.apiserver.Stop()
	d.strategy.Stop()
	return nil
}

func (d *Driver) poll() {
	d.Logger.Info("Waiting for all replicas to connect ...")
	for {
		select {
		case <-d.QuitCh():
			return
		default:
		}
		if d.ctx.Replicas.Count() == d.ctx.Config.NumReplicas {
			break
		}
	}
	d.Logger.Info("All replicas connected!")
	for {

		if d.ctx.Replicas.NumReady() != d.ctx.Config.NumReplicas {
			continue
		}

		select {
		case event := <-d.eventCh:
			go d.dispatchMessages(
				d.strategy.Step(
					event,
					d.filter.Step(event),
				),
			)
		case <-d.strategy.QuitCh():
			return
		case <-d.QuitCh():
			return
		}
	}
}

func (d *Driver) dispatchMessages(messages []*types.Message) {
	for _, m := range messages {
		d.dispatcher.DispatchMessage(m)
	}
}
