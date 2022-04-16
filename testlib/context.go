package testlib

import (
	"fmt"
	"sync"

	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/netrix/util"
)

var (
	partitionKey = "_partition"
)

// Context struct is passed to the calls of StateAction and Condition
// encapsulates all information needed by the StateAction and Condition to function
type Context struct {
	// MessagePool reference to an instance of the MessageStore
	MessagePool *types.MessageStore
	// Replicas reference to the replica store
	Replicas *types.ReplicaStore
	// EventDAG is the directed acyclic graph all prior events
	EventDAG *types.EventDAG
	// Vars is a generic key value store to facilate maintaining auxilliary information
	// during the execution of a testcase
	Vars *VarSet

	counter     *util.Counter
	testcase    *TestCase
	reportStore *types.ReportStore
	sends       map[string]*types.Event
	lock        *sync.Mutex
	once        *sync.Once
}

// newContext instantiates a Context from the RootContext
func newContext(c *context.RootContext, testcase *TestCase) *Context {
	return &Context{
		MessagePool: c.MessageStore,
		Replicas:    c.Replicas,
		EventDAG:    types.NewEventDag(c.Replicas),
		Vars:        NewVarSet(),

		counter:     util.NewCounter(),
		reportStore: c.ReportStore,
		testcase:    testcase,
		sends:       make(map[string]*types.Event),
		lock:        new(sync.Mutex),
		once:        new(sync.Once),
	}
}

func (c *Context) CreatePartition(sizes []int, labels []string) {
	partition, err := NewPartition(c.Replicas, sizes, labels)
	if err != nil {
		return
	}
	c.Vars.Set(partitionKey, partition)
}

// Logger returns the logger for the current testcase
func (c *Context) Logger() *log.Logger {
	return c.testcase.Logger
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
func (c *Context) NewMessage(cur *types.Message, data []byte) *types.Message {
	return &types.Message{
		From:      cur.From,
		To:        cur.To,
		Data:      data,
		Type:      cur.Type,
		ID:        fmt.Sprintf("%s_%s_change%d", cur.From, cur.To, c.counter.Next()),
		Intercept: cur.Intercept,
	}
}

// GetMessage returns the `Message` struct from the Message pool
// if the event provided is a message send ot receive event
func (c *Context) GetMessage(e *types.Event) (*types.Message, bool) {
	if !e.IsMessageSend() && !e.IsMessageReceive() {
		return nil, false
	}
	mID, _ := e.MessageID()
	return c.MessagePool.Get(mID)
}

func (c *Context) Log(keyvals map[string]string) {
	keyvals["testcase"] = c.testcase.Name
	c.reportStore.Log(keyvals)
}

func (c *Context) setEvent(e *types.Event) {
	c.lock.Lock()
	defer c.lock.Unlock()

	parents := make([]*types.Event, 0)
	switch e.Type.(type) {
	case *types.MessageReceiveEventType:
		eventType := e.Type.(*types.MessageReceiveEventType)
		send, ok := c.sends[eventType.MessageID]
		if ok {
			parents = append(parents, send)
		}
	case *types.MessageSendEventType:
		eventType := e.Type.(*types.MessageSendEventType)
		c.sends[eventType.MessageID] = e
	}
	c.Logger().With(log.LogParams{
		"event_id": e.ID,
		"parents":  parents,
	}).Debug("Adding node to DAG")
	c.EventDAG.AddNode(e, parents)
}
