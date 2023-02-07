package fuzzing

type DefaultMutator struct {
}

var _ Mutator = &DefaultMutator{}

func NewDefaultMutator() *DefaultMutator {
	return &DefaultMutator{}
}

func (d *DefaultMutator) Mutate(i *Input) *Input {
	return i
}
