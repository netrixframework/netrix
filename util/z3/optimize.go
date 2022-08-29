package z3

import (
	"sync"
	"unsafe"
)

// #include "go-z3.h"
import "C"

type Optimizer struct {
	ctx    *Context
	rawOpt C.Z3_optimize
	lock   *sync.Mutex
}

func (c *Context) NewOptimizer() *Optimizer {
	c.Mutex.Lock()
	rawOpt := C.Z3_mk_optimize(c.Raw)
	C.Z3_optimize_inc_ref(c.Raw, rawOpt)
	c.Mutex.Unlock()

	return &Optimizer{
		ctx:    c,
		rawOpt: rawOpt,
		lock:   new(sync.Mutex),
	}
}

func (o *Optimizer) Close() error {
	o.ctx.Lock()
	o.lock.Lock()
	C.Z3_optimize_dec_ref(o.ctx.Raw, o.rawOpt)
	o.lock.Unlock()
	o.ctx.Unlock()
	return nil
}

func (o *Optimizer) Assert(a *AST) {
	o.ctx.Lock()
	o.lock.Lock()
	C.Z3_optimize_assert(o.ctx.Raw, o.rawOpt, a.rawAST)
	o.lock.Unlock()
	o.ctx.Unlock()
}

type OptimizedValue struct {
	opt   *Optimizer
	index uint
	max   bool
}

func (o *OptimizedValue) lower() (float64, bool) {
	ctx := o.opt.ctx
	ctx.Lock()
	o.opt.lock.Lock()
	ast := &AST{
		ctx:    ctx,
		rawAST: C.Z3_optimize_get_lower(ctx.Raw, o.opt.rawOpt, C.uint(o.index)),
	}
	o.opt.lock.Unlock()
	ctx.Unlock()
	return ast.Float()
}

func (o *OptimizedValue) upper() (float64, bool) {
	ctx := o.opt.ctx
	ctx.Lock()
	o.opt.lock.Lock()
	ast := &AST{
		ctx:    ctx,
		rawAST: C.Z3_optimize_get_upper(ctx.Raw, o.opt.rawOpt, C.uint(o.index)),
	}
	o.opt.lock.Unlock()
	ctx.Unlock()
	return ast.Float()
}

func (o *OptimizedValue) Value() (float64, bool) {
	if o.max {
		return o.upper()
	}
	return o.lower()
}

func (o *Optimizer) Maximize(a *AST) *OptimizedValue {
	o.ctx.Lock()
	o.lock.Lock()
	v := &OptimizedValue{
		opt:   o,
		index: uint(C.Z3_optimize_maximize(o.ctx.Raw, o.rawOpt, a.rawAST)),
		max:   true,
	}
	o.lock.Unlock()
	o.ctx.Unlock()
	return v
}

func (o *Optimizer) Minimize(a *AST) *OptimizedValue {
	o.ctx.Lock()
	o.lock.Lock()
	v := &OptimizedValue{
		opt:   o,
		index: uint(C.Z3_optimize_minimize(o.ctx.Raw, o.rawOpt, a.rawAST)),
		max:   false,
	}
	o.lock.Unlock()
	o.ctx.Unlock()
	return v
}

func (o *Optimizer) Check(assumptions ...*AST) LBool {
	raws := make([]C.Z3_ast, len(assumptions)+1)
	for i, a := range assumptions {
		raws[i] = a.rawAST
	}
	o.ctx.Lock()
	o.lock.Lock()
	result := LBool(C.Z3_optimize_check(
		o.ctx.Raw,
		o.rawOpt,
		C.uint(len(assumptions)),
		(*C.Z3_ast)(unsafe.Pointer(&raws[0])),
	))
	o.lock.Unlock()
	o.ctx.Unlock()
	return result
}

func (o *Optimizer) Push() {
	o.ctx.Lock()
	o.lock.Lock()
	C.Z3_optimize_push(o.ctx.Raw, o.rawOpt)
	o.lock.Unlock()
	o.ctx.Unlock()
}

func (o *Optimizer) Pop() {
	o.ctx.Lock()
	o.lock.Lock()
	C.Z3_optimize_pop(o.ctx.Raw, o.rawOpt)
	o.lock.Unlock()
	o.ctx.Unlock()
}

func (o *Optimizer) Model() *Model {
	o.ctx.Lock()
	o.lock.Lock()
	m := &Model{
		ctx:      o.ctx,
		rawModel: C.Z3_optimize_get_model(o.ctx.Raw, o.rawOpt),
	}
	o.lock.Unlock()
	o.ctx.Unlock()
	m.IncRef()
	return m
}

func (o *Optimizer) Reset() {
	o.ctx.Lock()
	o.lock.Lock()
	C.Z3_optimize_dec_ref(o.ctx.Raw, o.rawOpt)
	rawOpt := C.Z3_mk_optimize(o.ctx.Raw)
	C.Z3_optimize_inc_ref(o.ctx.Raw, rawOpt)
	o.rawOpt = rawOpt
	o.lock.Unlock()
	o.ctx.Unlock()
}
