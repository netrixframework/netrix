// Package context defines the global context object used by Netrix.
package context

import (
	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/netrix/util"
)

// RootContext stores the context of the scheduler
type RootContext struct {
	// Config and instance of the configuration object
	Config *config.Config
	// EventIDGen a thread safe counter to generate event ids
	EventIDGen *util.Counter
	// Replicas instance of the ReplicaStore which contains information of all the replicas
	Replicas *types.ReplicaStore
	// MessageQueue stores the messages that are _intercepted_ as a queue
	MessageQueue *types.Queue[*types.Message]
	// MessageStore stores the messages that are _intercepted_ as a map
	MessageStore *types.Map[types.MessageID, *types.Message]
	// EventQueue stores the events sent by the replicas as a queue
	EventQueue *types.Queue[*types.Event]
	// Logger for logging purposes
	Logger *log.Logger
	// ReportStore contains a log of important events
	ReportStore *types.ReportLogs
}

// NewRootContext creates an instance of the RootContext from the configuration
func NewRootContext(config *config.Config, logger *log.Logger) *RootContext {
	return &RootContext{
		Config:       config,
		EventIDGen:   util.NewCounter(),
		Replicas:     types.NewReplicaStore(config.NumReplicas),
		MessageQueue: types.NewQueue[*types.Message](),
		MessageStore: types.NewMap[types.MessageID, *types.Message](),
		EventQueue:   types.NewQueue[*types.Event](),
		Logger:       logger,
		ReportStore:  types.NewReportLogs(),
	}
}

// Reset implements Service
func (c *RootContext) Reset() {
	c.MessageQueue.Reset()
	c.EventQueue.Reset()

	c.MessageStore.RemoveAll()
}

// PauseQueues invokes [types.Queue.Pause] on the message and event queues
func (c *RootContext) PauseQueues() {
	c.MessageQueue.Pause()
	c.EventQueue.Pause()
}

// ResumeQueues invokes [types.Queue.Resume] on the message and event queues
func (c *RootContext) ResumeQueues() {
	c.MessageQueue.Resume()
	c.EventQueue.Resume()
}
