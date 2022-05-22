package z3

import (
	"testing"
)

func TestASTReal(t *testing.T) {
	config := NewConfig()
	ctx := NewContext(config)
	config.Close()
	defer ctx.Close()
	solver := ctx.NewSolver()

	x := ctx.Const(ctx.Symbol("x"), ctx.IntSort())
	y := ctx.Const(ctx.Symbol("y"), ctx.IntSort())

	solver.Assert(
		x.Add(ctx.Real(3, 4).Mul(y)).Le(ctx.Int(0)),
	)

	solver.Check()
}

func TestASTAdd(t *testing.T) {
	config := NewConfig()
	defer config.Close()

	ctx := NewContext(config)
	defer ctx.Close()

	// Create an int
	v1 := ctx.Int(1)
	v2 := ctx.Int(2)
	v3 := ctx.Int(3)

	// Add
	raw := v1.Add(v2, v3)

	actual := raw.String()
	if actual != "(+ 1 2 3)" {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestASTMul(t *testing.T) {
	config := NewConfig()
	defer config.Close()

	ctx := NewContext(config)
	defer ctx.Close()

	// Create an int
	v1 := ctx.Int(1)
	v2 := ctx.Int(2)
	v3 := ctx.Int(3)

	// Mul
	raw := v1.Mul(v2, v3)

	actual := raw.String()
	if actual != "(* 1 2 3)" {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestASTSub(t *testing.T) {
	config := NewConfig()
	defer config.Close()

	ctx := NewContext(config)
	defer ctx.Close()

	// Create an int
	v1 := ctx.Int(1)
	v2 := ctx.Int(2)
	v3 := ctx.Int(3)

	// Sub
	raw := v1.Sub(v2).Sub(v3)

	actual := raw.String()
	if actual != "(- (- 1 2) 3)" {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestASTLt(t *testing.T) {
	config := NewConfig()
	defer config.Close()
	ctx := NewContext(config)
	defer ctx.Close()

	// Create an int
	x := ctx.Const(ctx.Symbol("x"), ctx.IntSort())
	y := ctx.Const(ctx.Symbol("y"), ctx.IntSort())

	// Add
	raw := x.Lt(y)

	actual := raw.String()
	if actual != "(< x y)" {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestASTLe(t *testing.T) {
	config := NewConfig()
	defer config.Close()
	ctx := NewContext(config)
	defer ctx.Close()

	// Create an int
	x := ctx.Const(ctx.Symbol("x"), ctx.IntSort())
	y := ctx.Const(ctx.Symbol("y"), ctx.IntSort())

	// Add
	raw := x.Le(y)

	actual := raw.String()
	if actual != "(<= x y)" {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestASTGt(t *testing.T) {
	config := NewConfig()
	defer config.Close()
	ctx := NewContext(config)
	defer ctx.Close()

	// Create an int
	x := ctx.Const(ctx.Symbol("x"), ctx.IntSort())
	y := ctx.Const(ctx.Symbol("y"), ctx.IntSort())

	// Add
	raw := x.Gt(y)

	actual := raw.String()
	if actual != "(> x y)" {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestASTGe(t *testing.T) {
	config := NewConfig()
	defer config.Close()
	ctx := NewContext(config)
	defer ctx.Close()

	// Create an int
	x := ctx.Const(ctx.Symbol("x"), ctx.IntSort())
	y := ctx.Const(ctx.Symbol("y"), ctx.IntSort())

	// Add
	raw := x.Ge(y)

	actual := raw.String()
	if actual != "(>= x y)" {
		t.Fatalf("bad:\n%s", actual)
	}
}
