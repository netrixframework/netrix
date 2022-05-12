package strategies

import (
	"fmt"
	"sync"
	"time"

	"github.com/netrixframework/netrix/apiserver"
	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

type Driver struct {
	strategy   Strategy
	apiserver  *apiserver.APIServer
	dispatcher *dispatcher.Dispatcher
	ctx        *context.RootContext

	eventCh       chan *types.Event
	receiveEvents bool
	config        *config.StrategyConfig
	strategyCtx   *Context
	lock          *sync.Mutex

	*types.BaseService
}

func NewStrategyDriver(config *config.Config, mp types.MessageParser, strategy Strategy) *Driver {
	log.Init(config.LogConfig)
	ctx := context.NewRootContext(config, log.DefaultLogger)
	d := &Driver{
		strategy:      strategy,
		dispatcher:    dispatcher.NewDispatcher(ctx),
		eventCh:       ctx.EventQueue.Subscribe("StrategyDriver"),
		ctx:           ctx,
		strategyCtx:   newContext(ctx),
		BaseService:   types.NewBaseService("StrategyDriver", ctx.Logger),
		receiveEvents: false,
		lock:          new(sync.Mutex),
	}
	d.apiserver = apiserver.NewAPIServer(ctx, mp, d)
	strategy.SetLogger(log.DefaultLogger.With(log.LogParams{
		"strategy": strategy.Name(),
	}))
	return d
}

func (d *Driver) Start() error {
	d.StartRunning()
	d.apiserver.Start()
	go d.poll()
	d.strategy.Start()
	return d.main()
}

func (d *Driver) Stop() error {
	d.StopRunning()
	d.apiserver.Stop()
	d.strategy.Stop()
	return nil
}

func (d *Driver) poll() {
	d.Logger.Info("Waiting for all replicas to connect ...")
	d.waitForReplicas(false)
	d.Logger.Info("All replicas connected!")
	for {
		if !d.canReceiveEvents() {
			continue
		}

		select {
		case event := <-d.eventCh:
			d.strategyCtx.EventDAG.AddNode(event, []*types.Event{})
			go d.performAction(
				d.strategy.Step(
					event,
					d.strategyCtx,
				),
			)
		case <-d.strategy.QuitCh():
			return
		case <-d.QuitCh():
			return
		}
	}
}

func (d *Driver) performAction(action Action) error {
	if action.Name == doNothingAction {
		return nil
	}
	d.Logger.With(log.LogParams{
		"action": action.Name,
	}).Debug("Performing action")
	return action.Do(d.strategyCtx, d.dispatcher)
}

func (d *Driver) main() error {
	d.waitForReplicas(false)
	for d.strategyCtx.CurIteration() < d.config.Iterations {
		d.waitForReplicas(true)
		d.unblockEvents()

		d.Logger.Info(fmt.Sprintf("Starting iteration %d", d.strategyCtx.CurIteration()))
		select {
		case <-d.strategy.QuitCh():
		case <-d.QuitCh():
			return nil

		case <-time.After(d.config.IterationTimeout):
		}
		d.blockEvents()

		d.strategyCtx.NextIteration()
		d.strategy.NextIteration()

		d.ctx.MessageStore.RemoveAll()
		d.ctx.EventQueue.Flush()

		d.dispatcher.RestartAll()
	}
	return nil
}

func (d *Driver) waitForReplicas(ready bool) {
	for {
		select {
		case <-d.QuitCh():
			return
		default:
		}
		count := d.ctx.Replicas.Count()
		if ready {
			count = d.ctx.Replicas.NumReady()
		}
		if count == d.ctx.Config.NumReplicas {
			break
		}
	}
}

func (d *Driver) blockEvents() {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.receiveEvents = false
}

func (d *Driver) unblockEvents() {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.receiveEvents = true
}

func (d *Driver) canReceiveEvents() bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.receiveEvents
}
