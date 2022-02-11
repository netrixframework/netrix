package visualizer

import (
	"sync"

	"github.com/netrixframework/netrix/types"
)

type MessageStoreHelper struct {
	*types.MessageStore

	fromPending   map[types.ReplicaID]map[string]*types.Message
	toPending     map[types.ReplicaID]map[string]*types.Message
	fromDelivered map[types.ReplicaID]map[string]*types.Message
	toDelivered   map[types.ReplicaID]map[string]*types.Message
	lock          *sync.Mutex
}

func NewMessageStoreHelper(store *types.MessageStore) *MessageStoreHelper {
	return &MessageStoreHelper{
		MessageStore:  store,
		fromPending:   make(map[types.ReplicaID]map[string]*types.Message),
		toPending:     make(map[types.ReplicaID]map[string]*types.Message),
		fromDelivered: make(map[types.ReplicaID]map[string]*types.Message),
		toDelivered:   make(map[types.ReplicaID]map[string]*types.Message),

		lock: new(sync.Mutex),
	}
}

func (m *MessageStoreHelper) Add(message *types.Message) {
	m.MessageStore.Add(message)
	m.lock.Lock()
	defer m.lock.Unlock()
	_, ok := m.fromPending[message.From]
	if !ok {
		m.fromPending[message.From] = make(map[string]*types.Message)
	}
	m.fromPending[message.From][message.ID] = message

	_, ok = m.toPending[message.To]
	if !ok {
		m.toPending[message.To] = make(map[string]*types.Message)
	}
	m.toPending[message.To][message.ID] = message
}

func (m *MessageStoreHelper) MarkDelivered(messageID string) bool {
	message, ok := m.MessageStore.Get(messageID)
	if !ok {
		return false
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.fromPending[message.From], message.ID)
	delete(m.toPending[message.To], message.ID)

	_, ok = m.fromDelivered[message.From]
	if !ok {
		m.fromDelivered[message.From] = make(map[string]*types.Message)
	}
	m.fromDelivered[message.From][message.ID] = message
	_, ok = m.toDelivered[message.To]
	if !ok {
		m.toDelivered[message.To] = make(map[string]*types.Message)
	}
	m.toDelivered[message.To][message.ID] = message
	return true
}

func (m *MessageStoreHelper) GetPendingFrom(replica types.ReplicaID) (map[string]*types.Message, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	pending, ok := m.fromPending[replica]
	return pending, ok
}

func (m *MessageStoreHelper) GetPendingTo(replica types.ReplicaID) (map[string]*types.Message, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	pending, ok := m.toPending[replica]
	return pending, ok
}

func (m *MessageStoreHelper) GetDeliveredFrom(replica types.ReplicaID) (map[string]*types.Message, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delivered, ok := m.fromDelivered[replica]
	return delivered, ok
}

func (m *MessageStoreHelper) GetDeliveredTo(replica types.ReplicaID) (map[string]*types.Message, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delivered, ok := m.toDelivered[replica]
	return delivered, ok
}
