package pct

import (
	"sync"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
)

// PCTStrategyWithTestCase encapsulates PCTStrategy with a testcase
// For each event, we invoke the filters of the testcase and when none of the filter
// conditions are satisfied, the event is passed to PCTStrategy.
type PCTStrategyWithTestCase struct {
	*PCTStrategy
	testCase    *testlib.TestCase
	testCaseCtx *testlib.Context
	lock        *sync.Mutex
}

// NewPCTStrategyWithTestCase creates a new PCTStrategyWithTestCase
func NewPCTStrategyWithTestCase(config *PCTStrategyConfig, testCase *testlib.TestCase) *PCTStrategyWithTestCase {
	return &PCTStrategyWithTestCase{
		PCTStrategy: NewPCTStrategy(config),
		testCase:    testCase,
		lock:        new(sync.Mutex),
	}
}

func (p *PCTStrategyWithTestCase) Step(e *types.Event, ctx *strategies.Context) {
	p.lock.Lock()
	if p.testCaseCtx == nil {
		p.testCaseCtx = testlib.NewContextFrom(ctx.Context, p.testCase)
	}
	messages, handled := p.testCase.Step(e, p.testCaseCtx)
	p.lock.Unlock()

	for _, m := range messages {
		if !ctx.MessagePool.Exists(m.ID) {
			ctx.MessagePool.Add(m.ID, m)
		}
	}

	message, ok := ctx.GetMessage(e)
	if e.IsMessageSend() && ok {
		p.mo.AddSendEvent(message)
	} else if e.IsMessageReceive() && ok {
		p.mo.AddRecvEvent(message)
	}

	if handled {
		if len(messages) > 0 {
			p.Actions.BlockingAdd(strategies.DeliverMany(messages))
		}
		return
	}

	for _, m := range messages {
		p.Logger.With(log.LogParams{
			// "message": m.ParsedMessage.String(),
			"from": m.From,
			"to":   m.To,
			"id":   m.ID,
		}).Debug("Adding message to PCT")
		p.AddMessage(newPCTMessage(m), ctx)
	}

	event, ok := p.Schedule()
	if ok {
		message, ok := ctx.MessagePool.Get(event.messageID)
		if ok {
			p.Actions.BlockingAdd(strategies.DeliverMessage(message))
			return
		}
	}
}

func (p *PCTStrategyWithTestCase) EndCurIteration(ctx *strategies.Context) {
	p.lock.Lock()
	p.testCaseCtx = nil
	p.lock.Unlock()

	p.PCTStrategy.EndCurIteration(ctx)
}

func (p *PCTStrategyWithTestCase) NextIteration(ctx *strategies.Context) {
	p.lock.Lock()
	p.testCaseCtx = testlib.NewContextFrom(ctx.Context, p.testCase)
	p.testCase.StateMachine.Reset()
	p.testCase.Setup(p.testCaseCtx)
	p.lock.Unlock()

	p.PCTStrategy.NextIteration(ctx)
}
