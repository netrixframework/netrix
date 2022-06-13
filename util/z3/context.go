package z3

// #include "go-z3.h"
import "C"
import "sync"

// Context is what handles most of the interactions with Z3.
type Context struct {
	Raw    C.Z3_context
	rawCfg C.Z3_config
	*sync.Mutex
}

func NewContext(c *Config) *Context {
	return &Context{
		Raw:    C.Z3_mk_context(c.raw),
		rawCfg: c.raw,
		Mutex:  new(sync.Mutex),
	}
}

// Close frees the memory associated with this context.
func (c *Context) Close() error {
	c.Mutex.Lock()
	// Clear context
	C.Z3_del_context(c.Raw)

	// Clear error handling
	errorHandlerMapLock.Lock()
	delete(errorHandlerMap, c.Raw)
	errorHandlerMapLock.Unlock()
	c.Mutex.Unlock()

	return nil
}

func (c *Context) Reset() {
	c.Mutex.Lock()
	C.Z3_del_context(c.Raw)
	newRawCtx := C.Z3_mk_context(c.rawCfg)
	c.Raw = newRawCtx
	c.Mutex.Unlock()
}

func (c *Context) ResetWithConfig(config *Config) {
	c.Mutex.Lock()
	C.Z3_del_context(c.Raw)
	newRawCtx := C.Z3_mk_context(config.raw)
	c.Raw = newRawCtx
	c.rawCfg = config.raw
	c.Mutex.Unlock()
}
