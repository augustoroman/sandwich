<img align="right" height=400 src="http://s3.amazonaws.com/foodspotting-ec2/reviews/3957590/thumb_600.jpg?1377120135" />

# Sandwich: Delicious HTTP Middleware  [![Build Status](https://travis-ci.org/augustoroman/sandwich.svg?branch=master)](https://travis-ci.org/augustoroman/sandwich)  [![Coverage](https://gocover.io/_badge/github.com/augustoroman/sandwich?1)](https://gocover.io/github.com/augustoroman/sandwich)  [![Go Report Card](https://goreportcard.com/badge/github.com/augustoroman/sandwich)](https://goreportcard.com/report/github.com/augustoroman/sandwich)  [![GoDoc](https://pkg.go.dev/badge/github.com/augustoroman/sandwich)](https://pkg.go.dev/github.com/augustoroman/sandwich)

*Keep pilin' it on!*

Sandwich is a middleware & routing framework that lets you write your handlers
and middleware the way you want to and it takes care of tracking & validating
dependencies.

## Features

* Keeps middleware and handlers simple and testable.
* Consolidates error handling.
* Ensures that middleware dependencies are safely provided -- avoids unsafe
  casting from generic context objects.
* Detects missing dependencies *during route construction* (before the server
  starts listening!), not when the route is actually called.
* Provides clear and helpful error messages.
* Compatible with the [http.Handler](https://pkg.go.dev/net/http#Handler)
  interface and lots of existing middleware.
* Provides just a touch of magic: enough to make things easier, but not enough
  to induce a debugging nightmare.

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
	// Create a default sandwich middlware stack that includes logging and
	// a simple error handler.
	mux := sandwich.TheUsual()
	mux.Get("/", func(w http.ResponseWriter) {
		fmt.Fprintf(w, "Hello world!")
	})
	if err := http.ListenAndServe(":6060", mux); err != nil {
		log.Fatal(err)
	}
}
```

See the [examples directory](examples/) for:
* A [hello-world sample](examples/0-helloworld)
* A [basic usage](examples/1-simple) demo
* A [TODO app](examples/2-advanced) showing advanced usage including custom
  error handling, embedded files, code generation, & login and authentication
  via oauth.

## Usage

### Providing

Sandwich automatically calls your middleware with the necessary arguments to
run them based on the types they require.  These types can be provided by
previous middleware or directly during the initial setup.

For example, you can use this to provide your database to all handlers:

```go
func main() {
    db_conn := ConnectToDatabase(...)
    mux := sandwich.TheUsual()
    mux.Set(db_conn)
    mux.Get("/", Home)
}

func Home(w http.ResponseWriter, r *http.Request, db_conn *Database) {
    // process the request here, using the provided db_conn
}
```

Set(...) and SetAs(...) are excellent alternatives to using global
values, plus they keep your functions easy to test!


### Handlers

In many cases you want to initialize a value based on the request, for
example extracting the user login:

```go
func main() {
    mux := sandwich.TheUsual()
    mux.Get("/", ParseUserCookie, SayHi)
}
// You can write & test exactly this signature:
func ParseUserCookie(r *http.Request) (User, error) { ... }
// Then write your handler assuming User is available:
func SayHi(w http.ResponseWriter, u User) {
    fmt.Fprintf(w, "Hello %s", u.Name)
}
```

This starts to show off the real power of sandwich.  For each request, the
following occurs:

  * First `ParseUserCookie` is called.  If it returns a non-nil error,
    sandwich's `HandleError` is called and the request is aborted.  If the error
    is nil, processing continues.
  * Next `SayHi` is called with `User` returned from `ParseUserCookie`.

This allows you to write small, independently testable functions and let
sandwich chain them together for you.  Sandwich works hard to ensure that
you don't get annoying run-time errors: it's structured such that it must
always be possible to call your functions when the middleware is initialized
rather than when the http handler is being executed, so you don't get
surprised while your server is running.


### Error Handlers

When a handler returns an error, sandwich aborts the middleware chain and
looks for the most recently registered error handler and calls that.
Error handlers may accept any types that have been provided so far in the
middleware stack as well as the error type.  They must not have any return
values.

Here's an example of rendering errors with a custom error page:

```go
type ErrorPageTemplate *template.Template
func main() {
    tpl := template.Must(template.ParseFiles("path/to/my/error_page.tpl"))
    mux := sandwich.TheUsual()
    mux.Set(ErrorPageTemplate(tpl))
    mux.OnErr(MyErrorHandler)
    ...
}
func MyErrorHandler(w http.ResponseWriter, t ErrorPageTemplate, l *sandwich.LogEntry, err error) {
    if err == sandwich.Done {  // sandwich.Done can be returned to abort middleware.
        return                 // It indicates there was no actual error, so just return.
    }
    // Unwrap to a sandwich.Error that has Code, ClientMsg, and internal LogMsg.
    e := sandwich.ToError(err)
    // If there's an internal log message, add it to the request log.
    e.LogIfMsg(l)
    // Respond with my custom html error page, including the client-facing msg.
    w.WriteHeader(e.Code)
    t.Execute(w, map[string]string{Msg: e.ClientMsg})
}
```

Error handlers allow you consolidate the error handling of your web app.  You
can customize the error page, assign user-facing error codes, detect and fire
alerts for certain errors, and control which errors get logged -- all in one
place.

By default, sandwich never sends internal error details to the client and
insteads logs the details.

### Wrapping Handlers

Sandwich also allows registering handlers to run during AND after the middleware
(and error handling) stack has completed.  This is especially useful for handles
such as logging or gzip wrappers.  Once the before handle is run, the 'after'
handlers are queued to run and will be run regardless of whether an error aborts
any subsequent middleware handlers.

Typically this is done with the first function creating and initializing some
state to pass to the deferred handler.  For example, the logging handlers
are:

```go
// NewLogEntry creates a *LogEntry and initializes it with basic request
// information.
func NewLogEntry(r *http.Request) *LogEntry {
    return &LogEntry{Start: time.Now(), ...}
}

// Commit fills in the remaining *LogEntry fields and writes the entry out.
func (entry *LogEntry) Commit(w *ResponseWriter) {
    entry.Elapsed = time.Since(entry.Start)
    ...
    WriteLog(*entry)
}
```

and are added to the chain using:

```go
var LogRequests = Wrap{NewLogEntry, (*LogEntry).Commit}
```

In this case, `NewLogEntry` returns a `*LogEntry` that is then provided to
downstream handlers, including the deferred Commit handler -- in this case a
[method expression](https://golang.org/ref/spec#Method_expressions) that takes
the `*LogEntry` as its value receiver.


### Providing Interfaces

Unfortunately, set interface values is a little tricky.  Since interfaces in Go
are only used for static typing, the encapsulation isn't passed to functions
that accept interface{}, like Set().

This means that if you have an interface and a concrete implementation, such
as:

```go
type UserDatabase interface{
    GetUserProfile(u User) (Profile, error)
}
type userDbImpl struct { ... }
func (u *userDbImpl) GetUserProfile(u User) (Profile, error) { ... }
```

You cannot provide this to handlers directly via the Set() call.

```go
udb := &userDbImpl{...}
// DOESN'T WORK: this will provide *userDbImpl, not UserDatabase
mux.Set(udb)
mux.Set((UserDatabase)(udb)) // DOESN'T WORK EITHER
udb_iface := UserDatabase(udb)
mux.Set(&udb_iface)          // STILL DOESN'T WORK!
```

Instead, you have to either use SetAs() or a dedicated middleware function that
returns the interface:

```go
udb := &userDbImpl{...}
// either use SetAs() with a pointer to the interface
mux.SetAs(udb, (*UserDatabase)(nil))
// or add a handler that returns the interface
mux.Use(func() UserDatabase { return udb })
```

It's a bit silly, but there you are.


## FAQ

Sandwich uses reflection-based dependency-injection to call the middleware
functions with the parameters they need.

**Q: OMG reflection and dependency-injection, isn't that terrible and slow and
non-idiomatic go?!**

Whoa, nelly.  Let's deal with those one at time, m'kay?

**Q: Isn't reflection slow?**

Not compared to everything else a webserver needs to do.

Yes, sandwich's reflection-based dependency-injection code is slower than
middleware code that directly calls functions, however **the vast majority of
server code (especially during development) is not impacted by time spent
calling a few functions, but rather by HTTP network I/O, request parsing,
database I/O, response marshalling, etc.**

**Q: Ok, but aren't both reflection and dependency-injection non-idiomatic Go?**

Sorta.  The use of reflection in and of itself isn't non-idiomatic, but the use
of magical dependency injection is:  Go eschews magic.

However, one of the major goals of this library is to allow the HTTP handler
code (and all middleware) to be really clean, idiomatic go functions that are
testable by themselves.  The idea is that the magic is small, contained, doesn't
leak, and provides substantial benefit.

**Q: But wait, don't you get annoying run-time "dependency-not-found" errors
with dependency-injection?**

While it's true that you can't get the same compile-time checking that you do
with direct-call-based middleware, sandwich works really hard to ensure that you
don't get surprises while running your server.

At the time each middleware function is added to the stack, the library ensures
that it's dependencies have been explicitly provided.  One of the *features* of
sandwich is that you can't arbitrary inject values -- they need to have an
explicit provisioning source.

**Q: Doesn't the http.Request.Context in go 1.7 solve the middleware dependency
problem?**

Have a request-scoped context allows you to pass values between middleware
handlers, it's true.  However, there's no guarantee that the values are
available, so you get the same run-time bugs that you might get with a naive
dependency-injection framework.  In addition, you have to do type-assertions to
get your values, so there's another possible source of bugs.  One of the goals
of sandwich is to avoid these two types of bugs.

**Q: Why do I have to use _two_ functions (before & after) to wrap a request.
Why can't I just have one with a next() function?**

Many middleware frameworks provide the capability to wrap a request via a next()
function.  Sometimes it's part of a context object
([martini's Context.Next()](https://pkg.go.dev/github.com/go-martini/martini#Context),
[gin's Context.Next()](https://pkg.go.dev/github.com/gin-gonic/gin#Context.Next))
and sometimes it's directly provided
([negroni's third handler arg](https://pkg.go.dev/github.com/urfave/negroni#HandlerFunc)).

While implementing sandwich, I initially included a `next()` function until I
realized it was impossible to validate the dependencies with such a function.
Sandwich guarantees that dependencies can be supplied, and therefore `next()`
had to go.

Instead, I took a tip from go and instead implemented
[defer](https://pkg.go.dev/github.com/augustoroman/sandwich/chain#Func.Defer).
The wrap interface simply makes it obvious that there's a before and after.
This allows me to keep my dependency guarantee.

**Q: I don't know, it's still scary and terrible!**

Don't get scared off. Take a look at the library, try it out, and I hope you
enjoy it. If you don't, there are lots of great alternatives.
