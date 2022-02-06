package chain

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func New() Func { return Func{} }

func TestInitialInjection(t *testing.T) {
	var args []interface{}
	recordArgs := func(a int, b string) { args = append(args, a, b) }

	err := New().
		Arg(0).
		Arg("").
		Then(recordArgs).
		Set(3).
		Set("four").
		Then(recordArgs).
		Run(1, "two")
	assert.NoError(t, err)

	assert.EqualValues(t, []interface{}{1, "two", 3, "four"}, args)
}

func TestInitialDeferredInjection(t *testing.T) {
	var args []interface{}
	recordArgs := func(a int, b string) { args = append(args, a, b) }

	err := New().Arg(0).Arg("").Then(recordArgs).Run(2, "xyz")
	assert.NoError(t, err)

	assert.EqualValues(t, []interface{}{2, "xyz"}, args)
}

func TestDeferredExecutionOrder(t *testing.T) {
	var buf bytes.Buffer
	say := func(s string) func() { return func() { buf.WriteString(s + ":") } }
	err := New().
		Then(say("a"), say("b")).
		Defer(say("f")).
		Defer(say("e")).
		Then(say("c")).
		Defer(say("d")).
		Run()
	assert.NoError(t, err)
	assert.Equal(t, "a:b:c:d:e:f:", buf.String())
}

func TestDeferredExecutionOrderWithErrors(t *testing.T) {
	var buf bytes.Buffer
	say := func(s string) func() { return func() { buf.WriteString(s + ":") } }
	onErr := func(e error) { buf.WriteString("err[" + e.Error() + "]:") }
	fail := func() error { return errors.New("failed") }
	err := New().
		Then(say("a"), say("b")).
		Defer(say("f")).
		OnErr(onErr).
		Defer(say("e")).
		Then(fail).
		Then(say("c")).
		Defer(say("d")).
		Run()
	assert.NoError(t, err)
	assert.Equal(t, "a:b:err[failed]:e:f:", buf.String())
}

func TestBasicFuncExecution(t *testing.T) {
	a, b := 5, "hi"
	provide_initial := func() (*int, *string) { return &a, &b }
	verify_injected := func(x *int, y *string) {
		if *x != 5 {
			t.Errorf("Expected *int to be 6, got %d", *x)
		}
		if *y != "hi" {
			t.Errorf("Expected *string to be 'hi', got %s", *y)
		}
	}
	modify_injected := func(x *int, y *string) { *x = 6; *y = "bye" }

	err := New().Then(provide_initial, verify_injected, modify_injected).Run()
	assert.NoError(t, err)

	assert.Equal(t, 6, a)
	assert.Equal(t, "bye", b)
}

func TestSimpleErrorHandling(t *testing.T) {
	var out string
	chain := New().Arg("").Arg(0).
		OnErr(func(err error) { out += "First error handler: " + err.Error() }).
		Then(
			func(val string) (string, error) {
				if val == "foo" {
					return "bar", nil
				} else {
					return "", fmt.Errorf("%q is not foo", val)
				}
			}).
		OnErr(func(err error) { out += "Second error handler: " + err.Error() }).
		Then(
			func(num int) error {
				if num != 3 {
					return fmt.Errorf("%d is not 3", num)
				}
				return nil
			})

	out = ""
	assert.NoError(t, chain.Run("", 0))
	assert.Equal(t, `First error handler: "" is not foo`, out)

	out = ""
	assert.NoError(t, chain.Run("foo", 7))
	assert.Equal(t, `Second error handler: 7 is not 3`, out)

	out = ""
	assert.NoError(t, chain.Run("foo", 3))
	assert.Equal(t, ``, out)
}

func TestMustProvideTypes(t *testing.T) {
	assert.Panics(t, func() { New().Then(func(string) {}) })
	assert.NotPanics(t, func() { New().Set("").Then(func(string) {}) })

	assert.Panics(t, func() { New().Then(func(int, string) {}) })
	assert.NotPanics(t, func() { New().Set("").Set(3).Then(func(int, string) {}) })

	assert.NotPanics(t, func() {
		New().Then(
			func() string { return "" },
			func(string) int { return 3 },
			func(int) {},
		)
	}, "Should be OK: Everything is provided by earlier functions.")
	assert.NotPanics(t, func() {
		New().Then(
			func() string { return "" },
			func(string) int { return 3 },
		).OnErr(func(int, error) {})
	}, "Should be OK: Everything is provided by earlier functions")

	assert.Panics(t, func() {
		New().Then(
			func() string { return "" },
			func(string) int { return 3 },
			func(bool) {},
			func(int) {},
		)
	}, "Should FAIL: bool isn't provided anywhere")
	assert.Panics(t, func() {
		New().Then(func() string { return "" }, func(string) int { return 3 }).
			OnErr(func(bool, error) {}).
			Then(func(int) {})
	}, "Should FAIL: bool isn't provided anywhere (even error handlers need proper provisioning)")
}

func TestErrorAbortsHandling(t *testing.T) {
	var out string
	err := New().OnErr(func(err error) { out += "Failed @ " + err.Error() }).Then(
		func() error { out += "1 "; return nil },
		func() error { out += "2 "; return fmt.Errorf("2") },
		func() error { out += "3 "; return nil },
	).Run()
	assert.NoError(t, err)
	assert.Equal(t, "1 2 Failed @ 2", out)
}

func a() string                { return "hello " }
func b(s string) (string, int) { return s + "world", 42 }
func c(s string, n int)        {}

func TestCatchesPanics(t *testing.T) {
	var err error
	captureError := func(e error) { err = e }
	panics := func() { panic("ahhhh! 🔥") }

	assert.NoError(t,
		New().OnErr(captureError).Then(a, b, c).Defer(c).Then(panics).Run())

	assert.NotNil(t, err)

	e := err.(PanicError)
	assert.Equal(t, e.Val, "ahhhh! 🔥")
	assert.Equal(t, len(e.MiddlewareStack), 4) // defers haven't run yet.
	assert.Contains(t, e.MiddlewareStack[0].Name, "chain.TestCatchesPanics.func2")
	assert.Contains(t, e.MiddlewareStack[1].Name, "chain.c")
	assert.Contains(t, e.MiddlewareStack[2].Name, "chain.b")
	assert.Contains(t, e.MiddlewareStack[3].Name, "chain.a")

	assert.Contains(t, err.Error(), "Panic executing middleware")
	assert.Contains(t, err.Error(), "ahhhh! 🔥")
	// This is where the panic actually occurred. This will need to be updated if
	// this file changes, sadly.
	assert.Contains(t, err.Error(), "/home/aroman/code/sandwich/chain/chain_test.go:183")
	assert.Contains(t, err.Error(), "func() string")
	assert.Contains(t, err.Error(), "func(string) (string, int)")
	assert.Contains(t, err.Error(), "func(string, int)")
	assert.Contains(t, err.Error(), "chain.a")
	assert.Contains(t, err.Error(), "chain.b")
	assert.Contains(t, err.Error(), "chain.c")
}

func TestDefersCanAcceptErrors(t *testing.T) {
	var buf bytes.Buffer
	onerr := func(err error) { fmt.Fprintf(&buf, "onerr[%v]:", err) }
	deferred := func(err error) { fmt.Fprintf(&buf, "defer[%v]:", err) }
	fails := func() error { return errors.New("💣") }

	assert.NoError(t, New().
		OnErr(onerr).
		Then(a, b, c).
		Defer(deferred).
		Then(fails).
		Run())

	assert.Equal(t, "onerr[💣]:defer[💣]:", buf.String())

	// But what if nothing actually fails?  Defer's can still accept errors.
	buf.Reset()
	assert.NoError(t, New().
		OnErr(onerr).
		Then(a, b, c).
		Defer(deferred).
		// With(fails).  // no failure!
		Run())

	assert.Equal(t, "defer[<nil>]:", buf.String())
}

func TestDefaultErrorHandler(t *testing.T) {
	var buf bytes.Buffer
	onerr := func(err error) { fmt.Fprintf(&buf, "onerr[%v]:", err) }
	fails := func() error { return errors.New("☠") }

	// Restore the default error handler when we're done with the test.
	defer func(orig interface{}) { DefaultErrorHandler = orig }(DefaultErrorHandler)
	DefaultErrorHandler = onerr

	assert.NoError(t, New().Then(fails).Run())

	assert.Equal(t, "onerr[☠]:", buf.String())
}

func TestSetAs_Nil(t *testing.T) {
	worked := false
	check := func(s fmt.Stringer) {
		require.Nil(t, s)
		worked = true
	}
	err := New().SetAs(nil, (*fmt.Stringer)(nil)).Then(check).Run()
	require.NoError(t, err)
	require.True(t, worked)
}

func TestProvidingBadValues(t *testing.T) {
	assert.Panics(t, func() { New().Set(nil) })

	// ifacePtr must be a pointer to an interface
	assert.Panics(t, func() { New().SetAs(nil, 5) })
	type Struct struct{}
	assert.Panics(t, func() { New().SetAs(nil, Struct{}) })
	assert.Panics(t, func() { New().SetAs(nil, &Struct{}) })

	// SetAs value must actually implement the specified interface
	assert.Panics(t, func() { New().SetAs(5, (*fmt.Stringer)(nil)) })
	assert.Panics(t, func() { New().SetAs(Struct{}, (*fmt.Stringer)(nil)) })
}

func TestWithBadValues(t *testing.T) {
	type Struct struct{}
	assert.Panics(t, func() { New().Then(nil) })
	assert.Panics(t, func() { New().Then(5) })
	assert.Panics(t, func() { New().Then(Struct{}) })
}

func TestBadErrorHandler(t *testing.T) {
	//  The error handler must actually be a function
	assert.Panics(t, func() { New().OnErr(true) })
	//  The error handler may not return any values.
	returnsSomething := func(err error) bool { return true }
	assert.Panics(t, func() { New().OnErr(returnsSomething) })
	//  The error handler can't take args of types that have not yet been
	//  provided.
	takesAString := func(str string, err error) {}
	assert.Panics(t, func() { New().OnErr(takesAString) })
}

func TestBadDefer(t *testing.T) {
	assert.Panics(t, func() { New().Defer(true) },
		"deferred func must actually be a function")

	returnsSomething := func(err error) bool { return true }
	assert.Panics(t, func() { New().Defer(returnsSomething) },
		"deferred func may not return any values")

	takesAString := func(str string) {}
	assert.Panics(t, func() { New().Defer(takesAString) },
		"deferred func arg types must have already been provided")
}

func TestInterfaceConversionOnRun(t *testing.T) {
	chain := New().Arg((*fmt.Stringer)(nil))

	assert.Error(t, chain.Run(), "missing arg")
	assert.NoError(t, chain.Run(nil), "nil value is ok")

	var stringer Stringer
	assert.NoError(t, chain.Run(stringer), "implements stringer")
	assert.NoError(t, chain.Run(&stringer), "implements stringer")

	var ptrStringer PtrStringer
	assert.Error(t, chain.Run(ptrStringer), "does not implement stringer")
	assert.NoError(t, chain.Run(&ptrStringer), "implements stringer")

	var ptrToPtrStringer *PtrStringer
	assert.NoError(t, chain.Run(ptrToPtrStringer), "nil implements stringer")

	var nilStringer fmt.Stringer
	assert.NoError(t, chain.Run(nilStringer), "nil values are ok")

	chain = New().Arg(0)
	assert.Error(t, chain.Run(nil),
		"nil values are not ok for non-pointers and non-interfaces")

	type Struct struct{}
	chain = New().Arg(&Struct{})
	assert.NoError(t, chain.Run(nil), "nil values are ok for pointers to structs")
	assert.NoError(t, chain.Run(&Struct{}), "nil values are ok for pointers to structs")
	assert.Error(t, chain.Run(1), "ints don't match struct pointers")
}

type Stringer struct{}
type PtrStringer struct{}

func (Stringer) String() string     { return "yup" }
func (*PtrStringer) String() string { return "yup" }

func TestBadRunArgs(t *testing.T) {
	chain := New().
		Arg(int(0)).
		Arg((*fmt.Stringer)(nil))

	assert.Error(t, chain.Run(), "not all args are specified")
	assert.Error(t, chain.Run(0), "not all args are specified")
	assert.NoError(t, chain.Run(0, nil), "all args are specified")
}

func TestRunArgsMustExactlyMatchSpecifiedArgs(t *testing.T) {
	chain := New().
		Arg(int(0)).
		Arg("").
		Arg(true)

	// OK
	assert.NoError(t, chain.Run(0, "hi", true))

	// Wrong ordering
	assert.EqualError(t, chain.Run(true, "hi", 0),
		"bad arg: 1st arg of Run(...) should be a int but is bool")
	assert.EqualError(t, chain.Run(0, true, "hi"),
		"bad arg: 2nd arg of Run(...) should be a string but is bool")

	// Too many
	assert.EqualError(t, chain.Run(0, "hi", true, 'x', 0),
		"too many args: expected 3 args but got 5 args")
	// Not enough
	assert.EqualError(t, chain.Run(0, "hi"),
		"missing args of types: [bool]")
	assert.EqualError(t, chain.Run(0),
		"missing args of types: [string bool]")
}

func TestRunWithNilReservedInterface(t *testing.T) {
	var capturedStringer fmt.Stringer = time.Now()
	chain := New().
		Arg((*fmt.Stringer)(nil)).
		Then(func(s fmt.Stringer) { capturedStringer = s })

	require.NoError(t, chain.Run(nil))
	assert.Nil(t, capturedStringer)
}
