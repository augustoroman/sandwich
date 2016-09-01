package sandwich_test

import (
	"fmt"
	"net/http"

	"github.com/augustoroman/sandwich"
)

type UserId string
type User struct {
	Id   UserId
	Name string
}
type UserDb map[UserId]User

func GetUserIdFromRequest(r *http.Request) (UserId, error) {
	// Normally you'd decrypt & validate a secure cookie here.
	return UserId(r.FormValue("user")), nil
}

func LoadUser(id UserId, udb UserDb) (*User, error) {
	// Maybe the user would be part of the cookie, maybe you need more info
	// from the DB
	if id == "" {
		return nil, nil // no such user
	} else if user, ok := udb[id]; !ok {
		return nil, sandwich.Error{
			Code:  http.StatusUnauthorized,
			Cause: fmt.Errorf("Invalid id: %q", id),
		}
	} else {
		return &user, nil
	}
}

func WelcomePage(w http.ResponseWriter, u *User) {
	fmt.Fprintln(w, "<html><body>")
	if u == nil {
		fmt.Fprintln(w, "Welcome stranger!")
	} else {
		fmt.Fprintln(w, "Howdy ", u.Name+"!")
	}
	fmt.Fprintln(w, "</body></html>")
}

func ExampleMiddleware_Code() {
	udb := UserDb{}
	mw := sandwich.TheUsual().Provide(udb).
		With(GetUserIdFromRequest, LoadUser, WelcomePage)

	fmt.Print(mw.Code("sandwich_test", "WelcomePage"))

	// Output:
	// func WelcomePage(
	// 	userDb UserDb,
	// ) func(
	// 	rw http.ResponseWriter,
	// 	req *http.Request,
	// ) {
	// 	return func(
	// 		rw http.ResponseWriter,
	// 		req *http.Request,
	// 	) {
	// 		var pResponseWriter *sandwich.ResponseWriter
	// 		rw, pResponseWriter = sandwich.WrapResponseWriter(rw)
	//
	// 		var pLogEntry *sandwich.LogEntry
	// 		pLogEntry = sandwich.StartLog(req)
	//
	// 		defer func() {
	// 			(*sandwich.LogEntry).Commit(pLogEntry, pResponseWriter)
	// 		}()
	//
	// 		var userId UserId
	// 		var err error
	// 		userId, err = sandwich_test.GetUserIdFromRequest(req)
	// 		if err != nil {
	// 			sandwich.HandleError(rw, req, pLogEntry, err)
	// 			return
	// 		}
	//
	// 		var pUser *User
	// 		pUser, err = sandwich_test.LoadUser(userId, userDb)
	// 		if err != nil {
	// 			sandwich.HandleError(rw, req, pLogEntry, err)
	// 			return
	// 		}
	//
	// 		sandwich_test.WelcomePage(rw, pUser)
	//
	// 	}
	// }
}
