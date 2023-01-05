package sandwich_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/augustoroman/sandwich"
)

type UserID string

type User struct{}
type UserDB interface {
	Get(UserID) (*User, error)
	New(*User) (UserID, error)
	Del(UserID) error
	List() ([]*User, error)
}

func ExampleRouter() {
	var db UserDB // = NewUserDB

	root := sandwich.TheUsual()
	root.SetAs(db, &db)

	api := root.SubRouter("/api")
	api.OnErr(sandwich.HandleErrorJson)

	apiUsers := api.SubRouter("/users")
	apiUsers.Get("/:uid", UserIDFromParam, UserDB.Get, SendUser)
	apiUsers.Delete("/:uid", UserIDFromParam, UserDB.Del)
	apiUsers.Get("/", UserDB.List, SendUserList)
	apiUsers.Post("/", UserFromCreateRequest, UserDB.New, SendUserID)

	var staticFS fs.FS
	root.Get("/home/", GetLoggedInUser, UserDB.Get, Home)
	root.Get("/:path", http.FileServer(http.FS(staticFS)))

	// Output:
}

func GetLoggedInUser(r *http.Request) (UserID, error) {
	token := r.Header.Get("user-token")
	uid := UserID(token) // decode the token to get the user info
	if uid == "" {
		return "", sandwich.Error{Code: http.StatusUnauthorized, ClientMsg: "invalid user token"}
	}
	return uid, nil
}

func Home(w http.ResponseWriter, u *User) {
	fmt.Fprintf(w, "Hello %v", u)
}

func UserIDFromParam(p sandwich.Params) (UserID, error) {
	uid := UserID(p["id"])
	if uid == "" {
		return "", sandwich.Error{Code: 400, ClientMsg: "Missing UID param", LogMsg: "Request missing UID param"}
	}
	return "", nil
}

func UserFromCreateRequest(r *http.Request) (*User, error) {
	u := &User{}
	return u, json.NewDecoder(r.Body).Decode(u)
}

func SendUserList(w http.ResponseWriter, users []*User) error { return SendJson(w, users) }
func SendUser(w http.ResponseWriter, user *User) error        { return SendJson(w, user) }
func SendUserID(w http.ResponseWriter, id UserID) error       { return SendJson(w, id) }

func SendJson(w http.ResponseWriter, val interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(val)
}
