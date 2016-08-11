package main

import (
	"github.com/augustoroman/sandwich"
	"net/http"
)

func Generated_ShowLandingPage(
	main_UserDb_val UserDb,
) func(
	http_ResponseWriter_val http.ResponseWriter,
	http_Request_ptr_val *http.Request,
) {
	return func(
		http_ResponseWriter_val http.ResponseWriter,
		http_Request_ptr_val *http.Request,
	) {
		var sandwich_ResponseWriter_ptr_val *sandwich.ResponseWriter
		http_ResponseWriter_val, sandwich_ResponseWriter_ptr_val = sandwich.WrapResponseWriter(http_ResponseWriter_val)

		var sandwich_LogEntry_ptr_val *sandwich.LogEntry
		sandwich_LogEntry_ptr_val = sandwich.StartLog(http_Request_ptr_val)

		defer func() {
			(*sandwich.LogEntry).Commit(sandwich_LogEntry_ptr_val, sandwich_ResponseWriter_ptr_val, sandwich_LogEntry_ptr_val)
		}()

		var main_User_ptr_val *User
		var err error
		main_User_ptr_val, err = ParseUserIfLoggedIn(http_Request_ptr_val, main_UserDb_val, sandwich_LogEntry_ptr_val)
		if err != nil {
			sandwich.HandleError(http_ResponseWriter_val, http_Request_ptr_val, sandwich_LogEntry_ptr_val, err)
			return
		}

		ShowLandingPage(http_ResponseWriter_val, main_User_ptr_val)

	}
}

func Generated_ShowUserProfile(
	main_UserDb_val UserDb,
) func(
	http_ResponseWriter_val http.ResponseWriter,
	http_Request_ptr_val *http.Request,
) {
	return func(
		http_ResponseWriter_val http.ResponseWriter,
		http_Request_ptr_val *http.Request,
	) {
		var sandwich_ResponseWriter_ptr_val *sandwich.ResponseWriter
		http_ResponseWriter_val, sandwich_ResponseWriter_ptr_val = sandwich.WrapResponseWriter(http_ResponseWriter_val)

		var sandwich_LogEntry_ptr_val *sandwich.LogEntry
		sandwich_LogEntry_ptr_val = sandwich.StartLog(http_Request_ptr_val)

		defer func() {
			(*sandwich.LogEntry).Commit(sandwich_LogEntry_ptr_val, sandwich_ResponseWriter_ptr_val, sandwich_LogEntry_ptr_val)
		}()

		var main_User_ptr_val *User
		var err error
		main_User_ptr_val, err = ParseUserIfLoggedIn(http_Request_ptr_val, main_UserDb_val, sandwich_LogEntry_ptr_val)
		if err != nil {
			sandwich.HandleError(http_ResponseWriter_val, http_Request_ptr_val, sandwich_LogEntry_ptr_val, err)
			return
		}

		var main_User_val User
		main_User_val, err = FailIfNotAuthenticated(main_User_ptr_val)
		if err != nil {
			sandwich.HandleError(http_ResponseWriter_val, http_Request_ptr_val, sandwich_LogEntry_ptr_val, err)
			return
		}

		ShowUserProfile(http_ResponseWriter_val, main_User_val)

	}
}
