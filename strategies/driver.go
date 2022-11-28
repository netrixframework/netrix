package strategies

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/netrixframework/netrix/apiserver"
	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

var (
	errStrategyQuit = errors.New("quit signalled")
	errDriverStop   = errors.New("strategy driver signalled stop")
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
	strategy  Strategy
	apiserver *apiserver.APIServer
	ctx       *context.RootContext

	eventQueue    *types.Queue[*types.Event]
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
		eventQueue:    ctx.EventQueue,
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
	go d.pollEvents()
	go d.pollActions()
	d.strategy.Start()
	if err := d.main(); err != nil {
		d.Logger.With(log.LogParams{"error": err}).Debug("main loop exited")
		if err != errDriverStop && err != errStrategyQuit {
			return err
		}
		return nil
	}
	d.Logger.Info("Completed all iterations")
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

func (d *Driver) pollEvents() {
	d.Logger.Info("Waiting for all replicas to connect ...")
	d.waitForReplicas(false)
	d.Logger.Info("All replicas connected!")

EventLoop:
	for {

		select {
		case <-d.strategy.QuitCh():
			return
		case <-d.QuitCh():
			return
		default:
		}
		if !d.canReceiveEvents() {
			continue EventLoop
		}

		event, ok := d.eventQueue.Pop()
		if !ok {
			continue EventLoop
		}

		d.strategyCtx.EventDAG.AddNode(event, []*types.Event{})
		if d.config.StepFunc != nil {
			d.config.StepFunc(event, d.strategyCtx)
		}
		d.Logger.Debug("Stepping")
		d.strategy.Step(
			event,
			d.strategyCtx,
		)
	}
}

func (d *Driver) pollActions() {
	d.waitForReplicas(false)

ActionLoop:
	for {
		select {
		case <-d.strategy.QuitCh():
			return
		case <-d.QuitCh():
			return
		default:
		}

		if !d.canReceiveEvents() {
			continue ActionLoop
		}

		select {
		case <-d.strategy.QuitCh():
			return
		case <-d.QuitCh():
			return
		case action := <-d.strategy.ActionsCh().Ch():
			if action != nil {
				d.performAction(action)
			}
		}
	}
}

func (d *Driver) performAction(action *Action) {
	if action.Name == doNothingAction {
		return
	}
	d.strategyCtx.ActionSequence.Append(action)
	action.Do(d.strategyCtx, d.apiserver)
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
		d.strategy.NextIteration(d.strategyCtx)
		d.unblockEvents()

		d.Logger.Info(fmt.Sprintf("Starting iteration %d", d.strategyCtx.CurIteration()))
		select {
		case <-d.strategy.QuitCh():
			d.strategy.Finalize(d.strategyCtx)
			if d.config.FinalizeFunc != nil {
				d.config.FinalizeFunc(d.strategyCtx)
			}
			return errStrategyQuit
		case <-d.QuitCh():
			d.strategy.Finalize(d.strategyCtx)
			if d.config.FinalizeFunc != nil {
				d.config.FinalizeFunc(d.strategyCtx)
			}
			return errDriverStop
		case <-d.strategyCtx.AbortCh():
		case <-time.After(d.config.IterationTimeout):
		}
		d.blockEvents()
		d.Logger.Debug("Blocked events")

		d.apiserver.RestartAll()
		d.apiserver.ForgetSentMessages()
		d.ctx.EventIDGen.Reset()

		d.ctx.PauseQueues()
		// Waiting for some time for the messages in transit
		// time.Sleep(100 * time.Millisecond)

		// d.lock.Lock()
		// d.newIteration = true
		// d.lock.Unlock()

		d.strategy.EndCurIteration(d.strategyCtx)
		d.strategy.ActionsCh().Reset()
		d.strategyCtx.NextIteration()
		d.ctx.Reset()
		d.ctx.ResumeQueues()

	}
	d.strategy.Finalize(d.strategyCtx)
	if d.config.FinalizeFunc != nil {
		d.config.FinalizeFunc(d.strategyCtx)
	}
	return nil
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
