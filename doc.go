// Package sandwich is a middleware framework for go that lets you write
// testable web servers.
//
// Sandwich allows writing robust middleware handlers that are easily tested:
//   - Avoid globals, instead propagate per-request state automatically from
//     one handler to the next.
//   - Write your handlers to accept the parameters they need rather than
//     type-asserting from an untyped per-request context.
//   - Abort request handling by returning an error.
//
// Sandwich is provides a basic PAT-style router.
//
// # Example
//
// Here's a simple complete program using sandwich:
//
//	package main
//
//	import (
//	    "fmt"
//	    "log"
//	    "net/http"
//
//	    "github.com/augustoroman/sandwich"
//	)
//
//	func main() {
//	    mux := sandwich.TheUsual()
//	    mux.Get("/", func(w http.ResponseWriter) {
//	        fmt.Fprintf(w, "Hello world!")
//	    })
//	    if err := http.ListenAndServe(":6060", mux); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// # Providing
//
// Sandwich automatically calls your middleware with the necessary arguments to
// run them based on the types they require. These types can be provided by
// previous middleware or directly during the initial setup.
//
// For example, you can use this to provide your database to all handlers:
//
//	func main() {
//	    db_conn := ConnectToDatabase(...)
//	    mux := sandwich.TheUsual()
//	    mux.Set(db_conn)
//	    mux.Get("/", Home)
//	}
//
//	func Home(w http.ResponseWriter, r *http.Request, db_conn *Database) {
//	    // process the request here, using the provided db_conn
//	}
//
// Set(...) and SetAs(...) are excellent alternatives to using global values,
// plus they keep your functions easy to test!
//
// # Handlers
//
// In many cases you want to initialize a value based on the request, for
// example extracting the user login:
//
//	func main() {
//	    mux := sandwich.TheUsual()
//	    mux.Get("/", ParseUserCookie, SayHi)
//	}
//	// You can write & test exactly this signature:
//	func ParseUserCookie(r *http.Request) (User, error) { ... }
//	// Then write your handler assuming User is available:
//	func SayHi(w http.ResponseWriter, u User) {
//	    fmt.Fprintf(w, "Hello %s", u.Name)
//	}
//
// This starts to show off the real power of sandwich. For each request, the
// following occurs:
//   - First ParseUserCookie is called. If it returns a non-nil error,
//     sandwich's HandleError is called the request is aborted. If the error
//     is nil, processing continues.
//   - Next SayHi is called with the User value returned from ParseUserCookie.
//
// This allows you to write small, independently testable functions and let
// sandwich chain them together for you. Sandwich works hard to ensure that you
// don't get annoying run-time errors: it's structured such that it must always
// be possible to call your functions when the middleware is initialized rather
// than when the http handler is being executed, so you don't get surprised
// while your server is running.
//
// # Error Handlers
//
// When a handler returns an error, sandwich aborts the middleware chain and
// looks for the most recently registered error handler and calls that. Error
// handlers may accept any types that have been provided so far in the
// middleware stack as well as the error type. They must not have any return
// values.
//
// # Wrapping Handlers
//
// Sandwich also allows registering handlers to run during AND after the
// middleware (and error handling) stack has completed. This is especially
// useful for handles such as logging or gzip wrappers. Once the before handle
// is run, the 'after' handlers are queued to run and will be run regardless of
// whether an error aborts any subsequent middleware handlers.
//
// Typically this is done with the first function creating and initializing some
// state to pass to the deferred handler. For example, the logging handlers are:
//
//	// StartLog creates a *LogEntry and initializes it with basic request
//	// information.
//	func NewLogEntry(r *http.Request) *LogEntry {
//	  return &LogEntry{Start: time.Now(), ...}
//	}
//
//	// Commit fills in the remaining *LogEntry fields and writes the entry out.
//	func (entry *LogEntry) Commit(w *ResponseWriter) {
//	  entry.Elapsed = time.Since(entry.Start)
//	  ...
//	  WriteLog(*entry)
//	}
//
// and are added to the chain using:
//
//	var LogRequests = Wrap{NewLogEntry, (*LogEntry).Commit}
//
// In this case, the `Wrap` executes NewLogEntry during middleware processing
// that returns a *LogEntry which is provided to downstream handlers, including
// the deferred Commit handler -- in this case a method expression
// (https://golang.org/ref/spec#Method_expressions) that takes the *LogEntry as
// its value receiver.
//
// # Providing Interfaces
//
// Unfortunately, providing interfaces is a little tricky. Since interfaces in
// Go are only used for static typing, the encapsulation isn't passed to
// functions that accept interface{}, like Set().
//
// This means that if you have an interface and a concrete implementation, such
// as:
//
//	type UserDatabase interface{
//	    GetUserProfile(u User) (Profile, error)
//	}
//	type userDbImpl struct { ... }
//	func (u *userDbImpl) GetUserProfile(u User) (Profile, error) { ... }
//
// You cannot provide this to handlers directly via the Set() call.
//
//	udb := &userDbImpl{...}
//	// DOESN'T WORK: this will provide *userDbImpl, not UserDatabase
//	mux.Set(udb)
//	// STILL DOESN'T WORK
//	mux.Set((UserDatabase)(udb))
//	// *STILL* DOESN'T WORK
//	udb_iface := UserDatabase(udb)
//	mux.Set(&udb_iface)
//
// Instead, you have to either use SetAs() or a dedicated middleware function:
//
//	udb := &userDbImpl{...}
//	mux.SetAs(udb, (*UserDatabase)(nil))        // either use SetAs() with a pointer to the interface
//	mux.Use(func() UserDatabase { return udb }) // or add a handler that returns the interface
//
// It's a bit silly, but that's how it is.
package sandwich
