package testlib

import (
	"encoding/json"

	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
)

// FilterFunc type to define a conditional handler
// returns false in the second return value if the handler is not concerned about the event
type FilterFunc func(*types.Event, *Context) ([]*types.Message, bool)

// FilterSetStats keeps a count of the number of times a filter condition was satisfied for a given filter
// The filter index is the order in which the filter is specified.
type FilterSetStats struct {
	filterCount        map[int]int
	defaultFilterCount int
}

func (s *FilterSetStats) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toMap())
}

func (s *FilterSetStats) String() string {
	out, _ := json.Marshal(s.toMap())
	return string(out)
}

func (s *FilterSetStats) toMap() map[string]interface{} {
	data := make(map[string]interface{})
	data["default"] = s.defaultFilterCount
	data["filters"] = s.filterCount
	return data
}

func (s *FilterSetStats) addCount(index int) {
	cur, ok := s.filterCount[index]
	if !ok {
		s.filterCount[index] = 1
	} else {
		s.filterCount[index] = cur + 1
	}
}

func (s *FilterSetStats) addDefaultCount() {
	s.defaultFilterCount += 1
}

func (s *FilterSetStats) reset() {
	s.filterCount = make(map[int]int)
	s.defaultFilterCount = 0
}

// FilterSet implements Handler
// Executes handlers in the specified order until the event is handled
// If no handler handles the event then the default handler is called
type FilterSet struct {
	Filters       []FilterFunc
	DefaultFilter FilterFunc

	stats *FilterSetStats
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
		DefaultFilter: If(sm.IsMessageSend()).Then(DeliverMessage()),
		stats: &FilterSetStats{
			filterCount:        make(map[int]int),
			defaultFilterCount: 0,
		},
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

func (c *FilterSet) addStateMachine(sm *sm.StateMachine) {
	c.Filters = append(
		[]FilterFunc{NewStateMachineHandler(sm)},
		c.Filters...,
	)
}

// AddFilter adds a handler to the cascade
func (c *FilterSet) AddFilter(h FilterFunc) {
	c.Filters = append(c.Filters, h)
}

// Stats returns the stats
func (c *FilterSet) Stats() *FilterSetStats {
	return c.stats
}

func (c *FilterSet) resetStats() {
	c.stats.reset()
}

// handleEvent implements Handler
func (c *FilterSet) handleEvent(e *types.Event, ctx *Context) ([]*types.Message, bool) {
	for i, h := range c.Filters {
		ret, ok := h(e, ctx)
		if ok {
			c.stats.addCount(i)
			return ret, true
		}
	}
	ret, _ := c.DefaultFilter(e, ctx)
	c.stats.addDefaultCount()
	return ret, false
}
