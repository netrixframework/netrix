// Package z3 provides Go bindings to the Z3 SMT Solver.
//
// The bindings are a balance between idiomatic Go and being an obvious
// translation from the Z3 API so you can look up Z3 APIs and find the
// intuitive mapping in Go.
//
// The most foreign thing to Go programmers will be error handling. Rather
// than return the `error` type from almost every function, the z3 package
// mimics Z3's API by requiring you to set an error handler callback. This
// error handler will be invoked whenever an error occurs. See
// ErrorHandler and Context.SetErrorHandler for more information.
package z3

// #cgo LDFLAGS: /usr/local/lib/z3/bin/libz3.a -lstdc++
// #include <stdlib.h>
// #include "go-z3.h"
import "C"
