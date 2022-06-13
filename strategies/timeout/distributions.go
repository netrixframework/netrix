package timeout

import (
	"golang.org/x/exp/rand"

	"gonum.org/v1/gonum/stat/distuv"
)

type ParetoDistribution struct {
	*distuv.Pareto
}

func NewParetoDistribution(Xm, Alpha float64) *ParetoDistribution {
	return &ParetoDistribution{
		Pareto: &distuv.Pareto{
			Xm:    Xm,
			Alpha: Alpha,
		},
	}
}

func (p *ParetoDistribution) SetSrc(src rand.Source) {
	p.Pareto.Src = src
}

func (p *ParetoDistribution) Rand() int {
	return int(p.Pareto.Rand())
}

type WeibullDistribution struct {
	*distuv.Weibull
}

func NewWeibullDistribution(K, Lambda float64) *WeibullDistribution {
	return &WeibullDistribution{
		Weibull: &distuv.Weibull{
			K:      K,
			Lambda: Lambda,
		},
	}
}

func (p *WeibullDistribution) SetSrc(src rand.Source) {
	p.Weibull.Src = src
}

func (p *WeibullDistribution) Rand() int {
	return int(p.Weibull.Rand())
}

type ExpDistribution struct {
	*distuv.Exponential
}

func NewExpDistribution(Rate float64) *ExpDistribution {
	return &ExpDistribution{
		Exponential: &distuv.Exponential{
			Rate: Rate,
		},
	}
}

func (p *ExpDistribution) SetSrc(src rand.Source) {
	p.Exponential.Src = src
}

func (p *ExpDistribution) Rand() int {
	return int(p.Exponential.Rand())
}
