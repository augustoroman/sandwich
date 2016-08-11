package sandwich

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"

	"testing"
)

func TestMiddlware(t *testing.T) {
	t.Skip()

	recRw := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/foo/bar", nil)
	assert.NoError(t, err)

	type Path string
	New().Then(
		func(r *http.Request) Path { return Path(r.URL.Path) },
		func(w http.ResponseWriter, r *http.Request, p Path) {
			fmt.Fprintf(w, "%s %s", r.Method, p)
		},
	).ServeHTTP(recRw, r)

	assert.Equal(t, "GET /foo/bar", recRw.Body.String())
}

func TestMiddlewareDefer(t *testing.T) {
	recRw := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/foo/bar", nil)
	assert.NoError(t, err)

	New().Provide("xyz").Then(
		func(w http.ResponseWriter) string {
			fmt.Fprintf(w, "before:")
			return "abc"
		}).Defer(
		func(w http.ResponseWriter) {
			fmt.Fprintf(w, ":after")
		}).Then(
		func(w http.ResponseWriter, arg string) {
			fmt.Fprintf(w, "%s", arg)
		},
	).ServeHTTP(recRw, r)
	assert.NoError(t, err)

	assert.Equal(t, "before:abc:after", recRw.Body.String())
}
