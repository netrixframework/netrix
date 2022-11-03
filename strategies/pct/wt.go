package pct

import (
	"sync"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
)

type PCTStrategyWithTestCase struct {
	*PCTStrategy
	testCase    *testlib.TestCase
	testCaseCtx *testlib.Context
	lock        *sync.Mutex
}

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
		p.mo.AddSendEvent(m)
		if !ctx.MessagePool.Exists(m.ID) {
			ctx.MessagePool.Add(m.ID, m)
		}
	}

	if e.IsMessageReceive() {
		message, ok := ctx.GetMessage(e)
		if ok {
			p.mo.AddRecvEvent(message)
		}
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
		p.AddMessage(NewMessage(m), ctx)
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
