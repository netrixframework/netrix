package visualizer

import (
	"sync"

	"github.com/netrixframework/netrix/apiserver"
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/dispatcher"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

// Visualizer is one of the modes of executing the tool
// where the user can control the messages delivery order
// through an interactive dashboard
type Visualizer struct {
	apiserver   *apiserver.APIServer
	dispatcher  *dispatcher.Dispatcher
	ctx         *context.RootContext
	messagePool *MessageStoreHelper
	messagesCh  chan *types.Message
	eventCh     chan *types.Event
	eventDag    *types.EventDAG

	latestEvents map[types.ReplicaID]*types.Event
	sends        map[string]*types.Event
	lock         *sync.Mutex
	wg           *sync.WaitGroup

	*types.BaseService
}

// NewVisualizer instantiates a Visualizer
func NewVisualizer(ctx *context.RootContext) *Visualizer {
	v := &Visualizer{
		dispatcher:  dispatcher.NewDispatcher(ctx),
		ctx:         ctx,
		messagePool: NewMessageStoreHelper(ctx.MessageStore),
		messagesCh:  ctx.MessageQueue.Subscribe("Visualizer"),
		eventCh:     ctx.EventQueue.Subscribe("Visualizer"),
		eventDag:    types.NewEventDag(ctx.Replicas),

		latestEvents: make(map[types.ReplicaID]*types.Event),
		sends:        make(map[string]*types.Event),
		lock:         new(sync.Mutex),
		wg:           new(sync.WaitGroup),
		BaseService:  types.NewBaseService("Visualizer", log.DefaultLogger),
	}
	v.apiserver = apiserver.NewAPIServer(ctx, nil, v)
	return v
}

// Start implements Service
func (v *Visualizer) Start() {
	v.apiserver.Start()
	v.StartRunning()
	go v.eventloop()
	go v.messageloop()
}

// Stop implements Service
func (v *Visualizer) Stop() {
	v.apiserver.Stop()
	v.StopRunning()
}

func (v *Visualizer) messageloop() {
	for {
		select {
		case m := <-v.messagesCh:
			v.messagePool.Add(m)
		case <-v.QuitCh():
			return
		}
	}
}

func (v *Visualizer) addEvent(e *types.Event) {
	v.lock.Lock()
	defer v.lock.Unlock()
	defer v.wg.Done()

	latest, ok := v.latestEvents[e.Replica]
	parents := make([]*types.Event, 0)
	if ok {
		parents = append(parents, latest)
	}
	v.latestEvents[e.Replica] = e

	switch e.Type.(type) {
	case *types.MessageReceiveEventType:
		eventType := e.Type.(*types.MessageReceiveEventType)
		send, ok := v.sends[eventType.MessageID]
		if ok {
			parents = append(parents, send)
		}
	case *types.MessageSendEventType:
		eventType := e.Type.(*types.MessageSendEventType)
		v.sends[eventType.MessageID] = e
	}
	v.eventDag.AddNode(e, parents)
}

func (v *Visualizer) eventloop() {
	for {
		select {
		case e := <-v.eventCh:
			v.wg.Add(1)
			go v.addEvent(e)
		case <-v.QuitCh():
			return
		}
	}
}
