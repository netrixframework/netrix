package z3

// #include "go-z3.h"
import "C"
import "sync"

// Solver is a single solver tied to a specific Context within Z3.
//
// It is created via the NewSolver methods on Context. When a solver is
// no longer needed, the Close method must be called. This will remove the
// solver from the context and no more APIs on Solver may be called
// thereafter.
//
// Freeing the context (Context.Close) will NOT automatically close associated
// solvers. They must be managed separately.
type Solver struct {
	ctx       *Context
	rawSolver C.Z3_solver
	lock      *sync.Mutex
}

// NewSolver creates a new solver.
func (c *Context) NewSolver() *Solver {
	c.Mutex.Lock()
	rawSolver := C.Z3_mk_solver(c.Raw)
	C.Z3_solver_inc_ref(c.Raw, rawSolver)
	c.Mutex.Unlock()

	return &Solver{
		rawSolver: rawSolver,
		ctx:       c,
		lock:      new(sync.Mutex),
	}
}

// Close frees the memory associated with this.
func (s *Solver) Close() error {
	s.ctx.Lock()
	s.lock.Lock()
	C.Z3_solver_dec_ref(s.ctx.Raw, s.rawSolver)
	s.lock.Unlock()
	s.ctx.Unlock()
	return nil
}

// Assert asserts a constraint onto the Solver.
//
// Maps to: Z3_solver_assert
func (s *Solver) Assert(a *AST) {
	s.ctx.Lock()
	s.lock.Lock()
	C.Z3_solver_assert(s.ctx.Raw, s.rawSolver, a.rawAST)
	s.lock.Unlock()
	s.ctx.Unlock()
}

// Check checks if the currently set formula is consistent.
//
// Maps to: Z3_solver_check
func (s *Solver) Check() LBool {
	s.ctx.Lock()
	s.lock.Lock()
	check := C.Z3_solver_check(s.ctx.Raw, s.rawSolver)
	s.lock.Unlock()
	s.ctx.Unlock()
	return LBool(check)
}

// Model returns the last model from a Check.
//
// Maps to: Z3_solver_get_model
func (s *Solver) Model() *Model {
	s.ctx.Lock()
	s.lock.Lock()
	m := &Model{
		ctx:      s.ctx,
		rawModel: C.Z3_solver_get_model(s.ctx.Raw, s.rawSolver),
	}
	s.lock.Unlock()
	s.ctx.Unlock()
	m.IncRef()
	return m
}

func (s *Solver) Push() {
	s.ctx.Lock()
	s.lock.Lock()
	C.Z3_solver_push(s.ctx.Raw, s.rawSolver)
	s.lock.Unlock()
	s.ctx.Unlock()
}

func (s *Solver) Pop() {
	s.ctx.Lock()
	s.lock.Lock()
	C.Z3_solver_pop(s.ctx.Raw, s.rawSolver, C.uint(1))
	s.lock.Unlock()
	s.ctx.Unlock()
}

func (s *Solver) PopN(n uint) {
	s.ctx.Lock()
	s.lock.Lock()
	C.Z3_solver_pop(s.ctx.Raw, s.rawSolver, C.uint(n))
	s.lock.Unlock()
	s.ctx.Unlock()
}

func (s *Solver) Reset() {
	s.ctx.Lock()
	s.lock.Lock()
	C.Z3_solver_dec_ref(s.ctx.Raw, s.rawSolver)
	rawSolver := C.Z3_mk_solver(s.ctx.Raw)
	C.Z3_solver_inc_ref(s.ctx.Raw, rawSolver)
	s.rawSolver = rawSolver
	s.lock.Unlock()
	s.ctx.Unlock()
}
