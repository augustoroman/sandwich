package sandwich

import (
	"bytes"
	"github.com/augustoroman/sandwich/chain"

	"net/http"
)

// New constructs a clean Middleware instance, ready for you to start piling on
// the handlers.
func New() Middleware {
	return Middleware{
		chain.Chain{}.
			Reserve((*http.ResponseWriter)(nil)).
			Reserve((*http.Request)(nil)),
	}
}

// TheUsual constructs a popular new Middleware instance with some delicious
// default handlers installed and ready to go:  Request logging and simple error
// handling.
func TheUsual() Middleware {
	return New().
		Then(WrapResponseWriter, StartLog).
		Defer((*LogEntry).Commit).
		OnErr(HandleError)
}

// Middleware is the stack of middleware functions that pwoers sandwich's
// deliciousness.  It implements http.ServeHTTP and is immutable: all mutating
// functions create a new instance.
type Middleware struct{ c chain.Chain }

// Provide makes the specified value available as an input parameter to
// subsequent handlers.  If the same type has already been provided (either by
// previous Provide(...) calls or as the return value of a handler), the old
// value will be replaced by the newly provided value.
func (m Middleware) Provide(val interface{}) Middleware {
	return Middleware{m.c.Provide(val)}
}

// ProvideAs makes the specified value available under the interface type
// pointed to by ifacePtr.  Because interfaces in Go are only used for static
// typing, a pointer to the interface must be provided.   This is typically done
// with a nil pointer, as in:
//
//   m.ProvideAs(myConcreteImpl, (*someInterface)(nil))
func (m Middleware) ProvideAs(val, ifacePtr interface{}) Middleware {
	return Middleware{m.c.ProvideAs(val, ifacePtr)}
}

// Then adds one or more middleware handles to the stack.  Middleware handlers
// may use any of the values previously provided either directly via
// Provide(...) or from the return values of previous handlers in the entire
// middleware stack.
func (m Middleware) Then(handlers ...interface{}) Middleware {
	return Middleware{m.c.Then(handlers...)}
}

// OnErr adds a new error handler to the middleware stack.  The error handler
// will handle any errors from subsequent middleware handles -- it will not
// affect any handlers previously added to the middleware stack.  Error handlers
// may not return any values.
func (m Middleware) OnErr(handler interface{}) Middleware {
	return Middleware{m.c.OnErr(handler)}
}

// Defer adds a handler to be called after all normal handlers (and any error
// handler, if applicable) have been called.  Defers are executed in reverse
// order that they are added to the stack, analogous to Go's defer
// functionality.  If a middleware handler errors out and the middleware chain
// is aborted before it gets to the deferred handler, it will not be called.
// Defer'd handlers may not return any values.
func (m Middleware) Defer(handler interface{}) Middleware {
	return Middleware{m.c.Defer(handler)}
}

// ServerHTTP implements the http.Handler interface and provides the initial
// http.ResponseWriter and *http.Request to the middleware chain.
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := m.c.Run((*http.ResponseWriter)(&w), r); err != nil {
		panic(err) // This should never happen.
	}
}

// Code generates and returns pure-Go code for this middeware stack.
// Dependency-injection too magical for you?  Reflection too slow?  Auto-
// generate the code for the handler and use that instead!  You must provide
// the package name that the generated code should be in as well as the overall
// function name for the generated handler.
//
// The generated function takes all of the provided variables in the middleware
// stack and returns an http.HandlerFunc, like:
//
//   func NAME(provided vars...) func(w http.ResponseWriter, r *http.Request) {
//       return func(w http.ResponseWriter, r *http.Request) {
//         ...
//       }
//   }
func (m Middleware) Code(pkg, funcName string) string {
	var buf bytes.Buffer
	m.c.Code(funcName, pkg, &buf)
	return buf.String()
}
