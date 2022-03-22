package types

import "sync"

type Annotation struct{}

type AnnotatedDAG struct {
	*EventDAG
	annotations    map[EventID]*Annotation
	annotationLock *sync.Mutex
}

func NewAnnotatedDAG(dag *EventDAG) *AnnotatedDAG {
	return &AnnotatedDAG{
		EventDAG:       dag,
		annotations:    make(map[EventID]*Annotation),
		annotationLock: new(sync.Mutex),
	}
}

func (d *AnnotatedDAG) AnnotateNode(e *Event, a *Annotation) {
	d.annotationLock.Lock()
	defer d.annotationLock.Unlock()

	d.annotations[e.ID] = a
}

func (d *AnnotatedDAG) GetAnnotation(e *Event) (*Annotation, bool) {
	d.annotationLock.Lock()
	defer d.annotationLock.Unlock()

	a, ok := d.annotations[e.ID]
	return a, ok
}

func (d *AnnotatedDAG) RemoveAnnotations() {
	d.annotationLock.Lock()
	defer d.annotationLock.Unlock()
	d.annotations = make(map[EventID]*Annotation)
}

type EventDAGExtension struct {
	*EventDAG
	changedNodes map[EventID]*EventNode
	strands      map[ReplicaID]EventID
}
