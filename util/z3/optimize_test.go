package z3

import "testing"

func TestOptimize(t *testing.T) {
	config := NewConfig()
	defer config.Close()

	context := NewContext(config)
	defer context.Close()

	x := context.Const(context.Symbol("x"), context.IntSort())
	y := context.Const(context.Symbol("y"), context.IntSort())

	opt := context.NewOptimizer()

	opt.Assert(x.Add(y).Le(context.Int(10)))
	opt.Assert(x.Add(y).Gt(context.Int(0)))

	t.Log("checking assert")
	if opt.Check() != True {
		t.Fatalf("could not solve the constraints")
	}
	m := opt.Model()
	defer m.Close()
}

func TestOptimizeMaximize(t *testing.T) {
	config := NewConfig()
	defer config.Close()

	context := NewContext(config)
	defer context.Close()

	x := context.Const(context.Symbol("x"), context.IntSort())
	y := context.Const(context.Symbol("y"), context.IntSort())

	opt := context.NewOptimizer()

	opt.Assert(x.Add(y).Le(context.Int(10)))
	opt.Assert(x.Add(y).Gt(context.Int(0)))

	max := opt.Maximize(x.Add(y))

	if opt.Check() != True {
		t.Fatalf("could not solve the constraints")
	}
	m := opt.Model()
	defer m.Close()

	v, ok := max.Value()
	if !ok || v != 10 {
		t.Logf("\nModel:\n%s", m.String())
		t.Fatalf("did not maximize")
	}
}

func TestOptimizeMimimize(t *testing.T) {
	config := NewConfig()
	defer config.Close()

	context := NewContext(config)
	defer context.Close()

	x := context.Const(context.Symbol("x"), context.IntSort())
	y := context.Const(context.Symbol("y"), context.IntSort())

	opt := context.NewOptimizer()

	opt.Assert(x.Add(y).Le(context.Int(10)))
	opt.Assert(x.Add(y).Gt(context.Int(0)))

	min := opt.Minimize(x.Add(y))

	t.Log("checking assert")
	if opt.Check() != True {
		t.Fatalf("could not solve the constraints")
	}
	m := opt.Model()
	defer m.Close()

	t.Log("Fetching minimal value")
	v, ok := min.Value()
	if !ok || v != 1 {
		t.Logf("\nModel:\n%s", m.String())
		t.Fatalf("did not minimize")
	}
}
