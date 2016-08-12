package sandwich

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// Injected for testing
var time_Now = time.Now

// LogEntry is the information tracked on a per-request basis for the sandwich
// Logger.  All fields other than Note are automatically filled in.  The Note
// field is a generic key-value string map for adding additional per-request
// metadata to the logs.  You can take *sandwich.LogEntry to your functions to
// add fields to Note.
//
// For example:
//
//    func MyAuthCheck(r *http.Request, e *sandwich.LogEntry) (User, error) {
//        user, err := decodeAuthCookie(r)
//        if user != nil {
//            e.Note["user"] = user.Id()  // indicate which user is auth'd
//        }
//        return user, err
//    }
type LogEntry struct {
	RemoteIp     string
	Start        time.Time
	Request      *http.Request
	StatusCode   int
	ResponseSize int
	Elapsed      time.Duration
	Error        error
	Note         map[string]string
}

// StartLog creates a *LogEntry and initializes it with basic request
// information.
func StartLog(r *http.Request) *LogEntry {
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

// WriteLog is called to actually write a LogEntry out to the log. By default,
// it writes to stderr and colors errors red, however you can replace the
// function to adjust the formatting or whatever logging library you like.
var WriteLog = func(e LogEntry) {
	col, reset := "", ""
	if e.StatusCode >= 400 || e.Error != nil {
		col, reset = "\033[91m", "\033[0m" // high-intensity red + reset
	}
	fmt.Fprintf(os.Stderr, "%s%s %s \"%s %s\" (%d %dB %s) %s%s\n",
		col,
		e.Start.Format(time.RFC3339), e.RemoteIp,
		e.Request.Method, e.Request.URL.Path,
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

// remoteIp extracts the remote IP from the request.  Adapted from code in
// Martini:
//   https://github.com/go-martini/martini/blob/1d33529c15f19/logger.go#L14..L20
func remoteIp(r *http.Request) string {
	if addr := r.Header.Get("X-Real-IP"); addr != "" {
		return addr
	} else if addr := r.Header.Get("X-Forwarded-For"); addr != "" {
		return addr
	}
	return r.RemoteAddr
}
