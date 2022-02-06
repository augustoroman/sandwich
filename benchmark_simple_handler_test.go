package sandwich

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Sample data to JSON-encode for benchmarking.
var userInfo = struct {
	Id        uint64 `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}{
	12345467, "John Doe", "john.doe@example.com", "https://www.example.com/users/john.doe/image",
}

func write204(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
func hello(w http.ResponseWriter, r *http.Request)    { w.Write([]byte("Hello there!")) }
func sendjson(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(userInfo) }

func makeWrite204_TheUsual() Middleware { return TheUsual().Then(NoLog, write204) }
func makeWrite204_Bare() Middleware     { return New().Then(write204) }
func makeHello_TheUsual() Middleware    { return TheUsual().Then(NoLog, hello) }
func makeHello_Bare() Middleware        { return New().Then(hello) }
func makeSendJson_TheUsual() Middleware { return TheUsual().Then(NoLog, sendjson) }
func makeSendJson_Bare() Middleware     { return New().Then(sendjson) }

func bench(N int, h http.Handler) {
	req := httptest.NewRequest("GET", "/", nil)
	for i := 0; i < N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

// Just to shorten the following benchmark functions:
type Handler = http.HandlerFunc

func Benchmark_Hello_RawHTTP(b *testing.B)            { bench(b.N, Handler(hello)) }
func Benchmark_Hello_Generated_Bare(b *testing.B)     { bench(b.N, Handler(gen_hello_bare())) }
func Benchmark_Hello_Generated_TheUsual(b *testing.B) { bench(b.N, Handler(gen_hello_theusual())) }
func Benchmark_Hello_Dynamic_Bare(b *testing.B)       { bench(b.N, makeHello_Bare()) }
func Benchmark_Hello_Dynamic_TheUsual(b *testing.B)   { bench(b.N, makeHello_TheUsual()) }

func Benchmark_Write204_RawHTTP(b *testing.B)        { bench(b.N, Handler(write204)) }
func Benchmark_Write204_Generated_Bare(b *testing.B) { bench(b.N, Handler(gen_write204_bare())) }
func Benchmark_Write204_Generated_TheUsual(b *testing.B) {
	bench(b.N, Handler(gen_write204_theusual()))
}
func Benchmark_Write204_Dynamic_Bare(b *testing.B)     { bench(b.N, makeWrite204_Bare()) }
func Benchmark_Write204_Dynamic_TheUsual(b *testing.B) { bench(b.N, makeWrite204_TheUsual()) }

func Benchmark_SendJson_RawHTTP(b *testing.B)        { bench(b.N, Handler(sendjson)) }
func Benchmark_SendJson_Generated_Bare(b *testing.B) { bench(b.N, Handler(gen_sendjson_bare())) }
func Benchmark_SendJson_Generated_TheUsual(b *testing.B) {
	bench(b.N, Handler(gen_sendjson_theusual()))
}
func Benchmark_SendJson_Dynamic_Bare(b *testing.B)     { bench(b.N, makeSendJson_Bare()) }
func Benchmark_SendJson_Dynamic_TheUsual(b *testing.B) { bench(b.N, makeSendJson_TheUsual()) }

func TestGenBenchmarkCode(t *testing.T) {
	gen_functions := []string{
		makeHello_Bare().Code("sandwich", "gen_hello_bare"),
		makeHello_TheUsual().Code("sandwich", "gen_hello_theusual"),
		makeWrite204_Bare().Code("sandwich", "gen_write204_bare"),
		makeWrite204_TheUsual().Code("sandwich", "gen_write204_theusual"),
		makeSendJson_Bare().Code("sandwich", "gen_sendjson_bare"),
		makeSendJson_TheUsual().Code("sandwich", "gen_sendjson_theusual"),
	}
	gen_code := "\n\n" + strings.Join(gen_functions, "\n\n") + "\n\n"
	t.Log("Generated code: " + gen_code)
}

// ---------------------------------------------------------------------------
// Auto-generated code below here
// ---------------------------------------------------------------------------

func gen_hello_bare() func(
	rw http.ResponseWriter,
	req *http.Request,
) {
	return func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		hello(rw, req)

	}
}

func gen_hello_theusual() func(
	rw http.ResponseWriter,
	req *http.Request,
) {
	return func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		var pResponseWriter *ResponseWriter
		rw, pResponseWriter = WrapResponseWriter(rw)

		var pLogEntry *LogEntry
		pLogEntry = StartLog(req)

		defer func() {
			(*LogEntry).Commit(pLogEntry, pResponseWriter)
		}()

		NoLog(pLogEntry)

		hello(rw, req)

	}
}

func gen_write204_bare() func(
	rw http.ResponseWriter,
	req *http.Request,
) {
	return func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		write204(rw, req)

	}
}

func gen_write204_theusual() func(
	rw http.ResponseWriter,
	req *http.Request,
) {
	return func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		var pResponseWriter *ResponseWriter
		rw, pResponseWriter = WrapResponseWriter(rw)

		var pLogEntry *LogEntry
		pLogEntry = StartLog(req)

		defer func() {
			(*LogEntry).Commit(pLogEntry, pResponseWriter)
		}()

		NoLog(pLogEntry)

		write204(rw, req)

	}
}

func gen_sendjson_bare() func(
	rw http.ResponseWriter,
	req *http.Request,
) {
	return func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		sendjson(rw, req)

	}
}

func gen_sendjson_theusual() func(
	rw http.ResponseWriter,
	req *http.Request,
) {
	return func(
		rw http.ResponseWriter,
		req *http.Request,
	) {
		var pResponseWriter *ResponseWriter
		rw, pResponseWriter = WrapResponseWriter(rw)

		var pLogEntry *LogEntry
		pLogEntry = StartLog(req)

		defer func() {
			(*LogEntry).Commit(pLogEntry, pResponseWriter)
		}()

		NoLog(pLogEntry)

		sendjson(rw, req)

	}
}
