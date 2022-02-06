package martini_sandwich_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustoroman/sandwich/martini_sandwich"
	"github.com/go-martini/martini"
)

func TestMartiniParamsAvailability(t *testing.T) {
	// An example function using the martini.Params as an input.
	greet := func(w http.ResponseWriter, p martini.Params) {
		fmt.Fprintf(w, "%s %s", p["greeting"], p["name"])
	}

	// An example server using the martini_sandwich adapter.
	s := martini_sandwich.TheUsual()
	m := martini.Classic()
	m.Get("/say/:greeting/:name", s.Then(greet).H) // <-- notice the .H

	// Call the server.
	rw := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/say/Hi/Bob", nil)
	m.ServeHTTP(rw, r)

	// Validate the output.
	if rw.Body.String() != "Hi Bob" {
		t.Errorf("Wrong response: %q", rw.Body.String())
	}
}
