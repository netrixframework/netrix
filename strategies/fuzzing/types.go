package fuzzing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/netrixframework/netrix/types"
)

type ScheduleChoiceType string

var (
	ScheduleProcess ScheduleChoiceType = "Process"
	BooleanNonDet   ScheduleChoiceType = "Boolean"
	IntegerNonDet   ScheduleChoiceType = "Integer"
)

type SchedulingChoice struct {
	Type    ScheduleChoiceType
	Process types.ReplicaID
	Bool    bool
	Integer int
}

type Guider interface {
	HaveNewState(*types.List[*SchedulingChoice], *types.List[*types.Event]) bool
	Reset()
}

type Mutator interface {
	Mutate(*types.List[*SchedulingChoice], *types.List[*types.Event]) (*types.List[*SchedulingChoice], bool)
}

type Trace struct {
	*types.List[*SchedulingChoice]
}

func NewTrace() *Trace {
	return &Trace{
		List: types.NewEmptyList[*SchedulingChoice](),
	}
}

func (t *Trace) Hash() string {
	bs, _ := json.Marshal(t.List)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (t *Trace) Reset() {
	t.RemoveAll()
}
