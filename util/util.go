package util

import (
	crand "crypto/rand"
	"errors"
	"math"
	"math/big"
	"math/rand"
	"sync"
)

// Counter is a thread safe monotonic natural number counter
type Counter struct {
	counter int
	mtx     *sync.Mutex
}

// NewCounter instantiates Counter
func NewCounter() *Counter {
	return &Counter{
		counter: 0,
		mtx:     new(sync.Mutex),
	}
}

// Next returns the next value
func (id *Counter) Next() int {
	id.mtx.Lock()
	defer id.mtx.Unlock()

	cur := id.counter
	id.counter = id.counter + 1

	return cur
}

// Reset resets the counter to 0
func (id *Counter) Reset() {
	id.mtx.Lock()
	defer id.mtx.Unlock()

	id.counter = 0
}

func init() {
	r, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		panic(errors.New("could not read initialize random bytes"))
	}
	rand.Seed(r.Int64())
}
