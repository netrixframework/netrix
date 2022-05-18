package z3

import (
	"unsafe"
)

// #include "go-z3.h"
import "C"

type Optimizer struct {
	rawCtx C.Z3_context
	rawOpt C.Z3_optimize
}

func (c *Context) NewOptimizer() *Optimizer {
	rawOpt := C.Z3_mk_optimize(c.raw)
	C.Z3_optimize_inc_ref(c.raw, rawOpt)

	return &Optimizer{
		rawCtx: c.raw,
		rawOpt: rawOpt,
	}
}

func (o *Optimizer) Close() error {
	C.Z3_optimize_dec_ref(o.rawCtx, o.rawOpt)
	return nil
}

func (o *Optimizer) Assert(a *AST) {
	C.Z3_optimize_assert(o.rawCtx, o.rawOpt, a.rawAST)
}

type OptimizedValue struct {
	opt   *Optimizer
	index uint
	max   bool
}

func (o *OptimizedValue) lower() (float64, bool) {
	return (&AST{
		rawCtx: o.opt.rawCtx,
		rawAST: C.Z3_optimize_get_lower(o.opt.rawCtx, o.opt.rawOpt, C.uint(o.index)),
	}).Float()
}

func (o *OptimizedValue) upper() (float64, bool) {
	return (&AST{
		rawCtx: o.opt.rawCtx,
		rawAST: C.Z3_optimize_get_upper(o.opt.rawCtx, o.opt.rawOpt, C.uint(o.index)),
	}).Float()
}

func (o *OptimizedValue) Value() (float64, bool) {
	if o.max {
		return o.upper()
	}
	return o.lower()
}

func (o *Optimizer) Maximize(a *AST) *OptimizedValue {
	return &OptimizedValue{
		opt:   o,
		index: uint(C.Z3_optimize_maximize(o.rawCtx, o.rawOpt, a.rawAST)),
		max:   true,
	}
}

func (o *Optimizer) Minimize(a *AST) *OptimizedValue {
	return &OptimizedValue{
		opt:   o,
		index: uint(C.Z3_optimize_minimize(o.rawCtx, o.rawOpt, a.rawAST)),
		max:   false,
	}
}

func (o *Optimizer) Check(assumptions ...*AST) LBool {
	raws := make([]C.Z3_ast, len(assumptions)+1)
	for i, a := range assumptions {
		raws[i] = a.rawAST
	}
	return LBool(C.Z3_optimize_check(
		o.rawCtx,
		o.rawOpt,
		C.uint(len(assumptions)),
		(*C.Z3_ast)(unsafe.Pointer(&raws[0])),
	))
}

func (o *Optimizer) Push() {
	C.Z3_optimize_push(o.rawCtx, o.rawOpt)
}

func (o *Optimizer) Pop() {
	C.Z3_optimize_pop(o.rawCtx, o.rawOpt)
}

func (o *Optimizer) Model() *Model {
	m := &Model{
		rawCtx:   o.rawCtx,
		rawModel: C.Z3_optimize_get_model(o.rawCtx, o.rawOpt),
	}
	m.IncRef()
	return m
}
