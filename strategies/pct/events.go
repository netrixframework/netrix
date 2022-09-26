package pct

import (
	"fmt"

	"github.com/netrixframework/netrix/types"
)

type Message struct {
	messageID types.MessageID
	from      types.ReplicaID
	to        types.ReplicaID
	label     int
}

// Lt returns true if e < other in the given message order
func (e *Message) Lt(mo MessageOrder, other *Message) bool {
	return mo.Lt(e.messageID, other.messageID)
}

func (e *Message) HasLabel() bool {
	return e.label != -1
}

func (e *Message) Label() int {
	return e.label
}

func NewMessage(message *types.Message) *Message {
	return &Message{
		messageID: message.ID,
		to:        message.To,
		from:      message.From,
		label:     -1,
	}
}

type MessageOrder interface {
	AddSendEvent(*types.Message)
	AddRecvEvent(*types.Message)
	Lt(types.MessageID, types.MessageID) bool
	Reset()
}

type VCValue struct {
	vals map[types.ReplicaID]int
}

func DefaultVCValue(replica types.ReplicaID) *VCValue {
	val := &VCValue{
		vals: make(map[types.ReplicaID]int),
	}
	val.vals[replica] = 0
	return val
}

// Lt Returns true if v < other
func (v *VCValue) Lt(other *VCValue) bool {
	oneLess := false
	for r, val := range v.vals {
		valO, ok := other.vals[r]
		if !ok || val > valO {
			return false
		}
		if val < valO {
			oneLess = true
		}
	}
	return oneLess
}

func (v *VCValue) Next(replica types.ReplicaID) *VCValue {
	new := &VCValue{
		vals: make(map[types.ReplicaID]int),
	}
	for r, val := range v.vals {
		new.vals[r] = val
	}

	cur, ok := v.vals[replica]
	if !ok {
		new.vals[replica] = 0
	} else {
		new.vals[replica] = cur + 1
	}
	return new
}

func (v *VCValue) String() string {
	result := "{ "
	for r, v := range v.vals {
		result += fmt.Sprintf("%s:%d ", r, v)
	}
	result += "}"
	return result
}

type DefaultMessageOrder struct {
	latest       *types.Map[types.ReplicaID, *VCValue]
	sendVCValues *types.Map[types.MessageID, *VCValue]
	recvVCValues *types.Map[types.MessageID, *VCValue]
}

var _ MessageOrder = &DefaultMessageOrder{}

func NewDefaultMessageOrder() *DefaultMessageOrder {
	return &DefaultMessageOrder{
		latest:       types.NewMap[types.ReplicaID, *VCValue](),
		sendVCValues: types.NewMap[types.MessageID, *VCValue](),
		recvVCValues: types.NewMap[types.MessageID, *VCValue](),
	}
}

func (eo *DefaultMessageOrder) AddSendEvent(e *types.Message) {
	if eo.sendVCValues.Exists(e.ID) {
		return
	}

	latest, ok := eo.latest.Get(e.From)
	if !ok {
		vcVal := DefaultVCValue(e.From)
		eo.latest.Add(e.From, vcVal)
		eo.sendVCValues.Add(e.ID, vcVal)
		return
	}
	next := latest.Next(e.From)
	eo.sendVCValues.Add(e.ID, next)
	eo.latest.Add(e.From, next)
}

func (eo *DefaultMessageOrder) AddRecvEvent(e *types.Message) {
	if eo.recvVCValues.Exists(e.ID) {
		return
	}

	sendVCValue, _ := eo.sendVCValues.Get(e.ID)

	cur, ok := eo.latest.Get(e.To)
	if !ok {
		cur = DefaultVCValue(e.To)
	} else {
		cur = cur.Next(e.To)
	}

	recvVCValue := &VCValue{vals: make(map[types.ReplicaID]int)}
	replicas := make(map[types.ReplicaID]bool)
	for replica := range sendVCValue.vals {
		replicas[replica] = true
	}
	for replica := range cur.vals {
		replicas[replica] = true
	}
	for r := range replicas {
		v1, ok1 := sendVCValue.vals[r]
		v2, ok2 := cur.vals[r]
		if !ok1 && ok2 {
			recvVCValue.vals[r] = v2
		} else if !ok2 && ok1 {
			recvVCValue.vals[r] = v1
		} else if ok1 && ok2 {
			recvVCValue.vals[r] = types.Max(v1, v2)
		}
	}

	eo.latest.Add(e.To, recvVCValue)
	eo.recvVCValues.Add(e.ID, recvVCValue)
}

func (eo *DefaultMessageOrder) Lt(e1, e2 types.MessageID) bool {
	vc1, ok := eo.recvVCValues.Get(e1)
	if !ok {
		return false
	}
	vc2, ok := eo.sendVCValues.Get(e2)
	if !ok {
		return false
	}
	return vc1.Lt(vc2)
}

func (eo *DefaultMessageOrder) Reset() {
	eo.latest.RemoveAll()
	eo.recvVCValues.RemoveAll()
	eo.sendVCValues.RemoveAll()
}
