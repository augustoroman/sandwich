package sandwich

import (
	"net/http"

	"github.com/augustoroman/sandwich/chain"
)

// ChainMutation is a special type that allows modifying the chain directly when
// added to a router. This allows advanced usage and should generally not be
// used unless you know what you're doing. In particular, don't add `Arg`s to
// the chain, that will break the router.
type ChainMutation interface {
	// Modify the provided chain and return the modified chain.
	Apply(c chain.Func) chain.Func
}

// Wrap provides a mechanism to add two handlers: one that runs during the
// normal course of middleware handling (Before) and one that is defer'd and
// runs after the main set of middleware has executed (After). The defer'd
// handler may accept the `error` type and handle or ignore errors as desired.
//
// This is generally useful for specifying operations that need to run before
// and after subsequent middleware, such as timing, logging, or
// allocation/cleanup operations.
type Wrap struct {
	// `Before` is run in the normal course of middleware evaluation. Any returned
	// types from this will be available to the defer'd After handler. If Before
	// itself returns an error, After will not run.
	Before any
	// `After` is defer`d and run after the normal course of middleware has
	// completed, in reverse order of any registered `defer` handlers. Defer`d
	// handlers will always be executed if `Before` was executed, even in the case
	// of errors. The `After` handler may accept the `error` type -- that will be
	// nil unless a subsequent handler has returned an error.
	After any
}

// Apply modifies the chain to add Before and After.
func (w Wrap) Apply(c chain.Func) chain.Func {
	return c.Then(toHandlerFunc(w.Before)).Defer(toHandlerFunc(w.After))
}

func apply(c chain.Func, handlers ...any) chain.Func {
	for _, h := range handlers {
		if mod, ok := h.(ChainMutation); ok {
			c = mod.Apply(c)
		} else {
			c = c.Then(toHandlerFunc(h))
		}
	}
	return c
}

func toHandlerFunc(h any) any {
	if handlerInterface, ok := h.(http.Handler); ok {
		return handlerInterface.ServeHTTP
	}
	return h
}
