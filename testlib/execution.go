package testlib

import (
	"sync"
	"time"

	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

type executionState struct {
	allowEvents bool
	curCtx      *Context
	testcase    *TestCase
	lock        *sync.Mutex
}

func newExecutionState() *executionState {
	return &executionState{
		allowEvents: false,
		curCtx:      nil,
		testcase:    nil,
		lock:        new(sync.Mutex),
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
	r *reportStore,
	d *dispatcher.Dispatcher,
) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.testcase = testcase
	e.curCtx = newContext(ctx, testcase, r, d)
}

func (e *executionState) CurCtx() *Context {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.curCtx
}

func (e *executionState) CurTestCase() *TestCase {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.testcase
}

func (e *executionState) CanAllowEvents() bool {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.allowEvents
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
		srv.executionState.NewTestCase(srv.ctx, testcase, srv.reportStore, srv.dispatcher)
		err := testcase.setup(srv.executionState.CurCtx())
		if err != nil {
			testcaseLogger.With(log.LogParams{"error": err}).Error("Error setting up testcase")
			goto Finalize
		}
		// Wait for completion or timeout
		testcaseLogger.Debug("Waiting for completion")
		srv.executionState.Unblock()
		select {
		case <-testcase.doneCh:
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
		ctx := srv.executionState.CurCtx()
		if testcase.aborted {
			testcaseLogger.Info("Testcase was aborted")
		}
		testcaseLogger.Info("Checking assertion")
		ok := testcase.assert(ctx)
		if !ok {
			testcaseLogger.Info("Testcase failed")
		} else {
			testcaseLogger.Info("Testcase succeeded")
		}
		// TODO: Figure out a way to cleanly clear the message pool
		// Reset the servers and flush the queues after waiting for some time
		if err := srv.dispatcher.RestartAll(); err != nil {
			srv.Logger.With(log.LogParams{"error": err}).Error("Failed to restart replicas! Aborting!")
			break MainLoop
		}
		// TODO: need to wait for in transit messages before invoking reset
		srv.ctx.Reset()
	}
	close(srv.doneCh)
}

func (srv *TestingServer) pollEvents() {
	for {
		if !srv.executionState.CanAllowEvents() {
			continue
		}
		select {
		case e := <-srv.eventCh:

			ctx := srv.executionState.CurCtx()
			testcase := srv.executionState.CurTestCase()
			srv.reportStore.Log(map[string]string{
				"testcase":   testcase.Name,
				"type":       "event",
				"replica":    string(e.Replica),
				"event_type": e.TypeS,
			})

			testcaseLogger := srv.Logger.With(log.LogParams{"testcase": testcase.Name})

			testcaseLogger.With(log.LogParams{"event_id": e.ID, "type": e.TypeS}).Debug("Stepping")
			// 1. Add event to context and feed context to testcase
			ctx.setEvent(e)
			messages := testcase.step(e, ctx)

			select {
			case <-testcase.doneCh:
			default:
				go srv.dispatchMessages(ctx, messages)
			}
		case <-srv.QuitCh():
			return
		}
	}
}

func (srv *TestingServer) pollMessages() {
	for {
		if !srv.executionState.CanAllowEvents() {
			continue
		}
		select {
		case m := <-srv.messageCh:
			testcase := srv.executionState.CurTestCase()
			// Gathering metrics
			srv.reportStore.Log(map[string]string{
				"testcase":   testcase.Name,
				"type":       "message",
				"from":       string(m.From),
				"to":         string(m.To),
				"message":    m.Repr,
				"message_id": m.ID,
			})
		case <-srv.QuitCh():
			return
		}
	}
}

func (srv *TestingServer) dispatchMessages(ctx *Context, messages []*types.Message) {
	for _, m := range messages {
		if !ctx.MessagePool.Exists(m.ID) {
			ctx.MessagePool.Add(m)
		}
		srv.dispatcher.DispatchMessage(m)
	}
}
