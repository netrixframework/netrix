package strategies

import (
	"errors"
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

// StrategyConfig store the config used for running strategies
type StrategyConfig struct {
	// Iterations the number of iterations to be run
	Iterations int `json:"iterations"`
	// IterationTimeout timeout for each iteration
	IterationTimeout time.Duration `json:"iteration_timeout"`
	// SetupFunc invoked at the start of every iteration
	SetupFunc func(*Context)
	// StepFunc invoked for every event in all iterations
	StepFunc func(*types.Event, *Context)
	// FinalizeFunc is invoked after all the iterations are completed
	FinalizeFunc func(*Context)
}

type Driver struct {
	strategy   Strategy
	apiserver  *apiserver.APIServer
	dispatcher *dispatcher.Dispatcher
	ctx        *context.RootContext

	eventCh       chan *types.Event
	receiveEvents bool
	newIteration  bool
	config        *StrategyConfig
	strategyCtx   *Context
	lock          *sync.Mutex

	*types.BaseService
}

func NewStrategyDriver(
	config *config.Config,
	mp types.MessageParser,
	strategy Strategy,
	sConfig *StrategyConfig,
) *Driver {
	log.Init(config.LogConfig)
	ctx := context.NewRootContext(config, log.DefaultLogger)
	d := &Driver{
		strategy:      strategy,
		dispatcher:    dispatcher.NewDispatcher(ctx),
		eventCh:       ctx.EventQueue.Subscribe("StrategyDriver"),
		ctx:           ctx,
		config:        sConfig,
		strategyCtx:   nil,
		BaseService:   types.NewBaseService("StrategyDriver", ctx.Logger),
		receiveEvents: false,
		newIteration:  false,
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
	d.ctx.Start()
	go d.poll()
	d.strategy.Start()
	if err := d.main(); err != nil {
		return err
	}
	d.Logger.Info("completed all iterations")
	<-d.QuitCh()
	return nil
}

func (d *Driver) Stop() error {
	d.StopRunning()
	d.apiserver.Stop()
	d.ctx.Stop()
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
			d.lock.Lock()
			newI := d.newIteration
			d.lock.Unlock()
			if newI && event.ID != 0 {
				continue
			} else if newI && event.ID == 0 {
				d.lock.Lock()
				d.newIteration = false
				d.lock.Unlock()
			}
			d.strategyCtx.EventDAG.AddNode(event, []*types.Event{})
			if d.config.StepFunc != nil {
				d.config.StepFunc(event, d.strategyCtx)
			}
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
	if err := d.waitForReplicas(false); err != nil {
		return err
	}
	d.strategyCtx = newContext(d.ctx)
	for d.strategyCtx.CurIteration() < d.config.Iterations {
		if err := d.waitForReplicas(true); err != nil {
			return err
		}
		if d.config.SetupFunc != nil {
			d.config.SetupFunc(d.strategyCtx)
		}
		d.unblockEvents()

		d.Logger.Info(fmt.Sprintf("Starting iteration %d", d.strategyCtx.CurIteration()))
		select {
		case <-d.strategy.QuitCh():
			return nil
		case <-d.QuitCh():
			return nil
		case <-time.After(d.config.IterationTimeout):
		}
		d.blockEvents()
		d.Logger.Debug("Blocked events")

		// d.flushEvents()

		d.lock.Lock()
		d.newIteration = true
		d.lock.Unlock()

		d.strategyCtx.NextIteration()
		if d.strategyCtx.CurIteration() != d.config.Iterations {
			d.strategy.NextIteration(d.strategyCtx)
			d.ctx.Reset()

			d.dispatcher.RestartAll()
		}
		d.ctx.EventIDGen.Reset()
	}
	d.strategy.Finalize(d.strategyCtx)
	if d.config.FinalizeFunc != nil {
		d.config.FinalizeFunc(d.strategyCtx)
	}
	return nil
}

func (d *Driver) flushEvents() {
	for {
		l := len(d.eventCh)
		if l == 0 {
			break
		}
		<-d.eventCh
	}
}

func (d *Driver) waitForReplicas(ready bool) error {
	for {
		select {
		case <-d.QuitCh():
			return errors.New("driver quit")
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
	return nil
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
