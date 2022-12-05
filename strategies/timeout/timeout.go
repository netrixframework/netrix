// Package timeout encodes a strategy where timeout durations are chosen non deterministically
package timeout

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/netrixframework/netrix/apiserver"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/netrix/util"
	"github.com/netrixframework/netrix/util/z3"
)

var (
	// ErrBadConfig is returned when creating the strategy config and the required parameters are absent
	ErrBadConfig = errors.New("one of MaxMessageDelay or DelayDistribution should be specified")
)

// Distribution encodes a probability distribution to sample values from
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
}

func (c *TimeoutStrategyConfig) useDistribution() bool {
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
	return nil
}

func (c *TimeoutStrategyConfig) delayValue() int {
	if c.DelayDistribution != nil && c.Nondeterministic {
		// max := math.Min(float64(c.MaxMessageDelay.Milliseconds()), float64(c.DelayDistribution.Rand()))
		// return int(max)
		return c.DelayDistribution.Rand()
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

// FireTimeout action instructs the replica to fire a timeout
func FireTimeout(t *types.ReplicaTimeout) *strategies.Action {
	return &strategies.Action{
		Name: "FireTimeout",
		Do: func(ctx *strategies.Context, d *apiserver.APIServer) error {
			ctx.Logger.With(log.LogParams{
				"timeout": t.Key(),
			}).Debug("Firing timeout")
			return d.SendTimeout(t)
		},
	}
}

// TimeoutStrategy is an experimental strategy to test distributed systems.
// We non deterministically set the duration of the timeout while maintaining
// the constraints imposed by a natural monotonic order of time.
type TimeoutStrategy struct {
	Actions         *types.Channel[*strategies.Action]
	pendingEvents   *types.Map[string, *pendingEvent]
	symbolMap       *types.Map[string, *z3.AST]
	delayVals       *types.Map[string, int]
	pendingEventCtr *util.Counter
	config          *TimeoutStrategyConfig
	normalTimer     *timer
	records         *records

	z3config  *z3.Config
	z3context *z3.Context
	z3solver  *z3.Optimizer
	*types.BaseService
}

var _ strategies.Strategy = &TimeoutStrategy{}

// NewTimeoutStrategy creates a new [TimeoutStrategy] with the corresponding config. Returns and error when the config is invalid
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
		Actions:         types.NewChannel[*strategies.Action](),
		pendingEvents:   types.NewMap[string, *pendingEvent](),
		symbolMap:       types.NewMap[string, *z3.AST](),
		delayVals:       types.NewMap[string, int](),
		pendingEventCtr: util.NewCounter(),
		config:          config,
		records:         newRecords(config.RecordFilePath),
		z3config:        z3config,
		z3context:       z3context,
		z3solver:        z3context.NewOptimizer(),
		BaseService:     types.NewBaseService("TimeoutStrategy", nil),
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

func (t *TimeoutStrategy) ActionsCh() *types.Channel[*strategies.Action] {
	return t.Actions
}

func (t *TimeoutStrategy) Step(e *types.Event, ctx *strategies.Context) {
	if !t.Running() {
		return
	}
	t.records.step(ctx)
	// 1. Update constraints (add e to the graph and update the set of constraints in the solver)
	if !t.updateConstraints(e, ctx) {
		// Panic

		return
	}
	if !t.config.Nondeterministic {
		if e.IsMessageSend() {
			message, ok := ctx.GetMessage(e)
			if ok {
				t.Actions.BlockingAdd(strategies.DeliverMessage(message))
				return
			}
		} else if e.IsTimeoutStart() {
			timeout, _ := e.Timeout()
			t.Logger.With(log.LogParams{
				"timeout":  timeout.Key(),
				"duration": timeout.Duration.String(),
				"replica":  timeout.Replica,
			}).Debug("starting timeout")
			t.normalTimer.Add(timeout)
			return
		}
		for _, to := range t.normalTimer.Ready() {
			t.Actions.BlockingAdd(FireTimeout(to))
		}
		return
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
			t.Actions.BlockingAdd(FireTimeout(p.timeout))
		} else if p.message != nil {
			t.Logger.With(log.LogParams{
				"message_id": p.message.ID,
			}).Debug("delivering message")
			t.Actions.BlockingAdd(strategies.DeliverMessage(p.message))
		}
	}
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

	isInternalEvent := e.IsMessageSend() || e.IsGeneric()

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

		if isInternalEvent {
			constraints = append(
				constraints,
				pSymbol.Eq(eSymbol),
			)
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
				delayVal = t.config.delayValue()
				t.delayVals.Add(messageID, delayVal)
			}
			t.records.updateDistVal(ctx, delayVal)
			if t.config.useDistribution() {
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

		result := t.z3solver.Check()
		if result != z3.True {
			panic("constraint not solvable")
		}
	}
	return true
}

func (t *TimeoutStrategy) Finalize(c *strategies.Context) {
	t.records.summarize(c)
}

func (t *TimeoutStrategy) EndCurIteration(ctx *strategies.Context) {
	if t.z3solver != nil {
		spurious := false
		var expr *z3.AST = nil
		opts := make(map[string]*z3.OptimizedValue)
		for _, r := range ctx.ReplicaStore.Iter() {
			e, ok := ctx.EventDAG.GetLatestNode(r.ID)
			if ok {
				symb, ok := t.symbolMap.Get(fmt.Sprintf("e_%d", e.ID))
				if ok {
					if expr == nil {
						expr = symb
					} else {
						expr = expr.Add(symb)
					}
					opts[string(r.ID)] = t.z3solver.Minimize(symb)
				}
			}
		}
		if expr != nil {
			t.z3solver.Minimize(expr)
		}
		ctx.Logger.With(log.LogParams{"iteration": ctx.CurIteration()}).Debug("checking spuriousness")
		spurious = t.z3solver.Check() != z3.True
		ctx.Logger.With(log.LogParams{"iteration": ctx.CurIteration(), "spurious": spurious}).Debug("done checking spuriousness")
		if !spurious {
			params := log.LogParams{}
			for r, o := range opts {
				ctx.Logger.With(log.LogParams{"replica": r}).Debug("Fetching optimal value")
				val, ok := o.Value()
				if ok {
					params[r] = val
				}
			}
			ctx.Logger.With(params).Info("Optimized values")
		}
		t.records.newIteration(spurious, ctx.CurIteration()+1)
	}
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
