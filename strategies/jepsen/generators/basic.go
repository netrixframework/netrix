package generators

import (
	"time"

	"github.com/netrixframework/netrix/strategies"
	"github.com/netrixframework/netrix/strategies/jepsen"
	"github.com/netrixframework/netrix/types"
)

type TimeLimitGen struct {
	Limit time.Duration
	jepsen.Generator

	doneCh *types.Channel[struct{}]
}

func TimeLimit(d time.Duration, gen jepsen.Generator) *TimeLimitGen {
	return &TimeLimitGen{
		Limit:     d,
		Generator: gen,
		doneCh:    types.NewChannel[struct{}](),
	}
}

var _ jepsen.Generator = &TimeLimitGen{}

func (t *TimeLimitGen) Start(c *strategies.Context) error {
	err := t.Generator.Start(c)
	if err != nil {
		return err
	}
	go func() {
		select {
		case <-t.doneCh.Ch():
		case <-time.After(t.Limit):
		case <-t.Generator.Done():
		}
		t.doneCh.Close()
	}()
	return nil
}

func (t *TimeLimitGen) Reset(c *strategies.Context) error {
	t.doneCh.Close()
	err := t.Generator.Reset(c)
	if err != nil {
		return err
	}
	t.doneCh.Reset()
	return t.Start(c)
}

func (t *TimeLimitGen) Stop(c *strategies.Context) error {
	t.doneCh.Close()
	return t.Generator.Stop(c)
}

func (t *TimeLimitGen) Done() chan struct{} {
	return t.doneCh.Ch()
}

type StaggeredGen struct {
	jepsen.Generator
	StaggeredDuration time.Duration
	ticker            *time.Ticker

	outActions *types.Channel[*strategies.Action]
}

var _ jepsen.Generator = &StaggeredGen{}

func Staggered(d time.Duration, gen jepsen.Generator) *StaggeredGen {
	return &StaggeredGen{
		Generator:         gen,
		StaggeredDuration: d,
		ticker:            time.NewTicker(d),
		outActions:        types.NewChannel[*strategies.Action](),
	}
}

func (s *StaggeredGen) Start(c *strategies.Context) error {
	err := s.Generator.Start(c)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-s.Generator.Done():
				return
			case <-s.ticker.C:
				action := <-s.Generator.Actions().Ch()
				s.outActions.BlockingAdd(action)
			}
		}
	}()
	return nil
}

func (s *StaggeredGen) Actions() *types.Channel[*strategies.Action] {
	return s.outActions
}

func (s *StaggeredGen) Reset(c *strategies.Context) error {
	s.outActions.Reset()
	return s.Generator.Reset(c)
}

type CycleGen struct {
	Gens       []jepsen.Generator
	outActions *types.Channel[*strategies.Action]

	doneCh *types.Channel[struct{}]
}

var _ jepsen.Generator = &CycleGen{}

func Cycle(gens ...jepsen.Generator) *CycleGen {
	return &CycleGen{
		Gens:       gens,
		outActions: types.NewChannel[*strategies.Action](),
		doneCh:     types.NewChannel[struct{}](),
	}
}

func (c *CycleGen) Start(ctx *strategies.Context) error {
	for _, g := range c.Gens {
		if err := g.Start(ctx); err != nil {
			return err
		}
	}
	go func() {
	ActionsLoop:
		for {
			for _, g := range c.Gens {
				select {
				case <-c.doneCh.Ch():
					break ActionsLoop
				case action := <-g.Actions().Ch():
					c.outActions.BlockingAdd(action)
				}
			}
		}
		c.doneCh.Close()
	}()
	return nil
}

func (c *CycleGen) Done() chan struct{} {
	return c.doneCh.Ch()
}

func (c *CycleGen) Stop(ctx *strategies.Context) error {
	var err error
	for _, g := range c.Gens {
		if e := g.Stop(ctx); e != nil {
			err = e
		}
	}
	c.doneCh.Close()
	return err
}

func (c *CycleGen) Actions() *types.Channel[*strategies.Action] {
	return c.outActions
}

func (c *CycleGen) Reset(ctx *strategies.Context) error {
	c.doneCh.Close()
	var err error
	for _, g := range c.Gens {
		if e := g.Stop(ctx); e != nil {
			err = e
		}
	}
	c.doneCh.Reset()
	return err
}
