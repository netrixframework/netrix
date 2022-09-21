package pct

import (
	"sync"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
)

type PCTStrategyWithTest struct {
	*PCTStrategy
	testCase    *testlib.TestCase
	testCaseCtx *testlib.Context
	lock        *sync.Mutex
}

func NewPCTStrategyWithTest(config *PCTStrategyConfig, testCase *testlib.TestCase) *PCTStrategyWithTest {
	return &PCTStrategyWithTest{
		PCTStrategy: NewPCTStrategy(config),
		testCase:    testCase,
		lock:        new(sync.Mutex),
	}
}

func (p *PCTStrategyWithTest) Step(e *types.Event, ctx *strategies.Context) strategies.Action {
	p.lock.Lock()
	if p.testCaseCtx == nil {
		p.testCaseCtx = testlib.NewContextFrom(ctx.Context, p.testCase)
	}
	messages := p.testCase.Step(e, p.testCaseCtx)
	p.lock.Unlock()

	for _, m := range messages {
		p.mo.AddSendEvent(m)
		p.AddMessage(NewMessage(m), ctx)
	}

	if e.IsMessageReceive() {
		message, ok := ctx.GetMessage(e)
		if ok {
			p.mo.AddRecvEvent(message)
		}
	}

	event, ok := p.Schedule()
	if ok {
		message, ok := ctx.MessagePool.Get(event.messageID)
		if ok {
			return strategies.DeliverMessage(message)
		}
	}
	return strategies.DoNothing()
}

func (p *PCTStrategyWithTest) EndCurIteration(ctx *strategies.Context) {
	p.lock.Lock()
	p.testCaseCtx = nil
	p.lock.Unlock()

	p.PCTStrategy.EndCurIteration(ctx)
}

func (p *PCTStrategyWithTest) NextIteration(ctx *strategies.Context) {
	p.lock.Lock()
	p.testCaseCtx = testlib.NewContextFrom(ctx.Context, p.testCase)
	p.lock.Unlock()

	p.PCTStrategy.NextIteration(ctx)
}
