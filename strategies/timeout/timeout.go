package timeout

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/netrix/util"
	"github.com/netrixframework/netrix/util/z3"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
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
	// Z3Timeout duration for z3 spurious checking
	Z3Timeout time.Duration
	// RecordFilePath stores the data collected to this path
	RecordFilePath string
	// PendingEventThreshold dictates the minimum number of pending events required to choose one
	PendingEventThreshold int
	// MessageBias probability bias towards messages vs timeouts (between 0 and 1 where 1 is more messages vs timeouts)
	MessageBias float64
}

func (c *TimeoutStrategyConfig) UseDistribution() bool {
	return c.DelayDistribution != nil && c.Nondeterministic
}

func (c *TimeoutStrategyConfig) validate() error {
	if c.ClockDrift < 0 || c.ClockDrift >= 100 {
		return ErrBadConfig
	}
	if c.MaxMessageDelay <= 0 && c.DelayDistribution == nil {
		return ErrBadConfig
	}
	if c.PendingEventThreshold == 0 {
		c.PendingEventThreshold = 1
	}
	if c.MessageBias == 0 {
		c.MessageBias = 0.5
	}
	return nil
}

func (c *TimeoutStrategyConfig) DelayValue() int {
	if c.DelayDistribution != nil && c.Nondeterministic {
		max := math.Min(float64(c.MaxMessageDelay.Milliseconds()), float64(c.DelayDistribution.Rand()))
		return int(max)
	}
	return int(c.MaxMessageDelay.Milliseconds())
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
			ctx.Logger.With(log.LogParams{
				"timeout": t.Key(),
			}).Debug("Firing timeout")
			return d.DispatchTimeout(t)
		},
	}
}

type TimeoutStrategy struct {
	pendingEvents   *types.Map[string, *pendingEvent]
	symbolMap       *types.Map[string, *z3.AST]
	delayVals       *types.Map[string, int]
	pendingEventCtr *util.Counter
	config          *TimeoutStrategyConfig
	normalTimer     *timer
	records         *records
	dist            *distuv.Bernoulli

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
	if config.Z3Timeout != 0 {
		z3config.SetParamValue(
			"timeout",
			strconv.FormatInt(config.Z3Timeout.Milliseconds(), 10),
		)
	}
	z3context := z3.NewContext(z3config)

	strategy := &TimeoutStrategy{
		pendingEvents:   types.NewMap[string, *pendingEvent](),
		symbolMap:       types.NewMap[string, *z3.AST](),
		delayVals:       types.NewMap[string, int](),
		pendingEventCtr: util.NewCounter(),
		config:          config,
		records:         newRecords(config.RecordFilePath),
		dist: &distuv.Bernoulli{
			P:   config.MessageBias,
			Src: rand.NewSource(uint64(time.Now().UnixNano())),
		},
		z3config:    z3config,
		z3context:   z3context,
		z3solver:    z3context.NewSolver(),
		BaseService: types.NewBaseService("TimeoutStrategy", nil),
	}
	if !config.Nondeterministic {
		strategy.normalTimer = newTimer()
	}
	strategy.z3context.SetErrorHandler(func(ctx *z3.Context, ec z3.ErrorCode) {
		msg := ctx.Error(ec)
		strategy.Logger.With(log.LogParams{
			"error": msg,
		}).Error("Z3 error! Stopping!")
		strategy.Stop()
	})
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
			t.Logger.With(log.LogParams{
				"timeout":  timeout.Key(),
				"duration": timeout.Duration.String(),
				"replica":  timeout.Replica,
			}).Debug("starting timeout")
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
		t.pendingEvents.Remove(p.label)
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
	constraints := []*z3.AST{eSymbol.Ge(t.z3context.Int(0))}

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
		isParentReceive := false
		var messageID string
		isParentTimeoutStart := false
		if pNode.Event.IsMessageSend() && e.IsMessageReceive() {
			pmID, _ := pNode.Event.MessageID()
			emID, _ := e.MessageID()
			messageID = string(pmID)
			isParentReceive = pmID == emID
		} else if pNode.Event.IsTimeoutStart() && e.IsTimeoutEnd() {
			pTimeout, _ := pNode.Event.Timeout()
			eTimeout, _ := e.Timeout()
			isParentTimeoutStart = pTimeout.Eq(eTimeout)
		}

		if isParentReceive {
			delayVal, ok := t.delayVals.Get(messageID)
			if !ok {
				delayVal = t.config.DelayValue()
				t.delayVals.Add(messageID, delayVal)
			}
			t.records.updateDistVal(ctx, delayVal)
			if t.config.UseDistribution() {
				constraints = append(
					constraints,
					pSymbol.Mul(t.config.driftMin(t.z3context)).Add(t.z3context.Int(delayVal)).Le(eSymbol),
					eSymbol.Sub(pSymbol.Mul(t.config.driftMax(t.z3context))).Le(t.z3context.Int(delayVal)),
				)
			} else {
				constraints = append(
					constraints,
					pSymbol.Mul(t.config.driftMin(t.z3context)).Sub(eSymbol).Le(t.z3context.Int(0)),
					eSymbol.Sub(pSymbol.Mul(t.config.driftMax(t.z3context))).Le(t.z3context.Int(delayVal)),
				)
			}
		} else if isParentTimeoutStart {
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
		ctx.Logger.With(log.LogParams{
			"constraint": c.String(),
			"event_type": e.TypeS,
			"event_id":   e.ID,
		}).Debug("Adding constraint")
		t.z3solver.Assert(c)

		// result := t.z3solver.Check()
		// if result != z3.True {
		// 	panic("constraint not solvable")
		// }
	}
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
	t.delayVals.RemoveAll()

	if t.z3solver != nil {
		spurious := false
		ctx.Logger.With(log.LogParams{"iteration": ctx.CurIteration()}).Debug("checking spuriousness")
		solvable := t.z3solver.Check()
		if solvable != z3.True {
			spurious = true
		}
		ctx.Logger.With(log.LogParams{"iteration": ctx.CurIteration()}).Debug("done checking spuriousness")
		t.records.newIteration(spurious, ctx.CurIteration())
		t.z3solver.Reset()
	}

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
