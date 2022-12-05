package testlib

import (
	"sync"
	"time"

	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

type executionState struct {
	allowEvents    bool
	testcaseDriver *testCaseDriver
	lock           *sync.Mutex
}

func newExecutionState() *executionState {
	return &executionState{
		allowEvents:    false,
		testcaseDriver: nil,
		lock:           new(sync.Mutex),
	}
}

func (e *executionState) Block() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.allowEvents = false
}

func (e *executionState) Unblock() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.allowEvents = true
}

func (e *executionState) NewTestCase(
	ctx *context.RootContext,
	testcase *TestCase,
) error {
	driver := NewTestDriver(ctx, testcase)
	e.lock.Lock()
	e.testcaseDriver = driver
	e.lock.Unlock()
	return driver.setup()
}

func (e *executionState) CurTestCaseDriver() *testCaseDriver {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.testcaseDriver
}

func (e *executionState) IsBlocked() bool {
	e.lock.Lock()
	defer e.lock.Unlock()
	return !e.allowEvents
}

func (srv *TestingServer) execute() {
	srv.Logger.Info("Waiting for all replicas to connect...")
	for {
		select {
		case <-srv.QuitCh():
			return
		default:
		}
		if srv.ctx.Replicas.Count() == srv.ctx.Config.NumReplicas {
			break
		}
	}
	srv.Logger.Info("All replicas connected.")
	go srv.pollEvents()
	go srv.pollMessages()

MainLoop:
	for _, testcase := range srv.testCases {
		testcaseLogger := testcase.Logger
		testcaseLogger.Info("Starting testcase")
		testcaseLogger.Debug("Waiting for replicas to be ready")

	ReplicaReadyLoop:
		for {
			if srv.ctx.Replicas.NumReady() == srv.ctx.Config.NumReplicas {
				break ReplicaReadyLoop
			}
		}
		testcaseLogger.Debug("Replicas are ready")

		// Setup
		testcaseLogger.Debug("Setting up testcase")
		err := srv.executionState.NewTestCase(srv.ctx, testcase)
		if err != nil {
			testcaseLogger.With(log.LogParams{"error": err}).Error("Error setting up testcase")
			goto Finalize
		}
		// Wait for completion or timeout
		testcaseLogger.Debug("Waiting for completion")
		srv.executionState.Unblock()
		select {
		case <-testcase.doneCh.Ch():
		case <-time.After(testcase.Timeout):
			testcaseLogger.Info("Testcase timedout")
		case <-srv.QuitCh():
			break MainLoop
		}

		// Stopping further processing of events
		srv.executionState.Block()

	Finalize:
		// Finalize report
		testcaseLogger.Info("Finalizing")
		if testcase.aborted {
			testcaseLogger.Info("Testcase was aborted")
		}
		testcaseLogger.Info("Checking assertion")
		ok := testcase.assert()
		var okS string
		if !ok {
			testcaseLogger.Info("Testcase failed")
			okS = "fail"
		} else {
			testcaseLogger.Info("Testcase succeeded")
			okS = "succeed"
		}
		srv.ctx.ReportStore.Log(map[string]string{
			"testcase": testcase.Name,
			"type":     "testcase_result",
			"result":   okS,
		})
		if err := srv.apiserver.RestartAll(); err != nil {
			srv.Logger.With(log.LogParams{"error": err}).Error("Failed to restart replicas! Aborting!")
			break MainLoop
		}
		srv.ctx.Reset()
	}
	close(srv.doneCh)
}

func (srv *TestingServer) pollEvents() {
EventLoop:
	for {
		select {
		case <-srv.QuitCh():
			return
		default:
		}

		if srv.executionState.IsBlocked() {
			continue EventLoop
		}
		e, ok := srv.ctx.EventQueue.Pop()
		if !ok {
			continue EventLoop
		}
		testcaseDriver := srv.executionState.CurTestCaseDriver()
		messages := testcaseDriver.Step(e)

		go srv.dispatchMessages(messages)
	}
}

func (srv *TestingServer) pollMessages() {
MessageLoop:
	for {
		select {
		case <-srv.QuitCh():
			return
		default:
		}
		if srv.executionState.IsBlocked() {
			continue MessageLoop
		}

		m, ok := srv.ctx.MessageQueue.Pop()
		if !ok {
			continue MessageLoop
		}
		testcaseDriver := srv.executionState.CurTestCaseDriver()
		// Gathering metrics
		srv.ctx.ReportStore.Log(map[string]string{
			"testcase":   testcaseDriver.TestCase.Name,
			"type":       "message",
			"from":       string(m.From),
			"to":         string(m.To),
			"message":    m.Repr,
			"message_id": string(m.ID),
		})
	}
}

func (srv *TestingServer) dispatchMessages(messages []*types.Message) {
	for _, m := range messages {
		srv.apiserver.SendMessage(m)
	}
}
