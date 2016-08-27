// Package httprouter_sandwich is a httprouter-adapter for sandwich that
// provides the httprouter path parameters to the middleware stack.
package httprouter_sandwich

import (
	"net/http"

	"github.com/augustoroman/sandwich"
	"github.com/augustoroman/sandwich/chain"
	"github.com/julienschmidt/httprouter"
)

func Adapt(mw sandwich.Middleware) sandwich.Middleware {
	return sandwich.Middleware(chain.Chain(mw).Reserve((httprouter.Params)(nil)))
}

func Handler(mw sandwich.Middleware) func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		err := chain.Chain(mw).Run((*http.ResponseWriter)(&w), r, p)
		if err != nil {
			panic(err)
		}
	}
}
