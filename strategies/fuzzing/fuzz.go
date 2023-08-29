package fuzzing

import (
	"math/rand"
	"sync"
	"time"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
)

type FuzzStrategyConfig struct {
	Mutator           Mutator
	Guider            Guider
	TickDuration      time.Duration
	Steps             int
	Seed              rand.Source
	InitialPopulation int
	MutationsPerTrace int
}

// Coverage guided fuzzing strategy
// TODO: goals
// 1. Allow plugging in different mutation strategies
// 2. Allow measuring state coverage using a guider
// 3. Maintain corpus of successful mutations

// Main idea - the input is a sequence of actions.
// An action can be one of - Deliver to Process p, Drop to process p
// Generate a random input and use quick mutations to improve coverage
type FuzzStrategy struct {
	*types.BaseService
	actions *types.Channel[*strategies.Action]

	mailBoxes *types.Map[types.ReplicaID, []types.MessageID]
	corpus    *types.Queue[*Trace]

	replicaStore  *types.ReplicaStore
	curTrace      *types.List[*SchedulingChoice]
	curEventTrace *types.List[*types.Event]

	mutatedNodeChoices *types.Queue[types.ReplicaID]

	fuzzEnabled bool
	lock        *sync.Mutex

	mutator Mutator
	guider  Guider
	config  *FuzzStrategyConfig
}

var _ strategies.Strategy = &FuzzStrategy{}

func NewFuzzStrategy(config *FuzzStrategyConfig) *FuzzStrategy {
	return &FuzzStrategy{
		BaseService: types.NewBaseService("FuzzStrategy", nil),
		actions:     types.NewChannel[*strategies.Action](),

		mailBoxes:    types.NewMap[types.ReplicaID, []types.MessageID](),
		corpus:       types.NewQueue[*Trace](),
		mutator:      config.Mutator,
		guider:       config.Guider,
		config:       config,
		fuzzEnabled:  false,
		lock:         new(sync.Mutex),
		replicaStore: nil,

		curTrace:           types.NewEmptyList[*SchedulingChoice](),
		curEventTrace:      types.NewEmptyList[*types.Event](),
		mutatedNodeChoices: types.NewQueue[types.ReplicaID](),
	}
}

func (f *FuzzStrategy) ActionsCh() *types.Channel[*strategies.Action] {
	return f.actions
}

func (f *FuzzStrategy) EndCurIteration(ctx *strategies.Context) {
	f.lock.Lock()
	f.fuzzEnabled = false
	f.lock.Unlock()

	if f.guider.HaveNewState(f.curTrace, f.curEventTrace) {
		for i := 0; i < f.config.MutationsPerTrace; i++ {
			newTrace, ok := f.mutator.Mutate(f.curTrace, f.curEventTrace)
			if ok {
				f.corpus.Add(&Trace{
					List: newTrace,
				})
			}
		}
	}
}

func (f *FuzzStrategy) NextIteration(ctx *strategies.Context) {
	if f.replicaStore == nil {
		f.replicaStore = ctx.ReplicaStore
	}
	f.lock.Lock()
	f.fuzzEnabled = true
	f.lock.Unlock()

	// Reset current traces
	f.curTrace.RemoveAll()
	f.curEventTrace.RemoveAll()
	for _, k := range f.mailBoxes.Keys() {
		f.mailBoxes.Add(k, make([]types.MessageID, 0))
	}

	// Check if we are still populating the initial population
	if ctx.CurIteration() < f.config.InitialPopulation {
		// Nothing to do
		return
	}

	// Pick the next trace from the corpus and set the current mutated trace to it
	f.mutatedNodeChoices.Reset()
	if t, ok := f.corpus.Pop(); ok {
		for _, choice := range t.Iter() {
			switch choice.Type {
			case ScheduleProcess:
				f.mutatedNodeChoices.Add(choice.Process)
			}
		}
	}
}

func (f *FuzzStrategy) Finalize(ctx *strategies.Context) {
	// Record metrics
}

func (f *FuzzStrategy) Start() error {
	f.BaseService.StartRunning()
	return nil
}

func (f *FuzzStrategy) Stop() error {
	f.BaseService.StopRunning()
	return nil
}

func (f *FuzzStrategy) Step(e *types.Event, ctx *strategies.Context) {
	// 1. Record event
	f.curEventTrace.Append(e)

	// 2. Add message to mailbox if the event is a message send
	if e.IsMessageSend() {
		if m, ok := ctx.GetMessage(e); ok {
			if !f.mailBoxes.Exists(m.To) {
				f.mailBoxes.Add(m.To, make([]types.MessageID, 0))
			}
			curMailbox, _ := f.mailBoxes.Get(m.To)
			curMailbox = append(curMailbox, m.ID)
			f.mailBoxes.Add(m.To, curMailbox)
		}

	}

	// 3. Check if there is anything to schedule
	if choice, ok := f.mutatedNodeChoices.Pop(); ok {
		mailbox, ok := f.mailBoxes.Get(choice)
		if ok && len(mailbox) > 0 {
			nextMessageID := mailbox[0]
			mailbox = mailbox[1:]
			f.mailBoxes.Add(choice, mailbox)
			m, ok := ctx.MessagePool.Get(nextMessageID)
			if ok {
				f.actions.BlockingAdd(strategies.DeliverMessage(m))
			}
		}
		// TODO: Need to record this choice in curTrace
	} else {
		// TODO: Else pick random node and schedule
	}
}
