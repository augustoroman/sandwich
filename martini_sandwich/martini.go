// Package martini_sandwich is a martini-adapter for sandwich that provides the
// martini request parameters to the middleware stack.
package martini_sandwich

import (
	"github.com/augustoroman/sandwich"
	"github.com/augustoroman/sandwich/chain"
	"github.com/go-martini/martini"

	"net/http"
)

func New() Middleware {
	return Middleware{
		chain.Chain{}.
			Reserve((*http.ResponseWriter)(nil)).
			Reserve((*http.Request)(nil)).
			Reserve((martini.Params)(nil)),
	}
}

func TheUsual() Middleware {
	return New().
		Then(sandwich.WrapResponseWriter, sandwich.StartLog).
		Defer((*sandwich.LogEntry).Commit).
		OnErr(sandwich.HandleError)
}

type Middleware struct{ c chain.Chain }

func (m Middleware) Provide(val interface{}) Middleware { return Middleware{m.c.Provide(val)} }
func (m Middleware) ProvideAs(val, ifacePtr interface{}) Middleware {
	return Middleware{m.c.ProvideAs(val, ifacePtr)}
}
func (m Middleware) Then(handlers ...interface{}) Middleware { return Middleware{m.c.Then(handlers...)} }
func (m Middleware) OnErr(handler interface{}) Middleware    { return Middleware{m.c.OnErr(handler)} }
func (m Middleware) Defer(handler interface{}) Middleware    { return Middleware{m.c.Defer(handler)} }
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, p martini.Params) {
	if err := m.c.Run((*http.ResponseWriter)(&w), r, p); err != nil {
		panic(err)
	}
}
