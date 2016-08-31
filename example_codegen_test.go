package sandwich_test

import (
	"fmt"

	"github.com/augustoroman/sandwich"

	"net/http"
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
	// 	sandwich_test_UserDb_val UserDb,
	// ) func(
	// 	http_ResponseWriter_val http.ResponseWriter,
	// 	http_Request_ptr_val *http.Request,
	// ) {
	// 	return func(
	// 		http_ResponseWriter_val http.ResponseWriter,
	// 		http_Request_ptr_val *http.Request,
	// 	) {
	// 		var sandwich_ResponseWriter_ptr_val *sandwich.ResponseWriter
	// 		http_ResponseWriter_val, sandwich_ResponseWriter_ptr_val = sandwich.WrapResponseWriter(http_ResponseWriter_val)
	//
	// 		var sandwich_LogEntry_ptr_val *sandwich.LogEntry
	// 		sandwich_LogEntry_ptr_val = sandwich.StartLog(http_Request_ptr_val)
	//
	// 		defer func() {
	// 			(*sandwich.LogEntry).Commit(sandwich_LogEntry_ptr_val, sandwich_ResponseWriter_ptr_val)
	// 		}()
	//
	// 		var sandwich_test_UserId_val UserId
	// 		var err error
	// 		sandwich_test_UserId_val, err = sandwich_test.GetUserIdFromRequest(http_Request_ptr_val)
	// 		if err != nil {
	// 			sandwich.HandleError(http_ResponseWriter_val, http_Request_ptr_val, sandwich_LogEntry_ptr_val, err)
	// 			return
	// 		}
	//
	// 		var sandwich_test_User_ptr_val *User
	// 		sandwich_test_User_ptr_val, err = sandwich_test.LoadUser(sandwich_test_UserId_val, sandwich_test_UserDb_val)
	// 		if err != nil {
	// 			sandwich.HandleError(http_ResponseWriter_val, http_Request_ptr_val, sandwich_LogEntry_ptr_val, err)
	// 			return
	// 		}
	//
	// 		sandwich_test.WelcomePage(http_ResponseWriter_val, sandwich_test_User_ptr_val)
	//
	// 	}
	// }
}
