package z3

// #include "go-z3.h"
import "C"

type SortKind struct {
	rawSortKind C.Z3_sort_kind
}

func (s *SortKind) Eq(other *SortKind) bool {
	return s.rawSortKind == other.rawSortKind
}

var (
	UninterpretedSort *SortKind = &SortKind{rawSortKind: C.Z3_UNINTERPRETED_SORT}
	BoolSort          *SortKind = &SortKind{rawSortKind: C.Z3_BOOL_SORT}
	IntSort           *SortKind = &SortKind{rawSortKind: C.Z3_INT_SORT}
	RealSort          *SortKind = &SortKind{rawSortKind: C.Z3_REAL_SORT}
	BvSort            *SortKind = &SortKind{rawSortKind: C.Z3_BV_SORT}
	ArraySort         *SortKind = &SortKind{rawSortKind: C.Z3_ARRAY_SORT}
	DatatypeSort      *SortKind = &SortKind{rawSortKind: C.Z3_DATATYPE_SORT}
	RelationSort      *SortKind = &SortKind{rawSortKind: C.Z3_RELATION_SORT}
	FiniteDomainSort  *SortKind = &SortKind{rawSortKind: C.Z3_FINITE_DOMAIN_SORT}
	FloatingPointSort *SortKind = &SortKind{rawSortKind: C.Z3_FLOATING_POINT_SORT}
	RoundingModeSort  *SortKind = &SortKind{rawSortKind: C.Z3_ROUNDING_MODE_SORT}
	SeqSort           *SortKind = &SortKind{rawSortKind: C.Z3_SEQ_SORT}
	ReSort            *SortKind = &SortKind{rawSortKind: C.Z3_RE_SORT}
	CharSort          *SortKind = &SortKind{rawSortKind: C.Z3_CHAR_SORT}
	UnknownSort       *SortKind = &SortKind{rawSortKind: C.Z3_UNKNOWN_SORT}
)

// Sort represents a sort in Z3.
type Sort struct {
	ctx     *Context
	rawSort C.Z3_sort
}

// BoolSort returns the boolean type.
func (c *Context) BoolSort() *Sort {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &Sort{
		ctx:     c,
		rawSort: C.Z3_mk_bool_sort(c.Raw),
	}
}

// IntSort returns the int type.
func (c *Context) IntSort() *Sort {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &Sort{
		ctx:     c,
		rawSort: C.Z3_mk_int_sort(c.Raw),
	}
}

func (c *Context) RealSort() *Sort {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return &Sort{
		ctx:     c,
		rawSort: C.Z3_mk_real_sort(c.Raw),
	}
}

func (s *Sort) Kind() *SortKind {
	s.ctx.Lock()
	defer s.ctx.Unlock()
	return &SortKind{
		rawSortKind: C.Z3_get_sort_kind(s.ctx.Raw, s.rawSort),
	}
}
