package types

import (
	"encoding/json"
	"sync"
)

type EventNodeSet struct {
	nodes map[uint64]bool
	lock  *sync.Mutex
}

func NewEventNodeSet() *EventNodeSet {
	return &EventNodeSet{
		nodes: make(map[uint64]bool),
		lock:  new(sync.Mutex),
	}
}

func (d *EventNodeSet) Clone() *EventNodeSet {
	return &EventNodeSet{
		nodes: d.nodes,
		lock:  new(sync.Mutex),
	}
}

func (d *EventNodeSet) Add(nid uint64) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.nodes[nid] = true
}

func (d *EventNodeSet) Exists(nid uint64) bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	_, ok := d.nodes[nid]
	return ok
}

func (d *EventNodeSet) Iter() []uint64 {
	d.lock.Lock()
	defer d.lock.Unlock()
	nodes := make([]uint64, len(d.nodes))
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
	nodes := make([]uint64, len(d.nodes))
	i := 0
	for n := range d.nodes {
		nodes[i] = n
		i = i + 1
	}
	d.lock.Unlock()
	return json.Marshal(nodes)
}

type EventNode struct {
	Event    *Event `json:"event"`
	prev     uint64
	next     uint64
	Parents  *EventNodeSet `json:"parents"`
	Children *EventNodeSet `json:"children"`
	dirty    bool
	lock     *sync.Mutex
}

func NewEventNode(e *Event) *EventNode {
	return &EventNode{
		Event:    e,
		prev:     0,
		next:     0,
		Parents:  NewEventNodeSet(),
		Children: NewEventNodeSet(),
		dirty:    false,
		lock:     new(sync.Mutex),
	}
}

func (n *EventNode) Clone() *EventNode {
	return &EventNode{
		Event:    n.Event,
		prev:     n.prev,
		next:     n.next,
		Parents:  n.Parents.Clone(),
		Children: n.Children.Clone(),
		dirty:    false,
		lock:     new(sync.Mutex),
	}
}

func (n *EventNode) SetNext(next uint64) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.next = next
}

func (n *EventNode) GetNext() uint64 {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.next
}

func (n *EventNode) SetPrev(prev uint64) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.prev = prev
}

func (n *EventNode) GetPrev() uint64 {
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

type EventDAG struct {
	nodes   map[uint64]*EventNode
	strands map[ReplicaID]uint64
	latest  map[ReplicaID]uint64
	lock    *sync.Mutex

	cleanCopies map[uint64]*EventNode
	cleanLock   *sync.Mutex
}

func NewEventDag() *EventDAG {
	return &EventDAG{
		nodes:   make(map[uint64]*EventNode),
		strands: make(map[ReplicaID]uint64),
		latest:  make(map[ReplicaID]uint64),
		lock:    new(sync.Mutex),

		cleanCopies: make(map[uint64]*EventNode),
		cleanLock:   new(sync.Mutex),
	}
}

func (d *EventDAG) GetNode(eid uint64) (*EventNode, bool) {
	d.lock.Lock()
	defer d.lock.Unlock()

	node, ok := d.nodes[eid]
	return node, ok
}

func (d *EventDAG) Clone() *EventDAG {
	nodes := make(map[uint64]*EventNode)
	d.lock.Lock()
	defer d.lock.Unlock()
	for nid, node := range d.nodes {
		nodes[nid] = node.Clone()
	}
	return &EventDAG{
		nodes:       nodes,
		strands:     d.strands,
		latest:      d.latest,
		lock:        new(sync.Mutex),
		cleanCopies: make(map[uint64]*EventNode),
		cleanLock:   new(sync.Mutex),
	}
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

	_, ok = d.strands[e.Replica]
	if !ok {
		d.strands[e.Replica] = e.ID
	}
}

func (d *EventDAG) markDirty(n *EventNode) {
	d.cleanLock.Lock()
	d.cleanCopies[n.Event.ID] = n.Clone()
	d.cleanLock.Unlock()
	n.MarkDirty()
}

func (d *EventDAG) AddDirty(e *Event, parents []*Event) {
	parentNodes := make([]*EventNode, len(parents))
	d.lock.Lock()
	defer d.lock.Unlock()
	for _, p := range parents {
		pN, ok := d.nodes[p.ID]
		if ok {
			parentNodes = append(parentNodes, pN)
			d.markDirty(pN)
		}
	}
	node := NewEventNode(e)
	node.MarkDirty()

	l, ok := d.latest[e.Replica]
	if ok {
		lN := d.nodes[l]
		lN.SetNext(e.ID)
		node.SetPrev(lN.Event.ID)
		parentNodes = append(parentNodes, lN)
		d.markDirty(lN)
	}
	d.latest[e.Replica] = e.ID

	node.AddParents(parentNodes)
	_, ok = d.strands[e.Replica]
	if !ok {
		d.strands[e.Replica] = e.ID
	}
}

func (d *EventDAG) Clean() {
	dirtyNodes := make([]uint64, 0)
	d.lock.Lock()
	for id, node := range d.nodes {
		if node.IsDirty() {
			dirtyNodes = append(dirtyNodes, id)
		}
	}
	d.lock.Unlock()
	d.cleanLock.Lock()
	for _, dID := range dirtyNodes {
		if cleanCopy, ok := d.cleanCopies[dID]; ok {
			d.nodes[dID] = cleanCopy
			delete(d.cleanCopies, dID)
		} else {
			delete(d.nodes, dID)
		}
	}
	d.cleanLock.Unlock()
}

func (d *EventDAG) MarshalJSON() ([]byte, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	keyvals := make(map[string]interface{})
	nodes := make(map[uint64]*EventNode, len(d.nodes))

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
