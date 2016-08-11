package chain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func New2() Chain { return Chain{} }

func Test2InitialInjection(t *testing.T) {
	var args []interface{}
	recordArgs := func(a int, b string) { args = append(args, a, b) }

	err := New2().Provide(1).Provide("two").Then(recordArgs).Provide(3).Provide("four").Then(recordArgs).Run()
	assert.NoError(t, err)

	assert.EqualValues(t, []interface{}{1, "two", 3, "four"}, args)
}

func Test2InitialDeferredInjection(t *testing.T) {
	var args []interface{}
	recordArgs := func(a int, b string) { args = append(args, a, b) }

	err := New2().Reserve(0).Reserve("").Then(recordArgs).Run(2, "xyz")
	assert.NoError(t, err)

	assert.EqualValues(t, []interface{}{2, "xyz"}, args)
}

func Test2Compilation(t *testing.T) {
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

	err := New2().Then(provide_initial, verify_injected, modify_injected).Run()
	assert.NoError(t, err)

	assert.Equal(t, 6, a)
	assert.Equal(t, "bye", b)
}

func Test2SimpleErrorHandling(t *testing.T) {
	var out string
	chain := New2().Reserve("").Reserve(0).
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

func Test2MustProvideTypes(t *testing.T) {
	assert.Panics(t, func() { New2().Then(func(string) {}) })
	assert.NotPanics(t, func() { New2().Provide("").Then(func(string) {}) })

	assert.Panics(t, func() { New2().Then(func(int, string) {}) })
	assert.NotPanics(t, func() { New2().Provide("").Provide(3).Then(func(int, string) {}) })

	assert.NotPanics(t, func() {
		New2().Then(
			func() string { return "" },
			func(string) int { return 3 },
			func(int) {},
		)
	}, "Should be OK: Everything is provided by earlier functions.")
	assert.NotPanics(t, func() {
		New2().Then(
			func() string { return "" },
			func(string) int { return 3 },
		).OnErr(func(int, error) {})
	}, "Should be OK: Everything is provided by earlier functions")

	assert.Panics(t, func() {
		New2().Then(
			func() string { return "" },
			func(string) int { return 3 },
			func(bool) {},
			func(int) {},
		)
	}, "Should FAIL: bool isn't provided anywhere")
	assert.Panics(t, func() {
		New2().Then(func() string { return "" }, func(string) int { return 3 }).
			OnErr(func(bool, error) {}).
			Then(func(int) {})
	}, "Should FAIL: bool isn't provided anywhere (even error handlers need proper provisioning)")
}

func Test2ErrorAbortsHandling(t *testing.T) {
	var out string
	New2().OnErr(func(err error) { out += "Failed @ " + err.Error() }).Then(
		func() error { out += "1 "; return nil },
		func() error { out += "2 "; return fmt.Errorf("2") },
		func() error { out += "3 "; return nil },
	).Run()
	assert.Equal(t, "1 2 Failed @ 2", out)
}
