<img align="right" height=400 src="https://upload.wikimedia.org/wikipedia/commons/2/29/Dagwood_sandwich.jpg" />

# Sandwich HTTP Middleware [![GoDoc](https://godoc.org/github.com/augustoroman/sandwich?status.png)](http://godoc.org/github.com/augustoroman/sandwich)

*Keep pilin' it on!*

## Features

* Keeps middleware simple and testable.
* Ensures that middleware dependencies are safely provided -- avoids unsafe casting from generic context objects.
* Detects missing dependencies during route initialization.
* Provides clear and helpful error messages.
* Provides just a touch of magic: enough to make things easier, but not enough to induce a debugging nightmare.

## Getting started

Here's a very simple example of using sandwich with the standard HTTP stack:

```go
   package main

   import (
       "fmt"
       "log"
       "net/http"
       "github.com/augustoroman/sandwich"
   )

   func main() {
       mw := sandwich.TheUsual()
       http.Handle("/", mw.Then(func(w http.ResponseWriter) {
           fmt.Fprintf(w, "Hello world!")
       }))
       if err := http.ListenAndServe(":6060", nil); err != nil {
           log.Fatal(err)
       }
   }
```

For more examples, see the examples directory.

## Overview

Sandwich is a middleware library designed with the following goals in mind:

* Make it really easy to write *testable* HTTP server code.
* Make it easy to write correct and *debuggable* HTTP server code.
* Make it easy to do *robust and consistent error handling* that differentiates between logging internal error details, choosing an appropriate status code, and sending an appropriate external error message.
* Make it *possible to convert that to extremely fast HTTP server code* once the easy and correct code is done.


## FAQ

Sandwich uses reflection-based dependency-injection to call the middleware functions with the parameters they need.

**Q: OMG reflection and dependency-injection, isn't that terrible and slow and non-idiomatic go?!**

Whoa, nelly.  Let's deal with those one at time, m'kay?

**Q: Isn't reflection slow?**

Not compared to everything else a webserver needs to do.

Yes, sandwich's reflection-based dependency-injection code is slower than middleware code that directly calls functions, however **the vast majority of server code (especially during development) is not impacted by time spent calling a few functions, but rather by HTTP network I/O, request parsing, database I/O, response marshalling, etc.**

The real time difference in calls is ~1000ns/call for reflection vs only 2ns/call for direct calls.  1000ns is noise for most HTTP handlers which typically last 2-5ms (so that's a 0.05% slowdown -- less than 1%).

**Q: Ok, but aren't both reflection and dependency-injection non-idiomatic Go?**

Sorta.  The use of reflection in and of itself isn't non-idiomatic, but the use of magical dependency injection is:  Go eschews magic.

However, one of the major goals of this library is to allow the HTTP handler code (and all middleware) to be really clean, idiomatic go functions that are testable by themselves.  The idea is that the magic is small, contained, doesn't leak, and provides substantial benefit.

**Q: But wait, don't you get annoying run-time "dependency-not-found" errors with dependency-injection?**

While it's true that you can't get the same compile-time checking that you do with direct-call-based middleware, sandwich works really hard to ensure that you don't get surprises while running your server.

At the time each middleware function is added to the stack, the library ensure that it's dependencies have been explicitly provided.  One of the *features* of sandwich is that you can't arbitrary inject values -- they need to have an explicit provisioning source.

**Q: Doesn't the http.Request.Context in go 1.7 solve this?**

Have a request-scoped context allows you to pass values bewteen middleware handlers, it's true.  However, there's no guarantee that the values are available, so you get the same run-time bugs that you might get with a naive dependency-injection framework.  In addition, you have to do type-assertions to get your values, so there's another possible source of bugs.  One of the goals of sandwich is to avoid these two types of bugs.

**Q: I like my hand-coded handlers, they are super fast!**

Guess what?! Because of the structure that sandwich imposes on constructing middleware chains, it can automatically generate a pure Go middleware function (with no reflection or depedency injection) to replace the sandwich calls!  So for those ultra time-sensitive functions, you can replace them with fast auto-generated code and still reap the benefits of using sandwich!

**Q: I don't know, it's still scary and terrible!**

I hope you don't get scared off by all this talk.  Take a look and the library, try it out, and I hope you enjoy it. If you don't, there are lots of great alternatives.
