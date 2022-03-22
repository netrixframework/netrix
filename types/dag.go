package types

import (
	"encoding/json"
	"sync"
)

type EventNodeSet struct {
	nodes map[EventID]bool
	lock  *sync.Mutex
}

func NewEventNodeSet() *EventNodeSet {
	return &EventNodeSet{
		nodes: make(map[EventID]bool),
		lock:  new(sync.Mutex),
	}
}

func (d *EventNodeSet) Clone() *EventNodeSet {
	return &EventNodeSet{
		nodes: d.nodes,
		lock:  new(sync.Mutex),
	}
}

func (d *EventNodeSet) Add(nid EventID) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.nodes[nid] = true
}

func (d *EventNodeSet) Exists(nid EventID) bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	_, ok := d.nodes[nid]
	return ok
}

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

type EventNode struct {
	Event      *Event     `json:"event"`
	ClockValue ClockValue `json:"-"`
	prev       EventID
	next       EventID
	Parents    *EventNodeSet `json:"parents"`
	Children   *EventNodeSet `json:"children"`
	dirty      bool
	lock       *sync.Mutex
}

func NewEventNode(e *Event) *EventNode {
	return &EventNode{
		Event:      e,
		ClockValue: nil,
		prev:       0,
		next:       0,
		Parents:    NewEventNodeSet(),
		Children:   NewEventNodeSet(),
		dirty:      false,
		lock:       new(sync.Mutex),
	}
}

func (n *EventNode) Clone() *EventNode {
	return &EventNode{
		Event:      n.Event,
		ClockValue: n.ClockValue,
		prev:       n.prev,
		next:       n.next,
		Parents:    n.Parents.Clone(),
		Children:   n.Children.Clone(),
		dirty:      false,
		lock:       new(sync.Mutex),
	}
}

func (n *EventNode) SetClock(cv ClockValue) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.ClockValue = cv
}

func (n *EventNode) SetNext(next EventID) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.next = next
}

func (n *EventNode) GetNext() EventID {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.next
}

func (n *EventNode) SetPrev(prev EventID) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.prev = prev
}

func (n *EventNode) GetPrev() EventID {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.prev
}

func (n *EventNode) MarkDirty() {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.dirty = true
}

func (n *EventNode) MarkClean() {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.dirty = false
}

func (n *EventNode) AddParents(parents []*EventNode) {
	for _, p := range parents {
		n.Parents.Add(p.Event.ID)
		p.Children.Add(n.Event.ID)
	}
}

func (n *EventNode) IsDirty() bool {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.dirty
}

func (n *EventNode) Lt(other *EventNode) bool {
	if n.ClockValue == nil {
		return true
	}
	if other.ClockValue == nil {
		return false
	}
	return n.ClockValue.Lt(other.ClockValue)
}

type EventDAG struct {
	nodes   map[EventID]*EventNode
	strands map[ReplicaID]EventID
	latest  map[ReplicaID]EventID
	lock    *sync.Mutex

	latestClocks map[ReplicaID]ClockValue
	clockLock    *sync.Mutex
	replicaStore *ReplicaStore
}

func NewEventDag(replicaStore *ReplicaStore) *EventDAG {
	d := &EventDAG{
		nodes:   make(map[EventID]*EventNode),
		strands: make(map[ReplicaID]EventID),
		latest:  make(map[ReplicaID]EventID),
		lock:    new(sync.Mutex),

		latestClocks: make(map[ReplicaID]ClockValue),
		replicaStore: replicaStore,
		clockLock:    new(sync.Mutex),
	}
	for _, r := range replicaStore.Iter() {
		d.latestClocks[r.ID] = ZeroClock(replicaStore.Cap())
	}
	return d
}

func (d *EventDAG) nextClock(e *EventNode, parents []*EventNode) ClockValue {
	next := make([]float64, d.replicaStore.Cap())
	d.clockLock.Lock()
	latestClockValue := d.latestClocks[e.Event.Replica]
	d.clockLock.Unlock()
	for i, r := range d.replicaStore.Iter() {
		maxParent := float64(0)
		for _, p := range parents {
			if p.ClockValue[i] > maxParent {
				maxParent = p.ClockValue[i]
			}
		}
		if r.ID == e.Event.Replica && maxParent < latestClockValue[i]+1 {
			maxParent = latestClockValue[i] + 1
		}
		next[i] = maxParent
	}
	d.clockLock.Lock()
	d.latestClocks[e.Event.Replica] = next
	d.clockLock.Unlock()
	return ClockValue(next)
}

func (d *EventDAG) GetNode(eid EventID) (*EventNode, bool) {
	d.lock.Lock()
	defer d.lock.Unlock()

	node, ok := d.nodes[eid]
	return node, ok
}

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

func (d *EventDAG) GetSendNode(e *Event) (*Event, bool) {
	if !e.IsMessageReceive() {
		return nil, false
	}
	mesageID, _ := e.MessageID()
	d.lock.Lock()
	defer d.lock.Unlock()
	node, ok := d.nodes[e.ID]
	if !ok {
		return nil, false
	}
	for _, p := range node.Parents.Iter() {
		pN := d.nodes[p]
		if pN.Event.IsMessageSend() {
			mID, _ := pN.Event.MessageID()
			if mID == mesageID {
				return pN.Event, true
			}
		}
	}
	return nil, false
}

func (d *EventDAG) GetReceiveNode(e *Event) (*Event, bool) {
	if !e.IsMessageSend() {
		return nil, false
	}
	mesageID, _ := e.MessageID()
	d.lock.Lock()
	defer d.lock.Unlock()
	node, ok := d.nodes[e.ID]
	if !ok {
		return nil, false
	}
	for _, p := range node.Parents.Iter() {
		pN := d.nodes[p]
		if pN.Event.IsMessageReceive() {
			mID, _ := pN.Event.MessageID()
			if mID == mesageID {
				return pN.Event, true
			}
		}
	}
	return nil, false
}

func (d *EventDAG) GetTimeoutStart(e *Event) (*Event, bool) {
	if !e.IsTimeoutEnd() {
		return nil, false
	}
	timeout, _ := e.Timeout()
	d.lock.Lock()
	defer d.lock.Unlock()
	node, ok := d.nodes[e.ID]
	if !ok {
		return nil, false
	}
	for _, p := range node.Parents.Iter() {
		pN := d.nodes[p]
		if pN.Event.IsTimeoutStart() {
			t, _ := pN.Event.Timeout()
			if timeout.Eq(t) {
				return pN.Event, true
			}
		}
	}
	return nil, false
}

func (d *EventDAG) GetTimeoutEnd(e *Event) (*Event, bool) {
	if !e.IsTimeoutStart() {
		return nil, false
	}
	timeout, _ := e.Timeout()
	d.lock.Lock()
	defer d.lock.Unlock()
	node, ok := d.nodes[e.ID]
	if !ok {
		return nil, false
	}
	for _, p := range node.Parents.Iter() {
		pN := d.nodes[p]
		if pN.Event.IsTimeoutEnd() {
			t, _ := pN.Event.Timeout()
			if timeout.Eq(t) {
				return pN.Event, true
			}
		}
	}
	return nil, false
}

func (d *EventDAG) GetLatestNode(e *Event) (*Event, bool) {
	return nil, false
}
