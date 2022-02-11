package types

type ClockValue []float64

func (c ClockValue) Lt(other ClockValue) bool {
	if len(c) != len(other) {
		return false
	}
	oneless := false
	for i := 0; i < len(c); i++ {
		if c[i] > other[i] {
			return false
		}
		if c[i] < other[i] {
			oneless = true
		}
	}
	return oneless
}

func (c ClockValue) Eq(other ClockValue) bool {
	if len(c) != len(other) {
		return false
	}
	for i := 0; i < len(c); i++ {
		if c[i] != other[i] {
			return false
		}
	}
	return true
}

type GlobalClock struct {
	dag          *EventDAG
	messageStore *MessageStore
}

func NewGlobalClock(dag *EventDAG, messageStore *MessageStore) *GlobalClock {
	return &GlobalClock{
		dag:          dag,
		messageStore: messageStore,
	}
}
