package pct

import (
	"fmt"

	"github.com/netrixframework/netrix/types"
)

type pctMessage struct {
	messageID types.MessageID
	from      types.ReplicaID
	to        types.ReplicaID
	label     int
}

// Lt returns true if e < other in the given message order
func (e *pctMessage) Lt(mo MessageOrder, other *pctMessage) bool {
	return mo.Lt(e.messageID, other.messageID)
}

func (e *pctMessage) HasLabel() bool {
	return e.label != -1
}

func (e *pctMessage) Label() int {
	return e.label
}

func newPCTMessage(message *types.Message) *pctMessage {
	return &pctMessage{
		messageID: message.ID,
		to:        message.To,
		from:      message.From,
		label:     -1,
	}
}

// MessageOrder interface defines the message order for PCTCP to partition messages into chains
type MessageOrder interface {
	// AddSendEvent is triggered when a send event of the corresponding message is processed
	AddSendEvent(*types.Message)
	// AddRecvEvent is triggered when a receive event of the corresponding message is processed
	AddRecvEvent(*types.Message)
	// Lt should return true if m_1 < m_2
	Lt(types.MessageID, types.MessageID) bool
	// Reset forgets all stored information and is invoked at the end of each iteration
	Reset()
}

type vcValue struct {
	vals map[types.ReplicaID]int
}

func defaultVCValue(replica types.ReplicaID) *vcValue {
	val := &vcValue{
		vals: make(map[types.ReplicaID]int),
	}
	val.vals[replica] = 0
	return val
}

// Lt Returns true if v < other
func (v *vcValue) Lt(other *vcValue) bool {
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

func (v *vcValue) Next(replica types.ReplicaID) *vcValue {
	new := &vcValue{
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

func (v *vcValue) String() string {
	result := "{ "
	for r, v := range v.vals {
		result += fmt.Sprintf("%s:%d ", r, v)
	}
	result += "}"
	return result
}

// DefaultMessageOrder implements [MessageOrder] and tracks the causal ordering of messages.
// Two messages m_1 and m_2 are related by m_1 < m_2 iff recv(m_1) < send(m_2)
type DefaultMessageOrder struct {
	latest       *types.Map[types.ReplicaID, *vcValue]
	sendVCValues *types.Map[types.MessageID, *vcValue]
	recvVCValues *types.Map[types.MessageID, *vcValue]
}

var _ MessageOrder = &DefaultMessageOrder{}

func NewDefaultMessageOrder() *DefaultMessageOrder {
	return &DefaultMessageOrder{
		latest:       types.NewMap[types.ReplicaID, *vcValue](),
		sendVCValues: types.NewMap[types.MessageID, *vcValue](),
		recvVCValues: types.NewMap[types.MessageID, *vcValue](),
	}
}

func (eo *DefaultMessageOrder) AddSendEvent(e *types.Message) {
	if eo.sendVCValues.Exists(e.ID) {
		return
	}

	latest, ok := eo.latest.Get(e.From)
	if !ok {
		vcVal := defaultVCValue(e.From)
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

	sendVCValue, sendExists := eo.sendVCValues.Get(e.ID)

	cur, ok := eo.latest.Get(e.To)
	if !ok {
		cur = defaultVCValue(e.To)
	} else {
		cur = cur.Next(e.To)
	}

	recvVCValue := &vcValue{vals: make(map[types.ReplicaID]int)}
	replicas := make(map[types.ReplicaID]bool)
	if sendExists {
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
	} else {
		for replica, val := range cur.vals {
			recvVCValue.vals[replica] = val
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
