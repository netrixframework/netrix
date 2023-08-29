package fuzzing

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/netrixframework/netrix/types"
)

type DefaultMutator struct {
}

var _ Mutator = &DefaultMutator{}

func NewDefaultMutator() *DefaultMutator {
	return &DefaultMutator{}
}

func (d *DefaultMutator) Mutate(_ *types.List[*SchedulingChoice], _ *types.List[*types.Event]) (*types.List[*SchedulingChoice], bool) {
	return nil, false
}

type SwapNodesMutator struct {
	NumSwaps int
	rand     *rand.Rand
}

func NewSwapNodeMutator(numSwaps int) *SwapNodesMutator {
	return &SwapNodesMutator{
		NumSwaps: numSwaps,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

var _ Mutator = &SwapNodesMutator{}

func (s *SwapNodesMutator) Mutate(trace *types.List[*SchedulingChoice], _ *types.List[*types.Event]) (*types.List[*SchedulingChoice], bool) {
	nodeChoiceIndices := make([]int, 0)
	for i, choice := range trace.Iter() {
		if choice.Type == ScheduleProcess {
			nodeChoiceIndices = append(nodeChoiceIndices, i)
		}
	}
	numNodeChoiceIndices := len(nodeChoiceIndices)
	if numNodeChoiceIndices == 0 {
		return nil, false
	}
	choices := numNodeChoiceIndices
	if s.NumSwaps < choices {
		choices = s.NumSwaps
	}
	toSwap := make(map[string]map[int]int)
	for len(toSwap) < choices {
		i := nodeChoiceIndices[s.rand.Intn(numNodeChoiceIndices)]
		j := nodeChoiceIndices[s.rand.Intn(numNodeChoiceIndices)]
		key := fmt.Sprintf("%d_%d", i, j)
		if _, ok := toSwap[key]; !ok {
			toSwap[key] = map[int]int{i: j}
		}
	}
	newTrace := copyTrace(trace)
	for _, v := range toSwap {
		for i, j := range v {
			first, _ := newTrace.Elem(i)
			second, _ := newTrace.Elem(j)
			newTrace.Set(i, second)
			newTrace.Set(j, first)
		}
	}
	return newTrace, true
}

func copyTrace(trace *types.List[*SchedulingChoice]) *types.List[*SchedulingChoice] {
	newT := types.NewEmptyList[*SchedulingChoice]()
	for _, e := range trace.Iter() {
		newT.Append(&SchedulingChoice{
			Type:    e.Type,
			Process: e.Process,
			Bool:    e.Bool,
			Integer: e.Integer,
		})
	}
	return newT
}
