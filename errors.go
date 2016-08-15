package sandwich

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is an error implementation that provides the ability to specify three
// things to the sandwich error handler:
//   * The HTTP status code that should be used in the response.
//   * The client-facing message that should be sent.  Typically this is a
//     sanitized error message, such as "Internal Server Error".
//   * Internal debugging detail including a log message and the underlying
//     error that should be included in the server logs.
// Note that Cause may be nil.
type Error struct {
	Code      int
	ClientMsg string
	LogMsg    string
	Cause     error
}

func (e Error) Error() string {
	msg := fmt.Sprintf("(%d) %s", e.Code, e.LogMsg)
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// LogIfMsg will set the Error field on the LogEntry if the Error's LogMsg
// field has something.
func (e Error) LogIfMsg(l *LogEntry) {
	if e.LogMsg != "" {
		l.Error = e
	}
}

// Done is a sentinel error value that can be used to interrupt the middleware
// chain without triggering the default error handling.  HandleError will not
// attempt to write any status code or client message, nor will it add the error
// to the log.
var Done = errors.New("<done>")

// ToError will convert a generic non-nil error to an explicit sandwich.Error
// type.  If err is already a sandwich.Error, it will be returned.  Otherwise, a
// generic 500 Error (internal server error) will be initialized and returned.
// Note that if err is nil, it will still return a generic 500 Error.
func ToError(err error) Error {
	e, ok := err.(Error)
	if !ok {
		e = Error{LogMsg: "Failure", Cause: err}
	}
	if e.Code == 0 {
		e.Code = 500
	}
	if e.ClientMsg == "" {
		e.ClientMsg = http.StatusText(e.Code)
	}
	return e
}

// HandleError is the default error handler included in sandwich.TheUsual.
// If the error is a sandwich.Error, it responds with the specified status code
// and client message.  Otherwise, it responds with a 500.  In both cases, the
// underlying error is added to the request log.
//
// If the error is sandwich.Done, HandleError does nothing.
func HandleError(w http.ResponseWriter, r *http.Request, l *LogEntry, err error) {
	if err == Done {
		return
	}
	e := ToError(err)
	e.LogIfMsg(l)
	http.Error(w, e.ClientMsg, e.Code)
}

// HandleErrorJson is identical to HandleError except that it responds to the
// client as JSON instead of plain text.  Again, detailed error info is added
// to the request log.
//
// If the error is sandwich.Done, HandleErrorJson does nothing.
func HandleErrorJson(w http.ResponseWriter, r *http.Request, l *LogEntry, err error) {
	if err == Done {
		return
	}
	e := ToError(err)
	e.LogIfMsg(l)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Code)
	fmt.Fprintf(w, `{"error":%q}`, e.ClientMsg)
}
