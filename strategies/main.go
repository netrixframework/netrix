package strategies

import (
	"errors"

	"github.com/netrixframework/netrix/apiserver"
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies/dummy"
	"github.com/netrixframework/netrix/types"
)

var (
	ErrNoStrategy = errors.New("strategy does not exist")
)

type Strategy interface {
	types.Service
	Step(*types.Event) []*types.Message
}

func GetStrategy(ctx *context.RootContext, s string) (*Driver, error) {
	var strategy Strategy = nil
	switch s {
	case "dummy":
		strategy = dummy.NewDummyStrategy(ctx)
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
	*types.BaseService
}

func NewDriver(ctx *context.RootContext, strategy Strategy) *Driver {
	ctx.Logger.With(log.LogParams{"strategy": strategy.Name()}).Debug("Creating driver")
	d := &Driver{
		strategy:    strategy,
		dispatcher:  dispatcher.NewDispatcher(ctx),
		eventCh:     ctx.EventQueue.Subscribe("StrategyDriver"),
		ctx:         ctx,
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
			messages := d.strategy.Step(event)
			for _, m := range messages {
				d.dispatcher.DispatchMessage(m)
			}
		case <-d.strategy.QuitCh():
			return
		case <-d.QuitCh():
			return
		}
	}
}
