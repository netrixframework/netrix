package fuzzing

import (
	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type ActionType string

var (
	DeliverAction ActionType = "Deliver"
	DropAction    ActionType = "Drop"
)

type Action struct {
	Type    ActionType
	Process types.ReplicaID
}

type Input []Action

func (i *Input) Hash() string {
	return ""
}

type Mutator interface {
	Mutate(*Input) *Input
}

type State interface {
	Hash() string
}

type Interpreter interface {
	CurState() State
	Reset()
	Update(*types.Event, *strategies.Context)
}
