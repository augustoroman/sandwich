<img align="right" height=400 src="http://s3.amazonaws.com/foodspotting-ec2/reviews/3957590/thumb_600.jpg?1377120135" />

# Sandwich: Delicious HTTP Middleware [![GoDoc](https://godoc.org/github.com/augustoroman/sandwich?status.png)](http://godoc.org/github.com/augustoroman/sandwich)

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
       http.Handle("/", mw.With(func(w http.ResponseWriter) {
           fmt.Fprintf(w, "Hello world!")
       }))
       if err := http.ListenAndServe(":6060", nil); err != nil {
           log.Fatal(err)
       }
   }
```

For more examples, see the examples directory.


## Usage

### Providing

Sandwich automatically calls your middleware with the necessary arguments to
run them based on the types they require.  These types can be provided by
previous middleware or directly during the initial setup.

For example, you can use this to provide your database to all handlers:

```go
  func main() {
      db_conn := ConnectToDatabase(...)
      mw := sandwich.TheUsual().Provide(db_conn)
      http.Handle("/", mw.With(home))
  }

  func Home(w http.ResponseWriter, r *http.Request, db_conn *Database) {
      // process the request here, using the provided db_conn
  }
```

Provide(...) and ProvideAs(...) are excellent alternatives to using global
values, plus they keep your functions easy to test!


### Handlers

In many cases you want to initialize a value based on the request, for
example extracting the user login:

```go
  func main() {
      mw := sandwich.TheUsual().With(ParseUserCookie)
      http.Handle("/", mw.With(SayHi))
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
  // StartLog creates a *LogEntry and initializes it with basic request
  // information.
  func StartLog(r *http.Request) *LogEntry {
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
    Wrap(StartLog, (*LogEntry).Commit)
```

In this case, `StartLog` returns a `*LogEntry` that is then provided to downstream
handlers, including the deferred Commit handler -- in this case a
[method expression](https://golang.org/ref/spec#Method_expressions) that takes
the `*LogEntry` as its value receiver.


### Providing Interfaces

Unfortunately, providing interfaces is a little tricky.  Since interfaces in
Go are only used for static typing, the encapsulation isn't passed to
functions that accept interface{}, like Provide().

This means that if you have an interface and a concrete implementation, such
as:

```go
  type UserDatabase interface{
      GetUserProfile(u User) (Profile, error)
  }
  type userDbImpl struct { ... }
  func (u *userDbImpl) GetUserProfile(u User) (Profile, error) { ... }
```

You cannot provide this to handlers directly via the Provide() call.

```go
  udb := &userDbImpl{...}
  // DOESN'T WORK: this will provide *userDbImpl, not UserDatabase
  mw.Provide(udb)
  mw.Provide((UserDatabase)(udb)) // DOESN'T WORK EITHER
  udb_iface := UserDatabase(udb)
  mw.Provide(&udb_iface)          // STILL DOESN'T WORK!
```

Instead, you have to either use ProvideAs() or With():

```go
  udb := &userDbImpl{...}
  // either use ProvideAs() with a pointer to the interface
  mw.ProvideAs(udb, (*UserDatabase)(nil))
  // or add a handler that returns the interface
  mw.With(func() UserDatabase { return udb })
```

It's a bit silly, but there you are.


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

At the time each middleware function is added to the stack, the library ensures that it's dependencies have been explicitly provided.  One of the *features* of sandwich is that you can't arbitrary inject values -- they need to have an explicit provisioning source.

**Q: Doesn't the http.Request.Context in go 1.7 solve this?**

Have a request-scoped context allows you to pass values bewteen middleware handlers, it's true.  However, there's no guarantee that the values are available, so you get the same run-time bugs that you might get with a naive dependency-injection framework.  In addition, you have to do type-assertions to get your values, so there's another possible source of bugs.  One of the goals of sandwich is to avoid these two types of bugs.

**Q: I like my hand-coded handlers, they are super fast!**

Guess what?! Because of the structure that sandwich imposes on constructing middleware chains, it can automatically generate a pure Go middleware function (with no reflection or depedency injection) to replace the sandwich calls!  So for those ultra time-sensitive functions, you can replace them with fast auto-generated code and still reap the benefits of using sandwich!

**Q: I don't know, it's still scary and terrible!**

I hope you don't get scared off by all this talk.  Take a look at the library, try it out, and I hope you enjoy it. If you don't, there are lots of great alternatives.
