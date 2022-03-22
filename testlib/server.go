package testlib

import (
	"github.com/netrixframework/netrix/apiserver"
	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

// TestingServer is used to run the scheduler tool for unit testing
type TestingServer struct {
	apiserver  *apiserver.APIServer
	dispatcher *dispatcher.Dispatcher
	ctx        *context.RootContext
	messageCh  chan *types.Message
	eventCh    chan *types.Event

	doneCh chan string

	testCases      map[string]*TestCase
	executionState *executionState
	*types.BaseService
}

// NewTestingServer instantiates TestingServer
// testcases are passed as arguments
func NewTestingServer(config *config.Config, messageParser types.MessageParser, testcases []*TestCase) (*TestingServer, error) {
	log.Init(config.LogConfig)
	ctx := context.NewRootContext(config, log.DefaultLogger)

	server := &TestingServer{
		apiserver:      nil,
		dispatcher:     dispatcher.NewDispatcher(ctx),
		ctx:            ctx,
		messageCh:      ctx.MessageQueue.Subscribe("testingServer"),
		eventCh:        ctx.EventQueue.Subscribe("testingServer"),
		doneCh:         make(chan string),
		testCases:      make(map[string]*TestCase),
		executionState: newExecutionState(),
		BaseService:    types.NewBaseService("TestingServer", log.DefaultLogger),
	}
	for _, t := range testcases {
		server.testCases[t.Name] = t
	}

	server.apiserver = apiserver.NewAPIServer(ctx, messageParser, server)

	for _, t := range testcases {
		t.Logger = server.Logger.With(log.LogParams{"testcase": t.Name})
	}
	return server, nil
}

// Start starts the TestingServer and implements Service
func (srv *TestingServer) Start() {
	srv.StartRunning()
	srv.apiserver.Start()
	srv.ctx.Start()
	srv.execute()

	// Just keep running until asked to stop
	// For dashboard purposes
	<-srv.QuitCh()
}

// Done returns the channel which will be closed once all testcases are run
func (srv *TestingServer) Done() chan string {
	return srv.doneCh
}

// Stop stops the TestingServer and implements Service
func (srv *TestingServer) Stop() {
	srv.StopRunning()
	srv.apiserver.Stop()
	srv.ctx.Stop()
}
