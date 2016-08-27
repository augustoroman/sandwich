package httprouter_sandwich

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
)

func TestParamsRouting(t *testing.T) {
	r := httprouter.New()
	printParam := func(w http.ResponseWriter, key string, p httprouter.Params) {
		for _, param := range p {
			if param.Key == key {
				fmt.Fprintf(w, "%s = %s", key, param.Value)
				return
			}
		}
		http.Error(w, "Param "+key+" not found", http.StatusNotFound)
	}
	mw := New()
	r.GET("/foo/:arg", mw.Provide("arg").With(printParam).ServeHTTP)
	r.GET("/bar/:baz", mw.Provide("baz").With(printParam).ServeHTTP)

	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/foo/asdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(rw, req)

	if rw.Code != 200 {
		t.Errorf("Bad response code: %v", rw.Code)
	}
	if rw.Body.String() != "arg = asdf" {
		t.Errorf("Wrong arg value: %q (expected 'asdf')", rw.Body.String())
	}
}
