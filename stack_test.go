package sandwich

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func say(s string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) { fmt.Fprintf(w, "%s:", s) }
}

func TestWrapOrder(t *testing.T) {
	mw := New().With(say("a")).Wrap(say("b"), say("e")).Wrap(say("c"), say("d"))

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	mw.ServeHTTP(w, r)

	if w.Body.String() != "a:b:c:d:e:" {
		t.Errorf("Wrong response: %q", w.Body.String())
	}
}
