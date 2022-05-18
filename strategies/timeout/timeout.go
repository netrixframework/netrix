package timeout

import (
	"errors"
	"fmt"
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
}

func (c *TimeoutStrategyConfig) delayValue() int {
	if c.MaxMessageDelay <= 0 {
		return c.DelayDistribution.Rand()
	}
	return int(c.MaxMessageDelay.Milliseconds())
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

	z3config  *z3.Config
	z3context *z3.Context
	z3solver  *z3.Solver
	*types.BaseService
}

var _ strategies.Strategy = &TimeoutStrategy{}

func NewTimeoutStrategy(config *TimeoutStrategyConfig) (*TimeoutStrategy, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	z3config := z3.NewConfig()
	z3context := z3.NewContext(z3config)

	return &TimeoutStrategy{
		pendingEvents:   types.NewMap[string, *pendingEvent](),
		symbolMap:       types.NewMap[string, *z3.AST](),
		pendingEventCtr: util.NewCounter(),
		config:          config,
		z3config:        z3config,
		z3context:       z3context,
		z3solver:        z3context.NewSolver(),
		BaseService:     types.NewBaseService("TimeoutStrategy", nil),
	}, nil
}

func (t *TimeoutStrategy) Step(e *types.Event, ctx *strategies.Context) strategies.Action {
	// TODO: fill this up

	if !t.Running() {
		return strategies.DoNothing()
	}

	// 1. Update constraints (add e to the graph and update the set of constraints in the solver)
	if !t.updateConstraints(e, ctx) {
		// Panic
		return strategies.DoNothing()
	}
	// 2. Update pending events
	t.updatePendingEvents(e, ctx)

	// 3. Pick a random event and check if its the least
	if p := t.findRandomPendingEvent(ctx); p != nil {
		// Found a candidate event
		if p.timeout != nil {
			t.Logger.With(log.LogParams{
				"timeout": p.timeout.Key(),
			}).Debug("firing timeout")
		} else if p.message != nil {
			t.Logger.With(log.LogParams{
				"message_id": p.message.ID,
			}).Debug("delivering message")
		}
	}

	if e.IsMessageSend() {
		messageID, _ := e.MessageID()
		message, ok := ctx.Messages.Get(messageID)
		if ok {
			return strategies.DeliverMessage(message)
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
		pSymbol, _ := t.symbolMap.Get(key)
		if pNode.Event.IsMessageSend() && e.IsMessageReceive() {
			// Add constraints for message receive
			constraints = append(
				constraints,
				pSymbol.Mul(t.config.driftMin(t.z3context)).Sub(eSymbol).Le(t.z3context.Int(0)),
				eSymbol.Sub(pSymbol.Mul(t.config.driftMax(t.z3context))).Le(t.z3context.Int(t.config.delayValue())),
			)
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
	for _, c := range constraints {
		t.z3solver.Assert(c)
	}
	return true
}

func (t *TimeoutStrategy) Finalize() {

}

func (t *TimeoutStrategy) NextIteration() {
	t.pendingEvents.RemoveAll()
	t.pendingEventCtr.Reset()
	t.symbolMap.RemoveAll()
	t.z3solver.Close()
	t.z3context.Close()
	t.z3config.Close()
	t.z3config = z3.NewConfig()
	t.z3context = z3.NewContext(t.z3config)
	t.z3solver = t.z3context.NewSolver()
}

func (t *TimeoutStrategy) Start() error {
	t.BaseService.StartRunning()
	return nil
}

func (t *TimeoutStrategy) Stop() error {
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
