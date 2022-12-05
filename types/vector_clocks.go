package types

// ClockValue represents a vector clock value
type ClockValue map[ReplicaID]int

// ZeroClock creates a new empty ClockValue
func ZeroClock() ClockValue {
	return make(map[ReplicaID]int)
}

// Lt returns true if c < other
func (c ClockValue) Lt(other ClockValue) bool {
	oneless := false
	for replica, v1 := range c {
		v2, ok := other[replica]
		if ok && v1 < v2 {
			oneless = true
		} else if ok && v1 > v2 {
			return false
		}

	}
	return oneless
}

// Next increments the clock value for the specified replica when it has been initialized
// and sets it to 0 otherwise
func (c ClockValue) Next(replica ReplicaID) ClockValue {
	new := ZeroClock()
	for r, val := range c {
		new[r] = val
	}

	cur, ok := c[replica]
	if !ok {
		new[replica] = 0
	} else {
		new[replica] = cur + 1
	}
	return new
}
