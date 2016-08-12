package chain

import (
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
