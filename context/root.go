package context

import (
	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

// RootContext stores the context of the scheduler
type RootContext struct {
	// Config and instance of the configuration object
	Config *config.Config
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
		Replicas:     types.NewReplicaStore(config.NumReplicas),
		MessageQueue: types.NewQueue[*types.Message](logger),
		MessageStore: types.NewMap[types.MessageID, *types.Message](),
		EventQueue:   types.NewQueue[*types.Event](logger),
		Logger:       logger,
		ReportStore:  types.NewReportLogs(),
	}
}

// Start implements Service and initializes the queues
func (c *RootContext) Start() {
	c.MessageQueue.Start()
	c.EventQueue.Start()
}

// Stop implements Service and terminates the queues
func (c *RootContext) Stop() {
	c.MessageQueue.Stop()
	c.EventQueue.Stop()
}

// Reset implements Service
func (c *RootContext) Reset() {
	c.MessageQueue.Flush()
	c.EventQueue.Flush()

	c.MessageStore.RemoveAll()
}
