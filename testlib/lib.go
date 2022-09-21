package testlib

import (
	"errors"
	"sync"

	"github.com/netrixframework/netrix/types"
)

var (
	ErrNotEnoughReplicas  = errors.New("not enough replicas")
	ErrSizeLabelsMismatch = errors.New("sizes and labels are not of the same length")
)

type part struct {
	replicas map[types.ReplicaID]bool
	label    string
	size     int
}

func newPart(label string, size int) *part {
	return &part{
		replicas: make(map[types.ReplicaID]bool),
		label:    label,
		size:     size,
	}
}

func (p *part) add(r types.ReplicaID) {
	p.replicas[r] = true
}

type ReplicaPartition struct {
	parts map[string]*part
	lock  *sync.Mutex
}

func NewPartition(replicas *types.ReplicaStore, sizes []int, labels []string) (*ReplicaPartition, error) {
	partition := &ReplicaPartition{
		parts: make(map[string]*part),
		lock:  new(sync.Mutex),
	}
	sum := 0
	for i, s := range sizes {
		sum += s
		partition.parts[labels[i]] = newPart(labels[i], sizes[i])
	}
	if sum > replicas.Cap() {
		return nil, ErrNotEnoughReplicas
	}

	if len(sizes) != len(labels) {
		return nil, ErrSizeLabelsMismatch
	}
	curIndex := 0
	curPartSize := 0
	for _, r := range replicas.Iter() {
		curLabel := labels[curIndex]
		curSize := sizes[curIndex]
		if curPartSize < curSize {
			partition.parts[curLabel].add(r.ID)
			curPartSize++
		} else {
			curIndex++
			curLabel = labels[curIndex]
			partition.parts[curLabel].add(r.ID)
		}
	}
	return partition, nil
}

func (p *ReplicaPartition) InPart(replicaID types.ReplicaID, partLabel string) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	part, ok := p.parts[partLabel]
	if !ok {
		return false
	}
	_, exists := part.replicas[replicaID]
	return exists
}
