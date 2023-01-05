package sandwich

import (
	"compress/gzip"
	"net/http"
	"strings"
)

const (
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerContentLength   = "Content-Length"
	headerContentType     = "Content-Type"
	headerVary            = "Vary"
)

// Gzip wraps a sandwich.Middleware to add gzip compression to the output for
// all subsequent handlers.
//
// For example, to gzip everything you could use:
//
//	router.Use(sandwich.Gzip)
//	...use as normal...
//
// Or, to gzip just a particular route you could do:
//
//	router.Get("/foo/bar", sandwich.Gzip, MyHandleFooBar)
//
// Note that this does NOT auto-detect the content and disable compression for
// already-compressed data (e.g. jpg images).
var Gzip = Wrap{provideGZipWriter, (*gZipWriter).Flush}

func provideGZipWriter(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *gZipWriter) {
	if !strings.Contains(r.Header.Get(headerAcceptEncoding), "gzip") {
		return w, nil
	}
	headers := w.Header()
	headers.Set(headerContentEncoding, "gzip")
	headers.Set(headerVary, headerAcceptEncoding)

	wr := &gZipWriter{w, gzip.NewWriter(w)}
	return wr, wr
}

type gZipWriter struct {
	http.ResponseWriter
	w *gzip.Writer
}

func (g *gZipWriter) Write(p []byte) (int, error) {
	if len(g.Header().Get(headerContentType)) == 0 {
		g.Header().Set(headerContentType, http.DetectContentType(p))
	}
	return g.w.Write(p)
}

func (g *gZipWriter) Flush() {
	g.Header().Del(headerContentLength)
	g.w.Close()
}
