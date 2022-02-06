package sandwich

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzip(t *testing.T) {
	greet := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hi there!")
	}
	handler := Gzip(New()).Then(greet)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add(headerAcceptEncoding, "gzip")
	handler.ServeHTTP(resp, req)

	if resp.Header().Get(headerContentEncoding) != "gzip" {
		t.Errorf("Not gzip'd? Content-encoding: %q", resp.Header())
	}

	if resp.Header().Get(headerContentLength) != "" {
		t.Errorf("Not supposed to include content-length: %q", resp.Header())
	}

	r, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if body, err := ioutil.ReadAll(r); err != nil {
		t.Fatal(err)
	} else if string(body) != "Hi there!" {
		t.Errorf("Wrong response: %q", string(body))
	}

	// Also, test without the accept header and make sure it's NOT gzip'd.
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/", nil)
	handler.ServeHTTP(resp, req)
	if resp.Header().Get(headerContentEncoding) == "gzip" {
		t.Errorf("Unexpectedly gzip'd: Content-encoding: %q", resp.Header())
	}
	if resp.Body.String() != "Hi there!" {
		t.Errorf("Wrong response: %q", resp.Body.String())
	}
}
