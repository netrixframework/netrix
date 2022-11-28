package rl

import (
	"encoding/json"
	"hash/fnv"
	"math"
	"sync"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/types"
	"gonum.org/v1/gonum/stat/sampleuv"
)

type SoftMaxPolicy struct {
}

var _ Policy = &SoftMaxPolicy{}

func NewSoftMaxPolicy() *SoftMaxPolicy {
	return &SoftMaxPolicy{}
}

func (p *SoftMaxPolicy) NextAction(saMap *StateActionMap, state State, actions []*strategies.Action) (*strategies.Action, bool) {
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))

	var sum float64 = 0
	for i, action := range actions {
		val, _ := saMap.Get(state.Hash(), action.Name)
		exp := math.Exp(val)
		vals[i] = exp
		sum += exp
	}

	for i, v := range vals {
		weights[i] = v / sum
	}
	i, ok := sampleuv.NewWeighted(weights, nil).Take()
	if !ok {
		return nil, false
	}
	return actions[i], true
}

type SimpleInterpreter struct {
	curMailBox      map[string][]string
	pendingMessages map[string]bool
	lock            *sync.Mutex
}

var _ Interpreter = &SimpleInterpreter{}

func NewSimpleInterpreter() *SimpleInterpreter {
	return &SimpleInterpreter{
		curMailBox:      make(map[string][]string),
		pendingMessages: make(map[string]bool),
		lock:            new(sync.Mutex),
	}
}

func (s *SimpleInterpreter) RewardFunc(_ State, _ *strategies.Action) float64 {
	return -1
}

func (s *SimpleInterpreter) Interpret(e *types.Event, ctx *strategies.Context) State {
	s.lock.Lock()
	defer s.lock.Unlock()
	if e.IsMessageReceive() {
		messageID, _ := e.MessageID()
		_, ok := s.curMailBox[string(e.Replica)]
		if !ok {
			s.curMailBox[string(e.Replica)] = make([]string, 0)
		}
		s.curMailBox[string(e.Replica)] = append(s.curMailBox[string(e.Replica)], string(messageID))
	}
	if e.IsMessageSend() {
		messageID, _ := e.MessageID()
		s.pendingMessages[string(messageID)] = true
	}

	return &SimpleInterpreterState{
		mailBox:         s.curMailBox,
		pendingMessages: s.pendingMessages,
	}
}

func (s *SimpleInterpreter) Actions(state State, ctx *strategies.Context) []*strategies.Action {
	actions := make([]*strategies.Action, 0)
	simpState, ok := state.(*SimpleInterpreterState)
	if !ok {
		return actions
	}

	for messageID := range simpState.pendingMessages {
		message, ok := ctx.MessagePool.Get(types.MessageID(messageID))
		if ok {
			actions = append(actions, strategies.DeliverMessage(message))
		}
	}

	return actions
}

type SimpleInterpreterState struct {
	mailBox         map[string][]string
	pendingMessages map[string]bool
}

func (s *SimpleInterpreterState) Hash() string {
	bs, _ := json.Marshal(s.mailBox)
	h := fnv.New64a()
	h.Write(bs)
	return string(h.Sum(nil))
}
