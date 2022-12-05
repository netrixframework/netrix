package types

import (
	"encoding/json"
	"sync"
)

// EventNodeSet is a thread safe node set.
type EventNodeSet struct {
	nodes map[EventID]bool
	lock  *sync.Mutex
}

// NewEventNodeSet creates an empty [EventNodeSet]
func NewEventNodeSet() *EventNodeSet {
	return &EventNodeSet{
		nodes: make(map[EventID]bool),
		lock:  new(sync.Mutex),
	}
}

// Clone returns a copy of the event node set
func (d *EventNodeSet) Clone() *EventNodeSet {
	return &EventNodeSet{
		nodes: d.nodes,
		lock:  new(sync.Mutex),
	}
}

// Add adds the event to the set
func (d *EventNodeSet) Add(nid EventID) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.nodes[nid] = true
}

// Exists returns true if the node is part of the set, false otherwise
func (d *EventNodeSet) Exists(nid EventID) bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	_, ok := d.nodes[nid]
	return ok
}

// Iter returns a list of the event node in a random order
func (d *EventNodeSet) Iter() []EventID {
	d.lock.Lock()
	defer d.lock.Unlock()
	nodes := make([]EventID, len(d.nodes))
	i := 0
	for k := range d.nodes {
		nodes[i] = k
		i = i + 1
	}
	return nodes
}

// Size returns the size of the EventNodeSet
func (d *EventNodeSet) Size() int {
	d.lock.Lock()
	defer d.lock.Unlock()
	return len(d.nodes)
}

func (d *EventNodeSet) MarshalJSON() ([]byte, error) {
	d.lock.Lock()
	nodes := make([]EventID, len(d.nodes))
	i := 0
	for n := range d.nodes {
		nodes[i] = n
		i = i + 1
	}
	d.lock.Unlock()
	return json.Marshal(nodes)
}

// EventNode encapsulates an [Event] with information about parents, children, clock value.
// Represents a node in the EventDAG.
type EventNode struct {
	Event      *Event     `json:"event"`
	ClockValue ClockValue `json:"-"`
	prev       EventID
	next       EventID
	Parents    *EventNodeSet `json:"parents"`
	Children   *EventNodeSet `json:"children"`
	lock       *sync.Mutex
}

// NewEventNode creates a new [EventNode]
func NewEventNode(e *Event) *EventNode {
	return &EventNode{
		Event:      e,
		ClockValue: nil,
		prev:       0,
		next:       0,
		Parents:    NewEventNodeSet(),
		Children:   NewEventNodeSet(),
		lock:       new(sync.Mutex),
	}
}

// Clone returns a copy of the event node
func (n *EventNode) Clone() *EventNode {
	return &EventNode{
		Event:      n.Event,
		ClockValue: n.ClockValue,
		prev:       n.prev,
		next:       n.next,
		Parents:    n.Parents.Clone(),
		Children:   n.Children.Clone(),
		lock:       new(sync.Mutex),
	}
}

// SetClock sets the vector clock value
func (n *EventNode) SetClock(cv ClockValue) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.ClockValue = cv
}

// SetNext sets the next event for the current node
func (n *EventNode) SetNext(next EventID) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.next = next
}

// GetNext returns the next event in the same replica
func (n *EventNode) GetNext() EventID {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.next
}

// SetPrev sets the previous node for the current node
func (n *EventNode) SetPrev(prev EventID) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.prev = prev
}

// GetPrev returns the previous node for the same replica
func (n *EventNode) GetPrev() EventID {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.prev
}

// AddParents add to the Parents event node set.
func (n *EventNode) AddParents(parents []*EventNode) {
	for _, p := range parents {
		n.Parents.Add(p.Event.ID)
		p.Children.Add(n.Event.ID)
	}
}

// Lt returns true when the underlying Events are related by cur < other
func (n *EventNode) Lt(other *EventNode) bool {
	if n.ClockValue == nil {
		return true
	}
	if other.ClockValue == nil {
		return false
	}
	return n.ClockValue.Lt(other.ClockValue)
}

// EventDAG is directed acyclic graph (DAG) of the events.
//
// The DAG can be traversed and also used to compare two events
// The nodes of the DAG are represented by [EventNode] which contains information about parents and children.
// Recursively enumerating the parents, children of the [EventNode] allows one to traverse the DAG
type EventDAG struct {
	nodes         map[EventID]*EventNode
	strands       map[ReplicaID]EventID
	latest        map[ReplicaID]EventID
	sends         map[MessageID]*EventNode
	timeoutStarts map[string]*EventNode
	lock          *sync.Mutex

	latestClocks map[ReplicaID]ClockValue
	clockLock    *sync.Mutex
	replicaStore *ReplicaStore
}

// NewEventDag creates a new [EventDAG]
func NewEventDag(replicaStore *ReplicaStore) *EventDAG {
	d := &EventDAG{
		nodes:         make(map[EventID]*EventNode),
		strands:       make(map[ReplicaID]EventID),
		latest:        make(map[ReplicaID]EventID),
		sends:         make(map[MessageID]*EventNode),
		timeoutStarts: make(map[string]*EventNode),
		lock:          new(sync.Mutex),

		latestClocks: make(map[ReplicaID]ClockValue),
		replicaStore: replicaStore,
		clockLock:    new(sync.Mutex),
	}
	for _, r := range replicaStore.Iter() {
		d.latestClocks[r.ID] = ZeroClock()
	}
	return d
}

// Reset clears the DAG
func (d *EventDAG) Reset() {
	d.lock.Lock()
	d.nodes = make(map[EventID]*EventNode)
	d.strands = make(map[ReplicaID]EventID)
	d.latest = make(map[ReplicaID]EventID)
	d.sends = make(map[MessageID]*EventNode)
	d.timeoutStarts = make(map[string]*EventNode)

	d.lock.Unlock()
	d.clockLock.Lock()
	d.latestClocks = make(map[ReplicaID]ClockValue)
	for _, r := range d.replicaStore.Iter() {
		d.latestClocks[r.ID] = ZeroClock()
	}
	d.clockLock.Unlock()
}

func (d *EventDAG) nextClock(e *EventNode, parents []*EventNode) ClockValue {
	next := make(map[ReplicaID]int)
	d.clockLock.Lock()
	latestClockValue := d.latestClocks[e.Event.Replica]
	d.clockLock.Unlock()
	latestClockValue = latestClockValue.Next(e.Event.Replica)

	clockValsToCmp := []ClockValue{latestClockValue}
	for _, p := range parents {
		clockValsToCmp = append(clockValsToCmp, p.ClockValue)
	}

	for _, r := range d.replicaStore.Iter() {
		maxParent := 0
		for _, c := range clockValsToCmp {
			parentVal, ok := c[r.ID]
			if ok && parentVal > maxParent {
				maxParent = c[r.ID]
			}
		}
		next[r.ID] = maxParent
	}

	d.clockLock.Lock()
	d.latestClocks[e.Event.Replica] = next
	d.clockLock.Unlock()
	return ClockValue(next)
}

// GetNode returns the [EventNode] of the corresponding event when it is part of the DAG,
// the second parameter is false otherwise.
func (d *EventDAG) GetNode(eid EventID) (*EventNode, bool) {
	d.lock.Lock()
	defer d.lock.Unlock()

	node, ok := d.nodes[eid]
	return node, ok
}

// AddNode adds a new event with the specified parents to the DAG
//
// Internally, the parents are inferred as follows.
// 1. The previous node in the same replica
// 2. The message send node when the new event is a message receive
// 3. The timeout start node when the new event is a timeout end
func (d *EventDAG) AddNode(e *Event, parents []*Event) {
	parentNodes := make([]*EventNode, 0)
	d.lock.Lock()
	defer d.lock.Unlock()
	for _, p := range parents {
		pN, ok := d.nodes[p.ID]
		if ok {
			parentNodes = append(parentNodes, pN)
		}
	}
	node := NewEventNode(e)
	d.nodes[e.ID] = node

	l, ok := d.latest[e.Replica]
	if ok {
		lN := d.nodes[l]
		lN.SetNext(e.ID)
		node.SetPrev(lN.Event.ID)
		parentNodes = append(parentNodes, lN)
	}
	d.latest[e.Replica] = e.ID

	if e.IsMessageSend() {
		messageID, _ := e.MessageID()
		d.sends[messageID] = node
	} else if e.IsMessageReceive() {
		messageID, _ := e.MessageID()
		send, ok := d.sends[messageID]
		if ok {
			parentNodes = append(parentNodes, send)
			delete(d.sends, messageID)
		}
	}

	if e.IsTimeoutStart() {
		timeout, _ := e.Timeout()
		d.timeoutStarts[timeout.Key()] = node
	} else if e.IsTimeoutEnd() {
		timeout, _ := e.Timeout()
		start, ok := d.timeoutStarts[timeout.Key()]
		if ok {
			parentNodes = append(parentNodes, start)
			delete(d.timeoutStarts, timeout.Key())
		}
	}

	node.AddParents(parentNodes)
	node.SetClock(d.nextClock(node, parentNodes))

	_, ok = d.strands[e.Replica]
	if !ok {
		d.strands[e.Replica] = e.ID
	}
}

func (d *EventDAG) MarshalJSON() ([]byte, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	keyvals := make(map[string]interface{})
	nodes := make(map[EventID]*EventNode, len(d.nodes))

	for id, node := range d.nodes {
		nodes[id] = node
	}

	keyvals["nodes"] = nodes
	keyvals["strands"] = d.strands
	return json.MarshalIndent(keyvals, "", "\t")
}

// GetLatestNode returns the latest event for the specified replica if one exists.
func (d *EventDAG) GetLatestNode(replica ReplicaID) (*Event, bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	eid, ok := d.latest[replica]
	if !ok {
		return nil, false
	}
	eventNode, ok := d.nodes[eid]
	if !ok {
		return nil, false
	}
	return eventNode.Event, true
}
