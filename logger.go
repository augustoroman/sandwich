package sandwich

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// Injected for testing
var time_Now = time.Now
var os_Stderr io.Writer = os.Stderr

// LogEntry is the information tracked on a per-request basis for the sandwich
// Logger.  All fields other than Note are automatically filled in.  The Note
// field is a generic key-value string map for adding additional per-request
// metadata to the logs.  You can take *sandwich.LogEntry to your functions to
// add fields to Note.
//
// For example:
//
//	func MyAuthCheck(r *http.Request, e *sandwich.LogEntry) (User, error) {
//	    user, err := decodeAuthCookie(r)
//	    if user != nil {
//	        e.Note["user"] = user.Id()  // indicate which user is auth'd
//	    }
//	    return user, err
//	}
type LogEntry struct {
	RemoteIp     string
	Start        time.Time
	Request      *http.Request
	StatusCode   int
	ResponseSize int
	Elapsed      time.Duration
	Error        error
	Note         map[string]string
	// set to true to suppress logging this request
	Quiet bool
}

// NoLog is a middleware function that suppresses log output for this request.
// For example:
//
//	// suppress logging of the favicon request to reduce log spam.
//	router.Get("/favicon.ico", sandwich.NoLog, staticHandler)
//
// This depends on WriteLog respecting the Quiet flag, which the default
// implementation does.
func NoLog(e *LogEntry) { e.Quiet = true }

// LogRequests is a middleware wrap that creates a log entry during middleware
// processing and then commits the log entry after the middleware has executed.
var LogRequests = Wrap{NewLogEntry, (*LogEntry).Commit}

// NewLogEntry creates a *LogEntry and initializes it with basic request
// information.
func NewLogEntry(r *http.Request) *LogEntry {
	return &LogEntry{
		RemoteIp: remoteIp(r),
		Start:    time_Now(),
		Request:  r,
		Note:     map[string]string{},
	}
}

// Commit fills in the remaining *LogEntry fields and writes the entry out.
func (entry *LogEntry) Commit(w *ResponseWriter) {
	entry.Elapsed = time_Now().Sub(entry.Start)
	entry.ResponseSize = w.Size
	entry.StatusCode = w.Code
	WriteLog(*entry)
}

// Some nice escape codes
const (
	_GREEN  = "\033[32m"
	_YELLOW = "\033[33m"
	_RESET  = "\033[0m"
	_RED    = "\033[91m"
)

// WriteLog is called to actually write a LogEntry out to the log. By default,
// it writes to stderr and colors normal requests green, slow requests yellow,
// and errors red.  You can replace the function to adjust the formatting or use
// whatever logging library you like.
var WriteLog = func(e LogEntry) {
	if e.Quiet {
		return
	}
	col, reset := logColors(e)
	fmt.Fprintf(os_Stderr, "%s%s %s \"%s %s\" (%d %dB %s) %s%s\n",
		col,
		e.Start.Format(time.RFC3339), e.RemoteIp,
		e.Request.Method, e.Request.RequestURI,
		e.StatusCode, e.ResponseSize, e.Elapsed,
		e.NotesAndError(),
		reset)
}

// NotesAndError formats the Note values and error (if any) for logging.
func (l LogEntry) NotesAndError() string {
	pairs := make([]string, len(l.Note))
	for k, v := range l.Note {
		pairs = append(pairs, fmt.Sprintf("%s=%q", k, v))
	}
	sort.Strings(pairs)
	msg := strings.Join(pairs, " ")
	if l.Error != nil {
		msg += "\n  ERROR: " + l.Error.Error()
	}
	return msg
}

func logColors(e LogEntry) (start, reset string) {
	col, reset := _GREEN, _RESET
	if e.Elapsed > 30*time.Millisecond {
		col = _YELLOW
	}
	if e.StatusCode >= 400 || e.Error != nil {
		col, reset = _RED, _RESET // high-intensity red + reset
	}
	return col, reset
}

// remoteIp extracts the remote IP from the request.  Adapted from code in
// Martini:
//
//	https://github.com/go-martini/martini/blob/1d33529c15f19/logger.go#L14..L20
func remoteIp(r *http.Request) string {
	if addr := r.Header.Get("X-Real-IP"); addr != "" {
		return addr
	} else if addr := r.Header.Get("X-Forwarded-For"); addr != "" {
		return addr
	}
	return r.RemoteAddr
}
