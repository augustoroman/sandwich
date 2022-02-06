package httprouter_sandwich_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustoroman/sandwich/httprouter_sandwich"
	"github.com/julienschmidt/httprouter"
)

func TestParamsRouting(t *testing.T) {
	// An example function using the httprouter.Params as an input.
	greet := func(w http.ResponseWriter, p httprouter.Params) {
		fmt.Fprintf(w, "%s %s", p[0].Value, p[1].Value)
	}

	// An example server using the martini_sandwich adapter.
	s := httprouter_sandwich.TheUsual()
	r := httprouter.New()
	r.GET("/say/:greeting/:name", s.Then(greet).H)

	// Call the server.
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/say/Hi/Bob", nil)
	r.ServeHTTP(rw, req)

	// Validate the output.
	if rw.Body.String() != "Hi Bob" {
		t.Errorf("Wrong response: %q", rw.Body.String())
	}
}
