// Package martini_sandwich is a martini-adapter for sandwich that provides the
// martini request parameters to the middleware stack.
//
// The Middleware implementation in this package is identical to the normal
// sandwich.Middleware except:
//  - it provides the martini.Params type by default for accessing route parameters
//  - it has a .H field for getting the martini handler easily.
//
// Here's a simple example of using this:
//
//     s := martini_sandwich.TheUsual().Set(...).Then(...)
//     m := martini.Classic()
//     ...
//     m.Get("/user/:id/", s.Set(userdb).Then(getUser).H) // <-- Note the .H
//     ...
//
//     func getUser(w http.ResponseWriter, p martini.Params, udb UserDb) error {
//         userId := p["id"]
//         user, err := udb.Lookup(userId)
//         if err != nil {
//             return err // or wrap with sandwich.Error{...}
//         }
//         return json.NewEncoder(w).Encode(user)
//     }
//
package martini_sandwich

import (
	"net/http"

	"github.com/augustoroman/sandwich"
	"github.com/augustoroman/sandwich/chain"
	"github.com/go-martini/martini"
)

// New constructs a clean Middleware instance that provides martini's routing
// params (martini.Params), ready for you to start piling on the handlers.
func New() Middleware {
	c := chain.Func(sandwich.New()).
		Arg((martini.Params)(nil))
	return Middleware(c)
}

// TheUsual constructs a popular new Middleware instance with some delicious
// default handlers installed and ready to go:  Request logging and simple error
// handling.  It also provides martini's routing params (martini.Params).
func TheUsual() Middleware {
	c := chain.Func(sandwich.TheUsual()).
		Arg((martini.Params)(nil))
	return Middleware(c)
}

// Middleware is the stack of middleware functions that powers sandwich's
// deliciousness.  In addition to the default sandwich.Middleware, this supports
// the martini.Params type so you can get the route parameters in your handlers.
//
// Martini expects a function rather than a particular interface, so use the
// .H accessor to get a martini-friendly handler.
type Middleware sandwich.Middleware

// H is the martini middleware handling function.  You normally won't call this
// function directly but rather you'll pass it to martini.
func (m Middleware) H(w http.ResponseWriter, r *http.Request, p martini.Params) {
	err := chain.Func(m).Run(w, r, p)
	if err != nil {
		panic(err)
	}
}

func (m Middleware) mw() sandwich.Middleware { return sandwich.Middleware(m) }

// ---------------------------------------------------------------------------
// Below are just wrappers for the sandwich.Middleware calls, but returning
// a martini_sandwich.Middleware wrapper instead.
// ---------------------------------------------------------------------------

// Set is the same as (sandwich.Middleware).Set, but returns a
// martini_sandwich.Middleware.
func (m Middleware) Set(val interface{}) Middleware {
	return Middleware(m.mw().Set(val))
}

// SetAs is the same as (sandwich.Middleware).SetAs, but returns a
// martini_sandwich.Middleware.
func (m Middleware) SetAs(val, ifacePtr interface{}) Middleware {
	return Middleware(m.mw().SetAs(val, ifacePtr))
}

// Then is the same as (sandwich.Middleware).Then, but returns a
// martini_sandwich.Middleware.
func (m Middleware) Then(handlers ...interface{}) Middleware {
	return Middleware(m.mw().Then(handlers...))
}

// OnErr is the same as (sandwich.Middleware).OnErr, but returns a
// martini_sandwich.Middleware.
func (m Middleware) OnErr(handler interface{}) Middleware {
	return Middleware(m.mw().OnErr(handler))
}

// Wrap is the same as (sandwich.Middleware).Wrap, but returns a
// martini_sandwich.Middleware.
func (m Middleware) Wrap(before, after interface{}) Middleware {
	return Middleware(m.mw().Wrap(before, after))
}

// Code is the same as (sandwich.Middleware).Code.
func (m Middleware) Code(pkg, funcName string) string {
	return m.mw().Code(pkg, funcName)
}
