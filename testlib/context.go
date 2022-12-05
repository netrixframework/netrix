package testlib

import (
	"fmt"

	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/netrix/util"
)

var (
	partitionKey = "_partition"
)

// Context struct is passed to the calls of StateAction and Condition
// encapsulates all information needed by the StateAction and Condition to function
type Context struct {
	*sm.Context

	counter     *util.Counter
	testcase    *TestCase
	reportStore *types.ReportLogs
}

// NewContext instantiates a Context from the RootContext
func NewContext(c *context.RootContext, testcase *TestCase) *Context {
	return &Context{
		Context:     sm.NewContext(c, testcase.Logger),
		counter:     util.NewCounter(),
		reportStore: c.ReportStore,
		testcase:    testcase,
	}
}

// NewContextFrom instantiates a context from the specified state machine context and testcase.
func NewContextFrom(sm *sm.Context, testcase *TestCase) *Context {
	return &Context{
		Context:     sm,
		counter:     util.NewCounter(),
		reportStore: nil,
		testcase:    testcase,
	}
}

// CreatePartition creates a new partition with the specified sizes and labels
func (c *Context) CreatePartition(sizes []int, labels []string) {
	partition, err := NewRandomPartition(sizes, labels)
	if err != nil {
		return
	}
	partition.Setup(c)
}

// Abort stops the execution of the testcase
func (c *Context) Abort() {
	c.testcase.Abort()
}

// Ends the testcase without failing. The assertion will determine the success of the testcase
func (c *Context) EndTestCase() {
	c.testcase.End()
}

// NewMessage crafts a new message with a new ID
// The current message contents are replaced with `data`
func (c *Context) NewMessage(cur *types.Message, data []byte, pMsg types.ParsedMessage) *types.Message {
	return &types.Message{
		From:          cur.From,
		To:            cur.To,
		Data:          data,
		Type:          cur.Type,
		ID:            types.MessageID(fmt.Sprintf("%s_%s_change%d", cur.From, cur.To, c.counter.Next())),
		Intercept:     cur.Intercept,
		ParsedMessage: pMsg,
	}
}

// Log adds a report log for the current testcase
func (c *Context) Log(keyvals map[string]string) {
	if c.reportStore == nil {
		return
	}
	keyvals["testcase"] = c.testcase.Name
	c.reportStore.Log(keyvals)
}
