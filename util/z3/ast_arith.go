package z3

import (
	"unsafe"
)

// #include "go-z3.h"
import "C"

// Add creates an AST node representing adding.
//
// All AST values must be part of the same context.
func (a *AST) Add(args ...*AST) *AST {
	raws := make([]C.Z3_ast, len(args)+1)
	raws[0] = a.rawAST
	for i, arg := range args {
		raws[i+1] = arg.rawAST
	}

	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &AST{
		ctx: a.ctx,
		rawAST: C.Z3_mk_add(
			a.ctx.Raw,
			C.uint(len(raws)),
			(*C.Z3_ast)(unsafe.Pointer(&raws[0]))),
	}
}

// Mul creates an AST node representing multiplication.
//
// All AST values must be part of the same context.
func (a *AST) Mul(args ...*AST) *AST {
	raws := make([]C.Z3_ast, len(args)+1)
	raws[0] = a.rawAST
	for i, arg := range args {
		raws[i+1] = arg.rawAST
	}

	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &AST{
		ctx: a.ctx,
		rawAST: C.Z3_mk_mul(
			a.ctx.Raw,
			C.uint(len(raws)),
			(*C.Z3_ast)(unsafe.Pointer(&raws[0]))),
	}
}

func (a *AST) Div(other *AST) *AST {
	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &AST{
		ctx: a.ctx,
		rawAST: C.Z3_mk_div(
			a.ctx.Raw,
			a.rawAST,
			other.rawAST,
		),
	}
}

// Sub creates an AST node representing subtraction.
//
// All AST values must be part of the same context.
func (a *AST) Sub(other *AST) *AST {
	raws := make([]C.Z3_ast, 2)
	raws[0] = a.rawAST
	raws[1] = other.rawAST

	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &AST{
		ctx: a.ctx,
		rawAST: C.Z3_mk_sub(
			a.ctx.Raw,
			C.uint(len(raws)),
			(*C.Z3_ast)(unsafe.Pointer(&raws[0]))),
	}
}

// Lt creates a "less than" comparison.
//
// Maps to: Z3_mk_lt
func (a *AST) Lt(a2 *AST) *AST {
	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &AST{
		ctx:    a.ctx,
		rawAST: C.Z3_mk_lt(a.ctx.Raw, a.rawAST, a2.rawAST),
	}
}

// Le creates a "less than" comparison.
//
// Maps to: Z3_mk_le
func (a *AST) Le(a2 *AST) *AST {
	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &AST{
		ctx:    a.ctx,
		rawAST: C.Z3_mk_le(a.ctx.Raw, a.rawAST, a2.rawAST),
	}
}

// Gt creates a "greater than" comparison.
//
// Maps to: Z3_mk_gt
func (a *AST) Gt(a2 *AST) *AST {
	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &AST{
		ctx:    a.ctx,
		rawAST: C.Z3_mk_gt(a.ctx.Raw, a.rawAST, a2.rawAST),
	}
}

// Ge creates a "less than" comparison.
//
// Maps to: Z3_mk_ge
func (a *AST) Ge(a2 *AST) *AST {
	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &AST{
		ctx:    a.ctx,
		rawAST: C.Z3_mk_ge(a.ctx.Raw, a.rawAST, a2.rawAST),
	}
}
