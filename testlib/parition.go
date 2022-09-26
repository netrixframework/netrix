package testlib

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/netrixframework/netrix/log"
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

type Partition struct {
	parts    map[string]*part
	replicas map[types.ReplicaID]string
	lock     *sync.Mutex
}

func NewPartition(sizes []int, labels []string) (*Partition, error) {
	if len(sizes) != len(labels) {
		return nil, ErrSizeLabelsMismatch
	}

	partition := &Partition{
		parts:    make(map[string]*part),
		replicas: make(map[types.ReplicaID]string),
		lock:     new(sync.Mutex),
	}
	for i := 0; i < len(sizes); i++ {
		partition.parts[labels[i]] = newPart(labels[i], sizes[i])
	}

	return partition, nil
}

func (p *Partition) Setup(ctx *Context) error {
	p.lock.Lock()

	totSize := 0
	for _, part := range p.parts {
		totSize += part.size
	}
	if totSize != ctx.ReplicaStore.Cap() {
		return ErrNotEnoughReplicas
	}

	parts := make([]*part, 0)
	for _, part := range p.parts {
		parts = append(parts, part)
	}

	curIndex := 0
	for _, r := range ctx.ReplicaStore.Iter() {
		curPart := parts[curIndex]
		if len(curPart.replicas) < curPart.size {
			curPart.add(r.ID)
			p.replicas[r.ID] = curPart.label
		} else {
			curIndex++
			curPart = parts[curIndex]
			curPart.add(r.ID)
			p.replicas[r.ID] = curPart.label
		}
	}
	p.lock.Unlock()

	ctx.Logger.With(log.LogParams{
		"partition": p.String(),
	}).Debug("Created partition")
	ctx.Vars.Set(partitionKey, p)
	return nil
}

func (p *Partition) InPart(replica types.ReplicaID, partLabel string) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	part, ok := p.parts[partLabel]
	if !ok {
		return false
	}
	_, exists := part.replicas[replica]
	return exists
}

func (p *Partition) GetPartLabel(replica types.ReplicaID) (string, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	label, ok := p.replicas[replica]
	return label, ok
}

func (p *Partition) String() string {
	p.lock.Lock()
	defer p.lock.Unlock()

	data := make(map[string][]string)
	for label, part := range p.parts {
		data[label] = make([]string, 0)
		for replica := range part.replicas {
			data[label] = append(data[label], string(replica))
		}
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// func NewPartition(replicas *types.ReplicaStore, sizes []int, labels []string) (*ReplicaPartition, error) {
// 	partition := &ReplicaPartition{
// 		parts: make(map[string]*part),
// 		lock:  new(sync.Mutex),
// 	}
// 	sum := 0
// 	for i, s := range sizes {
// 		sum += s
// 		partition.parts[labels[i]] = newPart(labels[i], sizes[i])
// 	}
// 	if sum > replicas.Cap() {
// 		return nil, ErrNotEnoughReplicas
// 	}

// 	if len(sizes) != len(labels) {
// 		return nil, ErrSizeLabelsMismatch
// 	}
// 	curIndex := 0
// 	curPartSize := 0
// 	for _, r := range replicas.Iter() {
// 		curLabel := labels[curIndex]
// 		curSize := sizes[curIndex]
// 		if curPartSize < curSize {
// 			partition.parts[curLabel].add(r.ID)
// 			curPartSize++
// 		} else {
// 			curIndex++
// 			curLabel = labels[curIndex]
// 			partition.parts[curLabel].add(r.ID)
// 			curPartSize = 1
// 		}
// 	}
// 	return partition, nil
// }

// func (p *ReplicaPartition) InPart(replicaID types.ReplicaID, partLabel string) bool {
// 	p.lock.Lock()
// 	defer p.lock.Unlock()
// 	part, ok := p.parts[partLabel]
// 	if !ok {
// 		return false
// 	}
// 	_, exists := part.replicas[replicaID]
// 	return exists
// }
