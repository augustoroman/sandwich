package sandwich

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
func hello(w http.ResponseWriter, r *http.Request)    { _, _ = w.Write([]byte("Hello there!")) }
func sendjson(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(userInfo) }

func addTestRoutes(mux Router) {
	mux.Get("/204", write204)
	mux.Get("/hello", hello)
	mux.Get("/jsonuser", sendjson)

	mux.Get("/long/1/2/3/4/5/6/7/8/9/xyz/204", write204)
	mux.Get("/long/1/2/3/4/5/6/7/8/9/xyz/hello", hello)
	mux.Get("/long/1/2/3/4/5/6/7/8/9/xyz/jsonuser", sendjson)

	mux.Get("/1param/:var/204", write204)
	mux.Get("/1param/:var/hello", hello)
	mux.Get("/1param/:var/jsonuser", sendjson)

	mux.Get("/manyparams/:var/:x/:y/:z/:a/:b/:c/204", write204)
	mux.Get("/manyparams/:var/:x/:y/:z/:a/:b/:c/hello", hello)
	mux.Get("/manyparams/:var/:x/:y/:z/:a/:b/:c/jsonuser", sendjson)

	mux.Get("/greedy/:var*/204", write204)
	mux.Get("/greedy/:var*/hello", hello)
	mux.Get("/greedy/:var*/jsonuser", sendjson)
}

var usualRouter = func() Router {
	mux := TheUsual()
	mux.Use(NoLog)
	addTestRoutes(mux)
	return mux
}()
var bareRouter = func() Router {
	mux := BuildYourOwn()
	addTestRoutes(mux)
	return mux
}()

func bench(N int, route string, mux Router) {
	req := httptest.NewRequest("GET", route, nil)
	for i := 0; i < N; i++ {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}
}

func S(s ...string) []string { return s }

func runBenches(b *testing.B, mux Router, routes, endpoints []string) {
	for _, ep := range endpoints {
		for _, route := range routes {
			path := route + "/" + ep
			b.Run(ep+"::"+path, func(b *testing.B) {
				bench(b.N, path, mux)
			})
		}
	}
}

// Just to shorten the following benchmark functions:
type Handler = http.HandlerFunc

func BenchmarkUsual(b *testing.B) {
	runBenches(b, usualRouter,
		S("", "/long/1/2/3/4/5/6/7/8/9/xyz", "/1param/foo", "/manyparams/foo/x/y/z/a/b/c", "/greedy/short", "/greedy/x/y/z/a/b/c"),
		S("204", "hello", "jsonuser"),
	)
}
func BenchmarkBare(b *testing.B) {
	runBenches(b, bareRouter,
		S("", "/long/1/2/3/4/5/6/7/8/9/xyz", "/1param/foo", "/manyparams/foo/x/y/z/a/b/c", "/greedy/short", "/greedy/x/y/z/a/b/c"),
		S("204", "hello", "jsonuser"),
	)
}

func BenchmarkCalls(b *testing.B) {
	for i := 1; i < 20; i += 2 {
		b.Run(fmt.Sprintf("%02d", i), func(b *testing.B) {
			var calls []any
			for j := 0; j < i; j++ {
				calls = append(calls, hello)
			}
			mux := BuildYourOwn()
			mux.Get("/", calls...)
			b.ResetTimer()
			bench(b.N, "/", mux)
		})
	}
}

// func Benchmark_Usual_Short_Write204(b *testing.B) { bench(b.N, "/204", usualRouter) }
// func Benchmark_Usual_Short_Hello(b *testing.B)    { bench(b.N, "/hello", usualRouter) }
// func Benchmark_Usual_Short_SnedJson(b *testing.B) { bench(b.N, "/204", usualRouter) }

// // func Benchmark_Hello_RawHTTP(b *testing.B)          { bench(b.N, Handler(hello)) }
// // func Benchmark_Hello_Dynamic_Bare(b *testing.B)     { bench(b.N, makeHello_Bare()) }
// // func Benchmark_Hello_Dynamic_TheUsual(b *testing.B) { bench(b.N, makeHello_TheUsual()) }

// // func Benchmark_Write204_RawHTTP(b *testing.B) { bench(b.N, Handler(write204)) }

// // func Benchmark_Write204_Dynamic_Bare(b *testing.B)     { bench(b.N, makeWrite204_Bare()) }
// // func Benchmark_Write204_Dynamic_TheUsual(b *testing.B) { bench(b.N, makeWrite204_TheUsual()) }

// // func Benchmark_SendJson_RawHTTP(b *testing.B)          { bench(b.N, Handler(sendjson)) }
// // func Benchmark_SendJson_Dynamic_Bare(b *testing.B)     { bench(b.N, makeSendJson_Bare()) }
// // func Benchmark_SendJson_Dynamic_TheUsual(b *testing.B) { bench(b.N, makeSendJson_TheUsual()) }
