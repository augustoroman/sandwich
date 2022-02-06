package sandwich

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestMiddlewareWrap(t *testing.T) {
	recRw := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/foo/bar", nil)
	assert.NoError(t, err)

	before := func(w http.ResponseWriter) string {
		fmt.Fprintf(w, "before:")
		return "abc"
	}
	after := func(w http.ResponseWriter) {
		fmt.Fprintf(w, ":after")
	}
	during := func(w http.ResponseWriter, arg string) {
		fmt.Fprintf(w, "%s", arg)
	}

	New().Set("xyz").Wrap(before, after).Then(during).ServeHTTP(recRw, r)
	assert.NoError(t, err)

	assert.Equal(t, "before:abc:after", recRw.Body.String())
}
