// Package martini_sandwich is a martini-adapter for sandwich that provides the
// martini request parameters to the middleware stack.
package martini_sandwich

import (
	"github.com/augustoroman/sandwich"
	"github.com/augustoroman/sandwich/chain"
	"github.com/go-martini/martini"

	"net/http"
)

func Adapt(mw sandwich.Middleware) sandwich.Middleware {
	c := chain.Chain(mw).
		Reserve((martini.Params)(nil)).
		Reserve((*martini.ResponseWriter)(nil))
	return sandwich.Middleware(c)
}

func Handler(mw sandwich.Middleware) func(w martini.ResponseWriter, r *http.Request, p martini.Params) {
	return func(w martini.ResponseWriter, r *http.Request, p martini.Params) {
		err := chain.Chain(mw).Run((*martini.ResponseWriter)(&w), (*http.ResponseWriter)(&w), r, p)
		if err != nil {
			panic(err)
		}
	}
}
