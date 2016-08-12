// Package httprouter_sandwich is a httprouter-adapter for sandwich that
// provides the httprouter path parameters to the middleware stack.
package httprouter_sandwich

import (
	"github.com/augustoroman/sandwich"
	"github.com/augustoroman/sandwich/chain"
	"github.com/julienschmidt/httprouter"

	"net/http"
)

func New() Middleware {
	return Middleware{
		chain.Chain{}.
			Reserve((*http.ResponseWriter)(nil)).
			Reserve((*http.Request)(nil)).
			Reserve((httprouter.Params)(nil)),
	}
}

func TheUsual() Middleware {
	return New().
		With(sandwich.WrapResponseWriter).
		Wrap(sandwich.StartLog, (*sandwich.LogEntry).Commit).
		OnErr(sandwich.HandleError)
}

type Middleware struct{ c chain.Chain }

func (m Middleware) Provide(val interface{}) Middleware { return Middleware{m.c.Provide(val)} }
func (m Middleware) ProvideAs(val, ifacePtr interface{}) Middleware {
	return Middleware{m.c.ProvideAs(val, ifacePtr)}
}
func (m Middleware) With(handlers ...interface{}) Middleware { return Middleware{m.c.With(handlers...)} }
func (m Middleware) OnErr(handler interface{}) Middleware    { return Middleware{m.c.OnErr(handler)} }
func (m Middleware) Wrap(before, after interface{}) Middleware {
	c := m.c
	if before != nil {
		c = c.With(before)
	}
	c = c.Defer(after)
	return Middleware{c}
}
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := m.c.Run((*http.ResponseWriter)(&w), r, p); err != nil {
		panic(err)
	}
}
