// Package httprouter_sandwich is a httprouter-adapter for sandwich that
// provides the httprouter path parameters to the middleware stack.
package httprouter_sandwich

import (
	"net/http"

	"github.com/augustoroman/sandwich"
	"github.com/augustoroman/sandwich/chain"
	"github.com/julienschmidt/httprouter"
)

func New() Middleware {
	return Middleware(
		chain.Chain(sandwich.New()).Reserve(
			(httprouter.Params)(nil)))
}
func TheUsual() Middleware {
	return Middleware(
		chain.Chain(sandwich.TheUsual()).Reserve(
			(httprouter.Params)(nil)))
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
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	err := chain.Chain(m).Run((*http.ResponseWriter)(&w), r, p)
	if err != nil {
		panic(err)
	}
}
func (m Middleware) Code(pkg, funcName string) string { return m.w().Code(pkg, funcName) }
