package timeout

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/netrix/util"
	"github.com/netrixframework/netrix/util/z3"
)

var (
	ErrBadConfig = errors.New("one of MaxMessageDelay or DelayDistribution should be specified")
)

type Distribution interface {
	Rand() int
}

type TimeoutStrategyConfig struct {
	// Accepted as a percentage value (0-100)
	ClockDrift int
	// Maximum message delay time
	// One of MaxMessageDelay or DelayDistribution should be specified
	MaxMessageDelay time.Duration
	// DelayDistribution is the distribution from which message delay values will be sampled
	DelayDistribution Distribution
	// Nondeterministic will fire timeouts randomly ignoring the timeout duration
	Nondeterministic bool
	// SpuriousCheck will enable checking for spuriousness when firing non deterministically
	SpuriousCheck bool
}

func (c *TimeoutStrategyConfig) UseDistribution() bool {
	return c.MaxMessageDelay <= 0
}

func (c *TimeoutStrategyConfig) validate() error {
	if c.ClockDrift < 0 || c.ClockDrift >= 100 {
		return ErrBadConfig
	}
	if c.MaxMessageDelay <= 0 && c.DelayDistribution == nil {
		return ErrBadConfig
	}
	return nil
}

func (c *TimeoutStrategyConfig) driftMax(ctx *z3.Context) *z3.AST {
	one := ctx.Real(1, 1)
	return one.Add(ctx.Real(c.ClockDrift, 100)).Div(one.Sub(ctx.Real(c.ClockDrift, 100)))
}

func (c *TimeoutStrategyConfig) driftMin(ctx *z3.Context) *z3.AST {
	one := ctx.Real(1, 1)
	return one.Sub(ctx.Real(c.ClockDrift, 100)).Div(one.Add(ctx.Real(c.ClockDrift, 100)))
}

func FireTimeout(t *types.ReplicaTimeout) strategies.Action {
	return strategies.Action{
		Name: "FireTimeout",
		Do: func(ctx *strategies.Context, d *dispatcher.Dispatcher) error {
			return d.DispatchTimeout(t)
		},
	}
}

type TimeoutStrategy struct {
	pendingEvents   *types.Map[string, *pendingEvent]
	symbolMap       *types.Map[string, *z3.AST]
	pendingEventCtr *util.Counter
	config          *TimeoutStrategyConfig

	normalTimer *timer
	records     *records

	z3config   *z3.Config
	z3context  *z3.Context
	z3solver   *z3.Solver
	solverLock *sync.Mutex
	*types.BaseService
}

var _ strategies.Strategy = &TimeoutStrategy{}

func NewTimeoutStrategy(config *TimeoutStrategyConfig) (*TimeoutStrategy, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	z3config := z3.NewConfig()
	z3context := z3.NewContext(z3config)

	strategy := &TimeoutStrategy{
		pendingEvents:   types.NewMap[string, *pendingEvent](),
		symbolMap:       types.NewMap[string, *z3.AST](),
		pendingEventCtr: util.NewCounter(),
		config:          config,
		records:         newRecords(),
		z3config:        z3config,
		z3context:       z3context,
		z3solver:        z3context.NewSolver(),
		solverLock:      new(sync.Mutex),
		BaseService:     types.NewBaseService("TimeoutStrategy", nil),
	}
	if !config.Nondeterministic {
		strategy.normalTimer = newTimer()
	}
	return strategy, nil
}

func (t *TimeoutStrategy) Step(e *types.Event, ctx *strategies.Context) strategies.Action {
	if !t.Running() {
		return strategies.DoNothing()
	}
	t.records.step(ctx)
	// 1. Update constraints (add e to the graph and update the set of constraints in the solver)
	if !t.updateConstraints(e, ctx) {
		// Panic
		return strategies.DoNothing()
	}
	if !t.config.Nondeterministic {
		actions := make([]strategies.Action, 0)
		if e.IsMessageSend() {
			mID, _ := e.MessageID()
			message, ok := ctx.Messages.Get(mID)
			if ok {
				actions = append(actions, strategies.DeliverMessage(message))
			}
		} else if e.IsTimeoutStart() {
			timeout, _ := e.Timeout()
			t.normalTimer.Add(timeout)
		}
		for _, t := range t.normalTimer.Ready() {
			actions = append(actions, FireTimeout(t))
		}
		if len(actions) == 0 {
			return strategies.DoNothing()
		}
		return strategies.ActionSequence(actions...)
	}

	// 2. Update pending events
	t.updatePendingEvents(e, ctx)

	// 3. Pick a random event and check if its the least
	if p, ok := t.findRandomPendingEvent(ctx); ok {
		// Found a candidate event
		if p.timeout != nil {
			t.Logger.With(log.LogParams{
				"timeout": p.timeout.Key(),
			}).Debug("firing timeout")
			return FireTimeout(p.timeout)
		} else if p.message != nil {
			t.Logger.With(log.LogParams{
				"message_id": p.message.ID,
			}).Debug("delivering message")
			return strategies.DeliverMessage(p.message)
		}
	}
	return strategies.DoNothing()
}

func (t *TimeoutStrategy) updateConstraints(e *types.Event, ctx *strategies.Context) bool {
	eNode, ok := ctx.EventDAG.GetNode(e.ID)
	if !ok {
		return false
	}
	key := fmt.Sprintf("e_%d", e.ID)
	eSymbol := t.z3context.RealConst(key)
	t.symbolMap.Add(key, eSymbol)
	constraints := make([]*z3.AST, 0)
	for _, pID := range eNode.Parents.Iter() {
		pNode, ok := ctx.EventDAG.GetNode(pID)
		if !ok {
			// Need to panic if we can't find a parent node
			continue
		}
		key := fmt.Sprintf("e_%d", pID)
		pSymbol, ok := t.symbolMap.Get(key)
		if !ok {
			continue
		}
		if pNode.Event.IsMessageSend() && e.IsMessageReceive() {
			// Add constraints for message receive
			constraints = append(
				constraints,
				pSymbol.Mul(t.config.driftMin(t.z3context)).Sub(eSymbol).Le(t.z3context.Int(0)),
			)
			if t.config.UseDistribution() {
				delayVal := t.config.DelayDistribution.Rand()
				constraints = append(
					constraints,
					eSymbol.Sub(pSymbol.Mul(t.config.driftMax(t.z3context))).Eq(t.z3context.Int(delayVal)),
				)
			} else {
				delayVal := int(t.config.MaxMessageDelay.Milliseconds())
				constraints = append(
					constraints,
					eSymbol.Sub(pSymbol.Mul(t.config.driftMax(t.z3context))).Le(t.z3context.Int(delayVal)),
				)
			}

		} else if pNode.Event.IsTimeoutStart() && e.IsTimeoutEnd() {
			// Add constraints for timeout end
			timeout, _ := e.Timeout()
			constraints = append(
				constraints,
				eSymbol.Eq(pSymbol.Add(t.z3context.Int(int(timeout.Duration.Milliseconds())))),
			)
		} else {
			// Add normal constraints
			constraints = append(
				constraints,
				pSymbol.Sub(eSymbol).Le(t.z3context.Int(0)),
			)
		}
	}
	t.solverLock.Lock()
	for _, c := range constraints {
		t.z3solver.Assert(c)
	}
	t.solverLock.Unlock()
	return true
}

func (t *TimeoutStrategy) Finalize(c *strategies.Context) {
	t.records.summarize(c)
}

func (t *TimeoutStrategy) NextIteration(ctx *strategies.Context) {

	// Record metrics about current iteration
	if !t.config.Nondeterministic {
		t.normalTimer.EndAll()
	}
	t.pendingEvents.RemoveAll()
	t.pendingEventCtr.Reset()
	t.symbolMap.RemoveAll()

	t.solverLock.Lock()
	if t.z3solver != nil {
		spurious := false
		if t.z3solver.Check() == z3.False {
			spurious = true
		}
		t.records.newIteration(spurious, ctx.CurIteration())
		t.z3solver.Close()
	}
	t.z3solver = t.z3context.NewSolver()
	t.solverLock.Unlock()
}

func (t *TimeoutStrategy) Start() error {
	t.BaseService.StartRunning()
	return nil
}

func (t *TimeoutStrategy) Stop() error {

	if !t.config.Nondeterministic {
		t.normalTimer.EndAll()
	}

	if t.z3solver != nil {
		t.z3solver.Close()
	}
	if t.z3context != nil {
		t.z3context.Close()
	}
	if t.z3config != nil {
		t.z3config.Close()
	}
	t.BaseService.StopRunning()
	return nil
}
