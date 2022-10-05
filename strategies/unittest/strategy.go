package unittest

import (
	"sync"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
)

type TestCaseStrategy struct {
	*types.BaseService
	testCase    *testlib.TestCase
	testCaseCtx *testlib.Context

	success int
	lock    *sync.Mutex
}

func NewTestCaseStrategy(testCase *testlib.TestCase) *TestCaseStrategy {
	return &TestCaseStrategy{
		BaseService: types.NewBaseService("TestCaseStrategy", nil),
		testCase:    testCase,
		success:     0,
		lock:        new(sync.Mutex),
	}
}

var _ strategies.Strategy = &TestCaseStrategy{}

func (t *TestCaseStrategy) Step(e *types.Event, ctx *strategies.Context) strategies.Action {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.testCaseCtx == nil {
		t.testCaseCtx = testlib.NewContextFrom(ctx.Context, t.testCase)
	}
	messages := t.testCase.Step(e, t.testCaseCtx)
	if len(messages) == 0 {
		return strategies.DoNothing()
	}

	return strategies.DeliverMany(messages)
}

func (t *TestCaseStrategy) EndCurIteration(ctx *strategies.Context) {
	if t.testCase.StateMachine.InSuccessState() {
		t.lock.Lock()
		t.success += 1
		t.lock.Unlock()
	}
}

func (t *TestCaseStrategy) NextIteration(ctx *strategies.Context) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.testCaseCtx = testlib.NewContextFrom(ctx.Context, t.testCase)
	t.testCase.StateMachine.Reset()
	t.testCase.Setup(t.testCaseCtx)
}

func (t *TestCaseStrategy) Finalize(ctx *strategies.Context) {
	t.lock.Lock()
	success := t.success
	t.lock.Unlock()
	ctx.Logger.With(log.LogParams{
		"success": success,
	}).Info("Total successful iterations")
}

func (p *TestCaseStrategy) Start() error {
	p.BaseService.StartRunning()
	return nil
}

func (p *TestCaseStrategy) Stop() error {
	p.BaseService.StopRunning()
	return nil
}
