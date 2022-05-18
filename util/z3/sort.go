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
	rawCtx  C.Z3_context
	rawSort C.Z3_sort
}

// BoolSort returns the boolean type.
func (c *Context) BoolSort() *Sort {
	return &Sort{
		rawCtx:  c.raw,
		rawSort: C.Z3_mk_bool_sort(c.raw),
	}
}

// IntSort returns the int type.
func (c *Context) IntSort() *Sort {
	return &Sort{
		rawCtx:  c.raw,
		rawSort: C.Z3_mk_int_sort(c.raw),
	}
}

func (c *Context) RealSort() *Sort {
	return &Sort{
		rawCtx:  c.raw,
		rawSort: C.Z3_mk_real_sort(c.raw),
	}
}

func (s *Sort) Kind() *SortKind {
	return &SortKind{
		rawSortKind: C.Z3_get_sort_kind(s.rawCtx, s.rawSort),
	}
}
