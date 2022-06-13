package z3

import (
	"strconv"
	"unsafe"
)

// #include <stdlib.h>
// #include "go-z3.h"
import "C"

// Symbol represents a named
type Symbol struct {
	ctx       *Context
	rawSymbol C.Z3_symbol
}

// Create a symbol named by a string within the context.
//
// The memory associated with this symbol is freed when the context is freed.
func (c *Context) Symbol(name string) *Symbol {
	ns := C.CString(name)
	defer C.free(unsafe.Pointer(ns))

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	return &Symbol{
		ctx:       c,
		rawSymbol: C.Z3_mk_string_symbol(c.Raw, ns),
	}
}

// Create a symbol named by an int within the context.
//
// The memory associated with this symbol is freed when the context is freed.
func (c *Context) SymbolInt(name int) *Symbol {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &Symbol{
		ctx:       c,
		rawSymbol: C.Z3_mk_int_symbol(c.Raw, C.int(name)),
	}
}

// String returns a string value for this symbol no matter what kind
// of symbol it is. If it is an int, it will be converted to a string
// result.
func (s *Symbol) String() string {
	s.ctx.Lock()
	defer s.ctx.Unlock()
	switch C.Z3_get_symbol_kind(s.ctx.Raw, s.rawSymbol) {
	case C.Z3_INT_SYMBOL:
		return strconv.FormatInt(
			int64(C.Z3_get_symbol_int(s.ctx.Raw, s.rawSymbol)), 10)

	case C.Z3_STRING_SYMBOL:
		// We don't need to free this value since it uses statically allocated
		// space that is reused by Z3. The GoString call will copy the memory.
		return C.GoString(C.Z3_get_symbol_string(s.ctx.Raw, s.rawSymbol))

	default:
		return "unknown symbol kind"
	}
}
