package sandwich

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

// WrapResponseWriter creates a ResponseWriter and returns it as both an
// http.ResponseWriter and a *ResponseWriter.  The double return is redundant
// for native Go code, but is a necessary hint to the dependency injection.
func WrapResponseWriter(w http.ResponseWriter) (http.ResponseWriter, *ResponseWriter) {
	rw := &ResponseWriter{w, 0, 0}
	return rw, rw
}

// ResponseWriter wraps http.ResponseWriter to add tracking of the response size
// and response code.
type ResponseWriter struct {
	http.ResponseWriter
	Size int // The size of the response written so far, in bytes.
	Code int // The status code of the response, or 0 if not written yet.
}

func (w *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the ResponseWriter doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}

func (w *ResponseWriter) Flush() {
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (w *ResponseWriter) WriteHeader(code int) {
	if w.Code == 0 {
		w.Code = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWriter) Write(p []byte) (int, error) {
	if w.Code == 0 {
		w.Code = 200
	}
	n, err := w.ResponseWriter.Write(p)
	w.Size += n
	return n, err
}
