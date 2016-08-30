// Package martini_sandwich is a martini-adapter for sandwich that provides the
// martini request parameters to the middleware stack.
package martini_sandwich

import (
	"net/http"

	"github.com/augustoroman/sandwich"
	"github.com/augustoroman/sandwich/chain"
	"github.com/go-martini/martini"
)

func New() Middleware {
	c := chain.Chain(sandwich.New()).
		Reserve((martini.Params)(nil)).
		Reserve((*martini.ResponseWriter)(nil))
	return Middleware(c)
}
func TheUsual() Middleware {
	c := chain.Chain(sandwich.TheUsual()).
		Reserve((martini.Params)(nil)).
		Reserve((*martini.ResponseWriter)(nil))
	return Middleware(c)
}

type Middleware sandwich.Middleware

func (m Middleware) w() sandwich.Middleware { return sandwich.Middleware(m) }

func (m Middleware) Provide(val interface{}) Middleware { return Middleware(m.w().Provide(val)) }
func (m Middleware) ProvideAs(val, ifacePtr interface{}) Middleware {
	return Middleware(m.w().ProvideAs(val, ifacePtr))
}

func (m Middleware) With(handlers ...interface{}) Middleware {
	return Middleware(m.w().With(handlers...))
}
func (m Middleware) OnErr(handler interface{}) Middleware { return Middleware(m.w().OnErr(handler)) }
func (m Middleware) Wrap(before, after interface{}) Middleware {
	return Middleware(m.w().Wrap(before, after))
}
func (m Middleware) ServeHTTP(w martini.ResponseWriter, r *http.Request, p martini.Params) {
	rw := http.ResponseWriter(w)
	err := chain.Chain(m).Run((*martini.ResponseWriter)(&w), (*http.ResponseWriter)(&rw), r, p)
	if err != nil {
		panic(err)
	}
}
func (m Middleware) Code(pkg, funcName string) string { return m.w().Code(pkg, funcName) }
