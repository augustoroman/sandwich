// Package httprouter_sandwich is a httprouter-adapter for sandwich that
// provides the httprouter path parameters to the middleware stack.
package httprouter_sandwich

import (
	"net/http"

	"github.com/augustoroman/sandwich"
	"github.com/augustoroman/sandwich/chain"
	"github.com/julienschmidt/httprouter"
)

// New constructs a clean Middleware instance that provides httprouter's routing
// params (httprouter.Params), ready for you to start piling on the handlers.
func New() Middleware {
	return Middleware(
		chain.Chain(sandwich.New()).Reserve(
			(httprouter.Params)(nil)))
}

// TheUsual constructs a popular new Middleware instance with some delicious
// default handlers installed and ready to go:  Request logging and simple error
// handling.  It also provides httprouter's routing params (httprouter.Params).
func TheUsual() Middleware {
	return Middleware(
		chain.Chain(sandwich.TheUsual()).Reserve(
			(httprouter.Params)(nil)))
}

// Middleware is the stack of middleware functions that powers sandwich's
// deliciousness.  In addition to the default sandwich.Middleware, this supports
// the httprouter.Params type so you can get the route parameters in your handlers.
//
// httprouter expects a function rather than a particular interface, so use the
// .H accessor to get a httprouter-friendly handler.  For example:
//
//     s := httprouter_sandwich.TheUsual().Provide(...).With(...)
//     m := httprouter.New()
//     ...
//     m.GET("/user/:id/", s.Provide(userdb).With(getUser).H)
//     ...
//
//     func getUser(w http.ResponseWriter, p httprouter.Params, udb UserDb) error {
//         userId := p["id"]
//         user, err := udb.Lookup(userId)
//         if err != nil {
//             return err // or wrap with sandwich.Error{...}
//         }
//         return json.NewEncoder(w).Encode(user)
//     }
type Middleware sandwich.Middleware

// H is the httprouter middleware handling function.  You normally won't call this
// function directly but rather you'll pass it to httprouter.
func (m Middleware) H(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	err := chain.Chain(m).Run((*http.ResponseWriter)(&w), r, p)
	if err != nil {
		panic(err)
	}
}

func (m Middleware) mw() sandwich.Middleware { return sandwich.Middleware(m) }

// ---------------------------------------------------------------------------
// Below are just wrappers for the sandwich.Middleware calls, but returning
// an httprouter_sandwich.Middleware wrapper instead.
// ---------------------------------------------------------------------------

// Provide is the same as (sandwich.Middleware).Provide, but returns an
// httprouter_sandwich.Middleware.
func (m Middleware) Provide(val interface{}) Middleware {
	return Middleware(m.mw().Provide(val))
}

// ProvideAs is the same as (sandwich.Middleware).ProvideAs, but returns an
// httprouter_sandwich.Middleware.
func (m Middleware) ProvideAs(val, ifacePtr interface{}) Middleware {
	return Middleware(m.mw().ProvideAs(val, ifacePtr))
}

// With is the same as (sandwich.Middleware).With, but returns an
// httprouter_sandwich.Middleware.
func (m Middleware) With(handlers ...interface{}) Middleware {
	return Middleware(m.mw().With(handlers...))
}

// OnErr is the same as (sandwich.Middleware).OnErr, but returns an
// httprouter_sandwich.Middleware.
func (m Middleware) OnErr(handler interface{}) Middleware {
	return Middleware(m.mw().OnErr(handler))
}

// Wrap is the same as (sandwich.Middleware).Wrap, but returns an
// httprouter_sandwich.Middleware.
func (m Middleware) Wrap(before, after interface{}) Middleware {
	return Middleware(m.mw().Wrap(before, after))
}

// Code is the same as (sandwich.Middleware).Code.
func (m Middleware) Code(pkg, funcName string) string {
	return m.mw().Code(pkg, funcName)
}
