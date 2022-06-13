package z3

// #include <stdlib.h>
// #include "go-z3.h"
import "C"
import "unsafe"

type ASTKind struct {
	rawASTKind C.Z3_ast_kind
}

func (a *ASTKind) Eq(other *ASTKind) bool {
	return a.rawASTKind == other.rawASTKind
}

var (
	AppAST *ASTKind = &ASTKind{
		rawASTKind: C.Z3_APP_AST,
	}
	NumeralAST *ASTKind = &ASTKind{
		rawASTKind: C.Z3_NUMERAL_AST,
	}
	VarAST *ASTKind = &ASTKind{
		rawASTKind: C.Z3_VAR_AST,
	}
	QuantifierAST *ASTKind = &ASTKind{
		rawASTKind: C.Z3_QUANTIFIER_AST,
	}
	SortAST *ASTKind = &ASTKind{
		rawASTKind: C.Z3_SORT_AST,
	}
	FuncDeclAST *ASTKind = &ASTKind{
		rawASTKind: C.Z3_FUNC_DECL_AST,
	}
	UnknownAST *ASTKind = &ASTKind{
		rawASTKind: C.Z3_UNKNOWN_AST,
	}
)

// AST represents an AST value in Z3.
//
// AST memory management is automatically managed by the Context it
// is contained within. When the Context is freed, so are the AST nodes.
type AST struct {
	ctx    *Context
	rawAST C.Z3_ast
}

func (a *AST) Kind() *ASTKind {
	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &ASTKind{
		rawASTKind: C.Z3_get_ast_kind(a.ctx.Raw, a.rawAST),
	}
}

func (a *AST) Sort() *Sort {
	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &Sort{
		ctx:     a.ctx,
		rawSort: C.Z3_get_sort(a.ctx.Raw, a.rawAST),
	}
}

// String returns a human-friendly string version of the AST.
func (a *AST) String() string {
	a.ctx.Lock()
	defer a.ctx.Unlock()
	return C.GoString(C.Z3_ast_to_string(a.ctx.Raw, a.rawAST))
}

// DeclName returns the name of a declaration. The AST value must be a
// func declaration for this to work.
func (a *AST) DeclName() *Symbol {
	a.ctx.Lock()
	defer a.ctx.Unlock()
	return &Symbol{
		ctx: a.ctx,
		rawSymbol: C.Z3_get_decl_name(
			a.ctx.Raw, C.Z3_to_func_decl(a.ctx.Raw, a.rawAST)),
	}
}

//-------------------------------------------------------------------
// Var, Literal Creation
//-------------------------------------------------------------------

// Const declares a variable. It is called "Const" since internally
// this is equivalent to create a function that always returns a constant
// value. From an initial user perspective this may be confusing but go-z3
// is following identical naming convention.
func (c *Context) Const(s *Symbol, typ *Sort) *AST {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &AST{
		ctx:    c,
		rawAST: C.Z3_mk_const(c.Raw, s.rawSymbol, typ.rawSort),
	}
}

func (c *Context) IntConst(label string) *AST {
	name := C.CString(label)
	defer C.free(unsafe.Pointer(name))

	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &AST{
		ctx:    c,
		rawAST: C.Z3_mk_const(c.Raw, C.Z3_mk_string_symbol(c.Raw, name), C.Z3_mk_int_sort(c.Raw)),
	}
}

func (c *Context) RealConst(label string) *AST {
	name := C.CString(label)
	defer C.free(unsafe.Pointer(name))
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &AST{
		ctx:    c,
		rawAST: C.Z3_mk_const(c.Raw, C.Z3_mk_string_symbol(c.Raw, name), C.Z3_mk_real_sort(c.Raw)),
	}
}

// Int creates an integer type.
//
// Maps: Z3_mk_int
func (c *Context) Int(v int) *AST {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &AST{
		ctx:    c,
		rawAST: C.Z3_mk_int(c.Raw, C.int(v), C.Z3_mk_int_sort(c.Raw)),
	}
}

// Read creates a real type
//
// Maps: Z3_mk_real
func (c *Context) Real(num, den int) *AST {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &AST{
		ctx:    c,
		rawAST: C.Z3_mk_real(c.Raw, C.int(num), C.int(den)),
	}
}

// True creates the value "true".
//
// Maps: Z3_mk_true
func (c *Context) True() *AST {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &AST{
		ctx:    c,
		rawAST: C.Z3_mk_true(c.Raw),
	}
}

// False creates the value "false".
//
// Maps: Z3_mk_false
func (c *Context) False() *AST {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &AST{
		ctx:    c,
		rawAST: C.Z3_mk_false(c.Raw),
	}
}

//-------------------------------------------------------------------
// Value Readers
//-------------------------------------------------------------------

// Int gets the integer value of this AST. The value must be able to fit
// into a machine integer.
func (a *AST) Int() (int, bool) {
	if a == nil {
		return 0, false
	}
	sortKind := a.Sort().Kind()
	kind := a.Kind()
	if kind.Eq(NumeralAST) && sortKind.Eq(IntSort) {
		var dst C.int
		a.ctx.Lock()
		C.Z3_get_numeral_int(a.ctx.Raw, a.rawAST, &dst)
		a.ctx.Unlock()
		return int(dst), true
	}
	return 0, false
}

func (a *AST) numerator() (int, bool) {
	if a == nil {
		return 0, false
	}
	kind := a.Kind()
	sortKind := a.Sort().Kind()
	if !kind.Eq(NumeralAST) || !sortKind.Eq(RealSort) {
		return 0, false
	}
	a.ctx.Lock()
	numeratorAST := &AST{
		a.ctx,
		C.Z3_get_numerator(a.ctx.Raw, a.rawAST),
	}
	a.ctx.Unlock()
	return numeratorAST.Int()
}

func (a *AST) denominator() (int, bool) {
	if a == nil {
		return 0, false
	}
	kind := a.Kind()
	sortKind := a.Sort().Kind()
	if !kind.Eq(NumeralAST) || !sortKind.Eq(RealSort) {
		return 0, false
	}
	a.ctx.Lock()
	numeratorAST := &AST{
		a.ctx,
		C.Z3_get_denominator(a.ctx.Raw, a.rawAST),
	}
	a.ctx.Unlock()
	return numeratorAST.Int()
}

func (a *AST) Float() (float64, bool) {
	if a == nil {
		return 0, false
	}
	sortKind := a.Sort().Kind()
	kind := a.Kind()
	if kind.Eq(NumeralAST) {
		if sortKind.Eq(IntSort) {
			var dst C.int
			a.ctx.Lock()
			C.Z3_get_numeral_int(a.ctx.Raw, a.rawAST, &dst)
			a.ctx.Unlock()
			return float64(dst), true
		} else if sortKind.Eq(RealSort) {
			numerator, _ := a.numerator()
			denominator, _ := a.denominator()
			return float64(numerator) / float64(denominator), true
		}
	}
	return 0, false
}
