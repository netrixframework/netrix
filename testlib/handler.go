package testlib

import "github.com/netrixframework/netrix/types"

// FilterFunc type to define a conditional handler
// returns false in the second return value if the handler is not concerned about the event
type FilterFunc func(*types.Event, *Context) ([]*types.Message, bool)

// FilterSet implements Handler
// Executes handlers in the specified order until the event is handled
// If no handler handles the event then the default handler is called
type FilterSet struct {
	Filters       []FilterFunc
	DefaultFilter FilterFunc
}

// FilterSetOption changes the parameters of the HandlerCascade
type FilterSetOption func(*FilterSet)

// WithDefault changes the HandlerCascade default handler
func WithDefault(d FilterFunc) FilterSetOption {
	return func(hc *FilterSet) {
		hc.DefaultFilter = d
	}
}

// NewFilterSet creates a new cascade handler with the specified state machine and options
func NewFilterSet(opts ...FilterSetOption) *FilterSet {
	h := &FilterSet{
		Filters:       make([]FilterFunc, 0),
		DefaultFilter: If(IsMessageSend()).Then(DeliverMessage()),
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

func (c *FilterSet) addStateMachine(sm *StateMachine) {
	c.Filters = append(
		[]FilterFunc{NewStateMachineHandler(sm)},
		c.Filters...,
	)
}

// AddFilter adds a handler to the cascade
func (c *FilterSet) AddFilter(h FilterFunc) {
	c.Filters = append(c.Filters, h)
}

// handleEvent implements Handler
func (c *FilterSet) handleEvent(e *types.Event, ctx *Context) []*types.Message {
	for _, h := range c.Filters {
		ret, ok := h(e, ctx)
		if ok {
			return ret
		}
	}
	ret, _ := c.DefaultFilter(e, ctx)
	return ret
}
