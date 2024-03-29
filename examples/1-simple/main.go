// 1-simple is a demo webserver for the sandwich middleware package
// demonstrating basic usage.
//
// This example demonstrates most of the basic features of sandwich, including:
//
// * Providing user types to the middleware chain
// * Adding middleware handlers to the stack.
// * Writing handlers that provide request-scoped values.
// * Writing handlers using injected values.
// * Using the default sandwich logging system.
// * Using the default sandwich error system.
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/augustoroman/sandwich"
)

// Interface for abstracting out the user database.
type UserDb interface {
	Lookup(id string) (User, error)
}
type User struct{ Id, Name, Email string }

func main() {
	// To reduce log spam, we'll just put this here, not using any framework.
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Setup connections to the databases.
	udb := userDb{{"bob", "Bob", "bob@example.com"}, {"alice", "Alice", "alice@example.com"}}

	// Create a typical sandwich middleware with logging and error-handling.
	mux := sandwich.TheUsual()
	// Inject config and user database; now available to all handlers.
	mux.SetAs(udb, (*UserDb)(nil))
	// In this example, we'll always check to see if the user is logged in.
	// If so, we'll add the user ID to the log entries.
	mux.Use(ParseUserIfLoggedIn)

	// If the user is logged in, they'll get a personalized landing page.
	// Otherwise, they'll get a generic landing page.
	mux.Get("/", ShowLandingPage)
	mux.Post("/login", Login)

	// Some pages are only allowed if the user is logged in.
	mux.Get("/user/profile", FailIfNotAuthenticated, ShowUserProfile)
	// If you have multiple pages that require authentication, you could do:
	//   authed := mw.Then(FailIfNotAuthenticated)
	//   http.Handle("/user/profile", authed.Then(ShowUserProfile))
	//   http.Handle("/user/...", authed.Then(...))
	//   http.Handle("/user/...", authed.Then(...))

	log.Println("Serving on http://localhost:8080/")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal("Can't start webserver:", err)
	}
}

// The actual user DB implementation.
type userDb []User

func (udb userDb) Lookup(id string) (User, error) {
	for _, u := range udb {
		if id == u.Id {
			return u, nil
		}
	}
	return User{}, fmt.Errorf("no such user %q", id)
}

func ShowLandingPage(w http.ResponseWriter, u *User) {
	fmt.Fprintln(w, "<html><body style='font-family:sans-serif'>")
	if u == nil {
		fmt.Fprint(w, "Hello unknown person!")
		fmt.Fprintf(w, "  [<a href='/user/profile'>profile</a> will fail and log an error]")
	} else {
		fmt.Fprintf(w, "Welcome back, %s!", u.Name)
		fmt.Fprintf(w, "  [<a href='/user/profile'>profile</a>]")
	}
	fmt.Fprintln(w, `<p><hr>
        Login<br>
        <form action='/login' method='POST'>
            <input type='text' name='id'><br>
            <input type='submit'>
        </form>
        <hr>
        Try logging in with: <ul>
        	<li> "alice" will authenticate to Alice
        	<li> "bob" will authenticate to Bob but panic during request handling
        	<li> any other string for a non-authenticated user.
        </ul>
    `)
}

func ShowUserProfile(w http.ResponseWriter, u User) {
	fmt.Fprintln(w, "<html><body>")
	fmt.Fprintf(w, "Id: %s <br/>Name: %s <br/>Email:%s", u.Id, u.Name, u.Email)

	// Show an example of user-code panicking in a handler.
	if u.Id == "bob" {
		panic("oops")
	}
}

func Login(w http.ResponseWriter, r *http.Request, udb UserDb, e *sandwich.LogEntry) {
	u, err := udb.Lookup(r.FormValue("id"))
	if err != nil {
		log.Printf("No such user id: %q", r.FormValue("id"))
		http.SetCookie(w, &http.Cookie{Name: "auth", Value: "", Expires: time.Now()})
		// Redirect to /
		fmt.Fprintf(w, `<html><head>
			<meta http-equiv="refresh" content="0;URL='/'"/>`)
		return
	}

	e.Note["userId"] = u.Id
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    u.Id, // Encrypt cookie here, maybe include the whole user struct.
		Expires:  time.Now().Add(time.Hour),
		MaxAge:   int(time.Hour / time.Second),
		HttpOnly: true,
	})
	// Redirect to /user/profile
	fmt.Fprintf(w, `<html><head>
			<meta http-equiv="refresh" content="0;URL='/user/profile'"/>`)
}

func FailIfNotAuthenticated(u *User) (User, error) {
	if u == nil {
		return User{}, sandwich.Error{
			Code:      http.StatusUnauthorized,
			ClientMsg: "Not logged in",
			LogMsg:    "Unauthorized access attempt",
		}
	}
	return *u, nil
}

func getAndParseCookie(r *http.Request) (string, error) {
	c, err := r.Cookie("auth")
	if err != nil {
		return "", err
	}
	userid := c.Value // Decrypt cookie here, maybe getting a whole user struct.
	return userid, nil
}

func ParseUserIfLoggedIn(r *http.Request, udb UserDb, e *sandwich.LogEntry) (*User, error) {
	if user_id, err := getAndParseCookie(r); err != nil {
		return nil, nil // not logged in or expired or corrupt.  Ignore cookie.
	} else if user, err := udb.Lookup(user_id); err != nil {
		log.Printf("No such user: %q", user_id)
		return nil, nil // no such user
	} else {
		e.Note["userId"] = user.Id
		return &user, nil // Hello logged-in user!
	}
}
