package sandwich

import (
	"bytes"
	"net/http"

	"github.com/augustoroman/sandwich/chain"
)

// New constructs a clean Middleware instance, ready for you to start piling on
// the handlers.
func New() Middleware {
	return Middleware(
		chain.Chain{}.
			Reserve((*http.ResponseWriter)(nil)).
			Reserve((*http.Request)(nil)),
	)
}

// TheUsual constructs a popular new Middleware instance with some delicious
// default handlers installed and ready to go:  Request logging and simple error
// handling.
func TheUsual() Middleware {
	return New().
		With(WrapResponseWriter).
		Wrap(StartLog, (*LogEntry).Commit).
		OnErr(HandleError)
}

// Middleware is the stack of middleware functions that powers sandwich's
// deliciousness.  It implements http.ServeHTTP and is immutable: all mutating
// functions create a new instance.
type Middleware chain.Chain

func (m Middleware) chain() chain.Chain { return chain.Chain(m) }

// Provide makes the specified value available as an input parameter to
// subsequent handlers.  If the same type has already been provided (either by
// previous Provide(...) calls or as the return value of a handler), the old
// value will be replaced by the newly provided value.
func (m Middleware) Provide(val interface{}) Middleware {
	return Middleware(m.chain().Provide(val))
}

// ProvideAs makes the specified value available under the interface type
// pointed to by ifacePtr.  Because interfaces in Go are only used for static
// typing, a pointer to the interface must be provided.   This is typically done
// with a nil pointer, as in:
//
//   m.ProvideAs(myConcreteImpl, (*someInterface)(nil))
func (m Middleware) ProvideAs(val, ifacePtr interface{}) Middleware {
	return Middleware(m.chain().ProvideAs(val, ifacePtr))
}

// With adds one or more middleware handles to the stack.  Middleware handlers
// may use any of the values previously provided either directly via
// Provide(...) or from the return values of previous handlers in the entire
// middleware stack.
func (m Middleware) With(handlers ...interface{}) Middleware {
	return Middleware(m.chain().With(handlers...))
}

// OnErr adds a new error handler to the middleware stack.  The error handler
// will handle any errors from subsequent middleware handles -- it will not
// affect any handlers previously added to the middleware stack.  Error handlers
// may not return any values.
func (m Middleware) OnErr(handler interface{}) Middleware {
	return Middleware(m.chain().OnErr(handler))
}

// Wrap adds two handlers: one that is called during the normal middleware
// progression ('before') and one that is deferred until all other middleware
// handlers and error handlers ('after') have been called.  Deferred handlers
// are called in the reverse order that they are added, and may not return any
// values.  The 'after' handler will only be run if the 'before' handler has run
// -- if the middleware chain is aborted before getting to 'before' (or if
// 'before' returns an error), 'after' will not be called.  The 'before' handler
// may be nil in which case only the after handler will be registered.
func (m Middleware) Wrap(before, after interface{}) Middleware {
	c := m.chain()
	if before != nil {
		c = c.With(before)
	}
	c = c.Defer(after)
	return Middleware(c)
}

// ServerHTTP implements the http.Handler interface and provides the initial
// http.ResponseWriter and *http.Request to the middleware chain.
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := m.chain().Run((*http.ResponseWriter)(&w), r); err != nil {
		panic(err) // This should never happen.
	}
}

// Code generates and returns pure-Go code for this middleware stack.
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
	m.chain().Code(funcName, pkg, &buf)
	return buf.String()
}
