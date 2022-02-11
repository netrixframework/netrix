package types

// Clonable is any type which returns a copy of itself on Clone()
type Clonable interface {
	Clone() Clonable
}

// Generic subscriber to maintain state of the subsciber
type Subscriber struct {
	Ch chan interface{}
}
