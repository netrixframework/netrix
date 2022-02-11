package types

import "sync"

// Rules to assign
// 	1. For event that isn't message (send,receive)
//		1.1. Find the nearest greater element that has non zero max value and set max to that
//		1.2. Find the nearest smaller element that has non zero min value and set min to that
// 	2. For timeout message send (receive),
//		2.1. If timeout end (start) is empty then determine by above logic
// 		2.2. If timeout end (start) is non empty then set min, max based on timeout end
// 	3. For other messages
// 		3.1. If message send and receive has non zero max, then set max to that otherwise above logic
// 		3.2. If message receive and send has non zero min, then set min to that otherwise above logic

// For timeouts

type EventNodeTagType = string

var (
	MessageSend    EventNodeTagType = "MessageSend"
	MessageReceive EventNodeTagType = "MessageReceive"
	TimeoutStart   EventNodeTagType = "TimeoutStart"
	TimeoutEnd     EventNodeTagType = "TimeoutEnd"
	Other          EventNodeTagType = "Other"
)

type IntW struct {
	v int
	d bool
}

func NewIntW() *IntW {
	return &IntW{
		d: false,
	}
}

func (i *IntW) Undefined() bool {
	return !i.d
}

func (i *IntW) Set(v int) {
	i.v = v
	i.d = false
}

type EventNodeTag struct {
	Event *Event
	Type  EventNodeTagType
	Min   *IntW
	Max   *IntW
}

func NewEventNodeTag(e *Event) *EventNodeTag {
	var t EventNodeTagType

	switch e.Type.(type) {
	case *MessageSendEventType:
		t = MessageSend
	case *MessageReceiveEventType:
		t = MessageReceive
	case *TimeoutStartEventType:
		t = TimeoutStart
	case *TimeoutEndEventType:
		t = TimeoutEnd
	default:
		t = Other
	}

	return &EventNodeTag{
		Event: e,
		Type:  t,
		Min:   NewIntW(),
		Max:   NewIntW(),
	}
}

type AnnotatedEventDAG struct {
	dag   *EventDAG
	start *EventNodeTag
	tags  map[uint64]*EventNodeTag
	lock  *sync.Mutex
}

func NewAnnotatedEventDag(d *EventDAG) *AnnotatedEventDAG {
	return &AnnotatedEventDAG{
		dag:   d.Clone(),
		start: nil,
		tags:  make(map[uint64]*EventNodeTag),
		lock:  new(sync.Mutex),
	}
}

func (a *AnnotatedEventDAG) SetFrameOfReference(e *Event) bool {
	_, ok := a.dag.GetNode(e.ID)
	if !ok {
		return false
	}

	start := NewEventNodeTag(e)
	start.Min.Set(0)
	start.Max.Set(0)
	a.lock.Lock()
	defer a.lock.Unlock()
	a.start = start
	a.tags[start.Event.ID] = start
	return true
}

func (a *AnnotatedEventDAG) AddEvent(e *Event, parents []*Event) {
	a.dag.AddNode(e, parents)
}

func (a *AnnotatedEventDAG) Check() bool {
	a.computeTags()
	a.check()
	return false
}

func (a *AnnotatedEventDAG) computeTags() {

}

func (a *AnnotatedEventDAG) check() {

}

// TODO:
// 1. Compute adjacency matrix to determine <, > 	| - can be part of EventDag instead of here for reusability
// 2. Compute pairs for send/receive events  	|
// 3. Compute min max max values from frame of reference.
