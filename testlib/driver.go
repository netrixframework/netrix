package testlib

import (
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

type testCaseDriver struct {
	TestCase *TestCase
	ctx      *Context
}

func NewTestDriver(ctx *context.RootContext, testcase *TestCase) *testCaseDriver {
	return &testCaseDriver{
		TestCase: testcase,
		ctx:      NewContext(ctx, testcase),
	}
}

func (d *testCaseDriver) Step(e *types.Event) []*types.Message {
	d.ctx.reportStore.Log(map[string]string{
		"testcase":   d.TestCase.Name,
		"type":       "event",
		"replica":    string(e.Replica),
		"event_type": e.TypeS,
	})
	d.ctx.EventDAG.AddNode(e, []*types.Event{})
	d.TestCase.Logger.With(log.LogParams{"event_id": e.ID, "type": e.TypeS}).Debug("Stepping")
	messages, _ := d.TestCase.Step(e, d.ctx)

	for _, m := range messages {
		if !d.ctx.MessagePool.Exists(m.ID) {
			d.ctx.MessagePool.Add(m.ID, m)
		}
	}
	return messages
}

func (d *testCaseDriver) setup() error {
	return d.TestCase.Setup(d.ctx)
}
