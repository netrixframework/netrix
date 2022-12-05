package testlib

import (
	"sync"
	"time"

	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
)

// TestCase represents a unit test case
type TestCase struct {
	// Name name of the testcase
	Name string
	// Timeout maximum duration of the testcase execution
	Timeout time.Duration
	// Setup function called prior to initiation of the execution
	Setup    func(*Context) error
	assertFn func(*Context) bool
	// Cascade instance of *HandlerCascade
	Cascade *FilterSet
	// StateMachine instance of *StateMachine to assert a property
	StateMachine *sm.StateMachine
	aborted      bool
	// Logger to log information
	Logger *log.Logger

	doneCh *types.Channel[bool]
	once   *sync.Once
}

func defaultSetupFunc(c *Context) error {
	return nil
}

func defaultAssertFunc(c *Context) bool {
	return false
}

// NewTestCase instantiates a TestCase based on the parameters specified
// The new testcase has three states by default.
// - Start state where the execution starts from
// - Fail state that can be used to fail the testcase
// - Success state that can be used to indicate a success of the testcase
func NewTestCase(name string, timeout time.Duration, sm *sm.StateMachine, cascade *FilterSet) *TestCase {
	cascade.addStateMachine(sm)
	return &TestCase{
		Name:         name,
		Timeout:      timeout,
		Cascade:      cascade,
		StateMachine: sm,
		Setup:        defaultSetupFunc,
		assertFn:     defaultAssertFunc,
		aborted:      false,
		doneCh:       types.NewChannel[bool](),
		once:         new(sync.Once),
	}
}

// End the testcase
func (t *TestCase) End() {
	t.once.Do(func() {
		t.doneCh.Close()
	})
}

// Abort the testcase
func (t *TestCase) Abort() {
	t.aborted = true
	t.once.Do(func() {
		t.doneCh.Close()
	})
}

// Step is called to execute a step of the testcase with a new event
func (t *TestCase) Step(e *types.Event, c *Context) ([]*types.Message, bool) {
	return t.Cascade.handleEvent(e, c)
}

// SetupFunc can be used to set the setup function
func (t *TestCase) SetupFunc(setupFunc func(*Context) error) {
	t.Setup = setupFunc
}

func (t *TestCase) assert() bool {
	return t.StateMachine.InSuccessState()
}

// Reset clears the stats of the filter set and allows running the testcase again
func (t *TestCase) Reset() {
	t.aborted = false
	t.once = new(sync.Once)
	t.doneCh.Reset()
	t.Cascade.resetStats()
}
