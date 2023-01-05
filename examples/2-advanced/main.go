// 2-advanced is a demo webserver for the sandwich middleware package
// demonstrating advanced usage.
//
// This provides a sample, multi-user TODO list application that allows users
// to sign in via a Google account or sign in using fake credentials.
// This example demonstrates more advanced features of sandwich, including:
//
//   - Providing interface types to the middleware chain.
//     TaskDb is the interface provided to the handlers, the actual value injected
//     in main() is a taskDbImpl.
//   - Using 3rd party middleware (go.auth, go.rice)
//   - Using a 3rd party router (gorilla/mux)
//   - Using multiple error handlers, and custom error handlers.
//     Most web servers will want to server a custom HTML error page for user-facing
//     error pages.  An example of that is included here.  For AJAX calls, however,
//     ....
//   - Early exit of the middleware chain via the sandwich.Done error
//   - Auto-generating handler code
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/augustoroman/sandwich"
	auth "github.com/bradrydzewski/go.auth"
)

//go:embed static
var static embed.FS

func main() {
	// Read in configuration:
	var config struct {
		Host         string `json:"host"`
		Port         int    `json:"port"`
		CookieSecret string `json:"cookie-secret"`
		ClientId     string `json:"oauth2-client-id"`
		ClientSecret string `json:"oauth2-client-secret"`
	}
	failOnError(readJsonFile("config.json", &config))

	// Setup Oauth login framework:
	auth.Config.LoginRedirect = "/auth/login" // send user here to login
	auth.Config.LoginSuccessRedirect = "/"    // send user here post-login
	auth.Config.CookieSecure = false          // for local-testing only
	auth.Config.CookieSecret = []byte(config.CookieSecret)
	// This must match the authorized URLs entered in the google cloud api console.
	redirectUrl := fmt.Sprintf("http://%s/auth/google/callback", config.Host)
	authHandler := auth.Google(config.ClientId, config.ClientSecret, redirectUrl)

	// Setup task database:
	taskDb := taskDbImpl{}

	// Load our templates.
	tpl := template.Must(template.ParseFS(static, "static/*.tpl.html"))

	// Start setting up our server:
	mux := sandwich.TheUsual()
	mux.Use(ParseUserCookie, LogUser)
	mux.SetAs(taskDb, (*TaskDb)(nil))
	mux.Set(tpl)
	mux.OnErr(CustomErrorPage)

	// Don't log these requests since we don't have a favicon, it's just a
	// bunch of 404 spam.
	mux.Get("/favicon.ico", sandwich.NoLog, NotFound)

	// When login is called, we'll FIRST call our very own CheckForFakeLogin
	// handler.  If we detect the fake login form params, we'll process that
	// and then abort the middleware chain.
	// However, if we don't have fake parameters, we'll continue on and let
	// the authHandler take care of things.
	mux.Any("/auth/login", CheckForFakeLogin, authHandler)
	mux.Any("/auth/google/callback", authHandler)
	// Note that we can use auth.DeleteUserCookie directly.
	mux.Any("/auth/logout", auth.DeleteUserCookie,
		http.RedirectHandler("/", http.StatusTemporaryRedirect))

	// Static file handling.  The s.Then(...) wrapper isn't strictly necessary,
	// but it gives us logging (and potentially gzip or other middleware).
	// static := http.StripPrefix("/static", http.FileServer(http.FS(
	// 	mustSubFS(static, "static"))))
	mux.Get("/static/:path*", sandwich.ServeFS(static, "static", "path"))

	// OK, here are the core handlers:
	mux.Get("/", Home)
	// All API calls will use the api middleware that responds with JSON for
	// errors and requires users to be logged in.
	api := mux.SubRouter("/api/")
	api.OnErr(sandwich.HandleErrorJson)
	api.Use(RequireLoggedIn)
	api.Post("/task", TaskFromAddRequest, TaskDb.Add, SendTaskAsJson)
	api.Post("/task/:id", TaskOpFromUpdateRequest, UpdateTask)

	// Catch all remaining URLs and respond with not-found errors.  We
	// explicitly use the error-return mechanism so that we get the JSON
	// response under /api/ and normal HTML responses elsewhere.
	api.Any("/:*", NotFound)
	mux.Any("/:*", NotFound)

	// Otherwise, start serving!
	addr := fmt.Sprintf("localhost:%d", config.Port)
	log.Printf("Server listening on http://%s", addr)
	failOnError(http.ListenAndServe(addr, mux))
}

// ============================================================================
// Database

type UserId string
type TaskDb interface {
	List(UserId) ([]Task, error)
	Add(UserId, *Task) error
	Update(UserId, Task) error
}
type Task struct {
	Id   string `json:"id"`
	Desc string `json:"desc"`
	Done bool   `json:"done"`
}

type taskDbImpl map[UserId][]Task

func (db taskDbImpl) List(u UserId) ([]Task, error) { return db[u], nil }
func (db taskDbImpl) Add(u UserId, t *Task) error {
	t.Id = fmt.Sprint(time.Now().UnixNano())
	db[u] = append(db[u], *t)
	return nil
}
func (db taskDbImpl) Update(u UserId, t Task) error {
	tasks := db[u]
	for i, task := range tasks {
		if task.Id == t.Id {
			tasks[i] = t
			return nil
		}
	}
	return sandwich.Error{
		Code:      http.StatusBadRequest,
		ClientMsg: "No such task",
		Cause:     fmt.Errorf("No such task: %q", t.Id),
	}
}

// ============================================================================
// Core Handlers

func Home(
	w http.ResponseWriter,
	r *http.Request,
	uid UserId,
	u auth.User,
	db TaskDb,
	tpl *template.Template,
) error {
	if u == nil {
		// tpl := template.Must(template.New("").Parse(
		// 	rice.MustFindBox("static").MustString("landing-page.tpl.html")))
		return tpl.ExecuteTemplate(w, "landing-page.tpl.html", nil)
	}
	tasks, err := db.List(uid)
	if err != nil {
		return err
	}
	// tpl := template.Must(template.New("").Parse(
	// 	rice.MustFindBox("static").MustString("home.tpl.html")))
	return tpl.ExecuteTemplate(w, "home.tpl.html", map[string]interface{}{
		"User":  u,
		"Tasks": tasks,
	})
}

func TaskFromAddRequest(r *http.Request) (*Task, error) {
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		return nil, sandwich.Error{Code: http.StatusBadRequest, Cause: err}
	}
	if t.Desc == "" {
		return nil, sandwich.Error{Code: http.StatusBadRequest,
			ClientMsg: "Please include a task description"}
	}
	return &t, nil
}

func SendTaskAsJson(w http.ResponseWriter, t *Task) error {
	return json.NewEncoder(w).Encode(map[string]interface{}{"task": t})
}

type TaskOp struct {
	Toggle bool
	Id     string
}

func TaskOpFromUpdateRequest(r *http.Request) (TaskOp, error) {
	var op TaskOp
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		return op, sandwich.Error{Code: http.StatusBadRequest, Cause: err}
	}
	if op.Id == "" {
		return op, sandwich.Error{
			Code:      http.StatusBadRequest,
			ClientMsg: "Invalid op: missing task id",
		}
	}
	return op, nil
}

func UpdateTask(w http.ResponseWriter, r *http.Request, uid UserId, op TaskOp, db TaskDb) error {
	tasks, err := db.List(uid)
	if err != nil {
		return err
	}
	var t Task
	for i := range tasks {
		if tasks[i].Id == op.Id {
			t = tasks[i]
			break
		}
	}
	if t.Id == "" {
		return sandwich.Error{
			Code:      http.StatusBadRequest,
			ClientMsg: "No such task id: " + op.Id,
		}
	}

	if op.Toggle {
		t.Done = !t.Done
	}

	if err := db.Update(uid, t); err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(map[string]interface{}{"task": t})
}

// This will get called for any error that occurs outside of the API calls.
func CustomErrorPage(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	tpl *template.Template,
	l *sandwich.LogEntry,
) {
	// Make sure we actually have a real error:
	if err == sandwich.Done {
		return
	}
	// Convert the error to a sandwich.Error that has an error code.
	e := sandwich.ToError(err)
	// Always log the error and error details.
	l.Error = e

	w.WriteHeader(e.Code)
	err = tpl.ExecuteTemplate(w, "error.tpl.html", map[string]interface{}{
		"Error": e,
	})

	// But... what if our fancy template rendering fails?  At this point, we
	// fall back to the simplest possible thing: http.Error(...).  Maybe it'll
	// work, but we'll also log the error so it doesn't disappear.
	if err != nil {
		// Try putting a typo in the template name above, and you'll see this:
		l.Error = fmt.Errorf("Failed to render error page: %v\nTriggering error: %v",
			err, e)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func CheckForFakeLogin(w http.ResponseWriter, r *http.Request) error {
	if r.FormValue("id") == "" {
		return nil
	}

	user := &auth.GoogleUser{
		UserId:    r.FormValue("id"),
		UserEmail: r.FormValue("email"),
		UserName:  r.FormValue("name"),
	}
	auth.SetUserCookie(w, r, user)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

	// Great, everything is handled, so don't continue with the Google auth.
	return sandwich.Done
}

// ============================================================================
// Basic Handlers

// You could also use .Then(http.NotFound), but that wouldn't go through the
// error-handlers.  The advantage of using the error handlers is that you
// automatically get JSON vs HTML handling.
func NotFound() error { return sandwich.Error{Code: http.StatusNotFound} }

func RequireLoggedIn(u auth.User) error {
	if u == nil {
		return sandwich.Error{Code: http.StatusUnauthorized}
	}
	return nil
}
func ParseUserCookie(r *http.Request) (auth.User, UserId) {
	// Ignore errors.  If the cookie is invalid or expired or corrupt or
	// missing, just consider the user not-logged-in.
	u, _ := auth.GetUserCookie(r)
	var uid UserId
	if u != nil {
		uid = UserId(u.Id())
	}
	return u, uid
}

// Adds the current user to the per-request log notes, if logged in.
func LogUser(u auth.User, e *sandwich.LogEntry) {
	if u != nil {
		e.Note["user"] = u.Email()
		e.Note["userId"] = u.Id()
	} else {
		e.Note["user"] = "<none>"
	}
}

// ============================================================================
// Simple utilities

func readJsonFile(filename string, dst interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(dst)
}

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func mustSubFS(base fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(base, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
