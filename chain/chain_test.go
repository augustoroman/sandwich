package chain

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func New() Chain { return Chain{} }

func TestInitialInjection(t *testing.T) {
	var args []interface{}
	recordArgs := func(a int, b string) { args = append(args, a, b) }

	err := New().
		Reserve(0).
		Reserve("").
		With(recordArgs).
		Provide(3).
		Provide("four").
		With(recordArgs).
		Run(1, "two")
	assert.NoError(t, err)

	assert.EqualValues(t, []interface{}{1, "two", 3, "four"}, args)
}

func TestInitialDeferredInjection(t *testing.T) {
	var args []interface{}
	recordArgs := func(a int, b string) { args = append(args, a, b) }

	err := New().Reserve(0).Reserve("").With(recordArgs).Run(2, "xyz")
	assert.NoError(t, err)

	assert.EqualValues(t, []interface{}{2, "xyz"}, args)
}

func TestDeferredExecutionOrder(t *testing.T) {
	var buf bytes.Buffer
	say := func(s string) func() { return func() { buf.WriteString(s + ":") } }
	err := New().
		With(say("a"), say("b")).
		Defer(say("f")).
		Defer(say("e")).
		With(say("c")).
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
		With(say("a"), say("b")).
		Defer(say("f")).
		OnErr(onErr).
		Defer(say("e")).
		With(fail).
		With(say("c")).
		Defer(say("d")).
		Run()
	assert.NoError(t, err)
	assert.Equal(t, "a:b:err[failed]:e:f:", buf.String())
}

func TestBasicChainExecution(t *testing.T) {
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

	err := New().With(provide_initial, verify_injected, modify_injected).Run()
	assert.NoError(t, err)

	assert.Equal(t, 6, a)
	assert.Equal(t, "bye", b)
}

func TestSimpleErrorHandling(t *testing.T) {
	var out string
	chain := New().Reserve("").Reserve(0).
		OnErr(func(err error) { out += "First error handler: " + err.Error() }).
		With(
			func(val string) (string, error) {
				if val == "foo" {
					return "bar", nil
				} else {
					return "", fmt.Errorf("%q is not foo", val)
				}
			}).
		OnErr(func(err error) { out += "Second error handler: " + err.Error() }).
		With(
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
	assert.Panics(t, func() { New().With(func(string) {}) })
	assert.NotPanics(t, func() { New().Provide("").With(func(string) {}) })

	assert.Panics(t, func() { New().With(func(int, string) {}) })
	assert.NotPanics(t, func() { New().Provide("").Provide(3).With(func(int, string) {}) })

	assert.NotPanics(t, func() {
		New().With(
			func() string { return "" },
			func(string) int { return 3 },
			func(int) {},
		)
	}, "Should be OK: Everything is provided by earlier functions.")
	assert.NotPanics(t, func() {
		New().With(
			func() string { return "" },
			func(string) int { return 3 },
		).OnErr(func(int, error) {})
	}, "Should be OK: Everything is provided by earlier functions")

	assert.Panics(t, func() {
		New().With(
			func() string { return "" },
			func(string) int { return 3 },
			func(bool) {},
			func(int) {},
		)
	}, "Should FAIL: bool isn't provided anywhere")
	assert.Panics(t, func() {
		New().With(func() string { return "" }, func(string) int { return 3 }).
			OnErr(func(bool, error) {}).
			With(func(int) {})
	}, "Should FAIL: bool isn't provided anywhere (even error handlers need proper provisioning)")
}

func TestErrorAbortsHandling(t *testing.T) {
	var out string
	New().OnErr(func(err error) { out += "Failed @ " + err.Error() }).With(
		func() error { out += "1 "; return nil },
		func() error { out += "2 "; return fmt.Errorf("2") },
		func() error { out += "3 "; return nil },
	).Run()
	assert.Equal(t, "1 2 Failed @ 2", out)
}

func a() string                { return "hello " }
func b(s string) (string, int) { return s + "world", 42 }
func c(s string, n int)        {}

func TestCatchesPanics(t *testing.T) {
	var err error
	captureError := func(e error) { err = e }
	panics := func() { panic("ahhhh! 🔥") }

	New().OnErr(captureError).With(a, b, c).Defer(c).With(panics).Run()

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

	New().
		OnErr(onerr).
		With(a, b, c).
		Defer(deferred).
		With(fails).
		Run()

	assert.Equal(t, "onerr[💣]:defer[💣]:", buf.String())

	// But what if nothing actually fails?  Defer's can still accept errors.
	buf.Reset()
	New().
		OnErr(onerr).
		With(a, b, c).
		Defer(deferred).
		// With(fails).  // no failure!
		Run()

	assert.Equal(t, "defer[<nil>]:", buf.String())
}

func TestDefaultErrorHandler(t *testing.T) {
	var buf bytes.Buffer
	onerr := func(err error) { fmt.Fprintf(&buf, "onerr[%v]:", err) }
	fails := func() error { return errors.New("☠") }

	// Restore the default error handler when we're done with the test.
	defer func(orig interface{}) { DefaultErrorHandler = orig }(DefaultErrorHandler)
	DefaultErrorHandler = onerr

	New().With(fails).Run()

	assert.Equal(t, "onerr[☠]:", buf.String())
}

func TestProvideAsNil(t *testing.T) {
	check := func(s fmt.Stringer) {
		if s != nil {
			t.Error("s should be nil!")
		}
	}
	err := New().ProvideAs(nil, (*fmt.Stringer)(nil)).With(check).Run()
	if err != nil {
		t.Fatal(err)
	}
}
