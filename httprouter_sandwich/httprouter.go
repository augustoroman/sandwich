// Package httprouter_sandwich is a httprouter-adapter for sandwich that
// provides the httprouter path parameters to the middleware stack.
//
// The Middleware implementation in this package is identical to the normal
// sandwich.Middleware except:
//  - it provides the httprouter.Params type by default for accessing route parameters
//  - it has a .H field for getting the httprouter handler easily.
//
// Here's a simple example of using this:
//
//     s := httprouter_sandwich.TheUsual().Set(...).Then(...)
//     m := httprouter.New()
//     ...
//     m.GET("/user/:id/", s.Set(userdb).Then(getUser).H)
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
		chain.Func(sandwich.New()).Arg(
			(httprouter.Params)(nil)))
}

// TheUsual constructs a popular new Middleware instance with some delicious
// default handlers installed and ready to go:  Request logging and simple error
// handling.  It also provides httprouter's routing params (httprouter.Params).
func TheUsual() Middleware {
	return Middleware(
		chain.Func(sandwich.TheUsual()).Arg(
			(httprouter.Params)(nil)))
}

// Middleware is the stack of middleware functions that powers sandwich's
// deliciousness.  In addition to the default sandwich.Middleware, this supports
// the httprouter.Params type so you can get the route parameters in your handlers.
//
// httprouter expects a function rather than a particular interface, so use the
// .H accessor to get a httprouter-friendly handler.
type Middleware sandwich.Middleware

// H is the httprouter middleware handling function.  You normally won't call this
// function directly but rather you'll pass it to httprouter.
func (m Middleware) H(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	chain.Func(m).MustRun(w, r, p)
}

func (m Middleware) mw() sandwich.Middleware { return sandwich.Middleware(m) }

// ---------------------------------------------------------------------------
// Below are just wrappers for the sandwich.Middleware calls, but returning
// an httprouter_sandwich.Middleware wrapper instead.
// ---------------------------------------------------------------------------

// Set is the same as (sandwich.Middleware).Set, but returns an
// httprouter_sandwich.Middleware.
func (m Middleware) Set(val interface{}) Middleware {
	return Middleware(m.mw().Set(val))
}

// SetAs is the same as (sandwich.Middleware).SetAs, but returns an
// httprouter_sandwich.Middleware.
func (m Middleware) SetAs(val, ifacePtr interface{}) Middleware {
	return Middleware(m.mw().SetAs(val, ifacePtr))
}

// Then is the same as (sandwich.Middleware).Then, but returns an
// httprouter_sandwich.Middleware.
func (m Middleware) Then(handlers ...interface{}) Middleware {
	return Middleware(m.mw().Then(handlers...))
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
