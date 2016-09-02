package sandwich

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/augustoroman/sandwich/chain"
)

type fakeClock struct {
	now     time.Time
	advance time.Duration
}

func (f *fakeClock) Now() time.Time {
	now := f.now
	f.now = now.Add(f.advance)
	return now
}

func (f *fakeClock) Sleep(dt time.Duration) {
	f.now = f.now.Add(dt)
}

func validateLogMessage(t *testing.T, logs, expectedColor, expectedMsg string) {
	logs = strings.TrimSpace(logs)

	if !strings.HasPrefix(logs, expectedColor) {
		t.Errorf("Expected color prefix of %q: %q", expectedColor, logs)
	} else {
		logs = strings.TrimPrefix(logs, expectedColor)
	}
	if !strings.HasSuffix(logs, _RESET) {
		t.Errorf("Expected reset suffix: %q", logs)
	} else {
		logs = strings.TrimSuffix(logs, _RESET)
	}
	logs = strings.TrimSpace(logs)
	expectedMsg = strings.TrimSpace(expectedMsg)
	if logs != expectedMsg {
		t.Errorf("Wrong log message:\nExp: %q\nGot: %q", expectedMsg, logs)
	}
}

func TestLogger(t *testing.T) {
	// Restore the world from insanity when we're done:
	orig := WriteLog
	defer func() { time_Now = time.Now; os_Stderr = os.Stderr; WriteLog = orig }()

	// Setup our fake world.
	var logBuf bytes.Buffer
	os_Stderr = &logBuf
	clk := &fakeClock{time.Date(2001, 2, 3, 4, 5, 6, 7, time.UTC), 13 * time.Millisecond}
	time_Now = clk.Now

	// Useful handlers:
	sendMsg := func(w http.ResponseWriter) { w.Write([]byte("Hi there")) }
	slowSendMsg := func(w http.ResponseWriter) { clk.Sleep(100 * time.Millisecond); sendMsg(w) }
	fail := func() error { return errors.New("It went horribly wrong") }
	slowFail := func() error { clk.Sleep(time.Second); return fail() }
	panics := func(w http.ResponseWriter) { sendMsg(w); panic("oops") }
	addsNote := func(w http.ResponseWriter, e *LogEntry) { e.Note["a"] = "x"; e.Note["b"] = "y"; sendMsg(w) }

	var resp *httptest.ResponseRecorder
	var req *http.Request

	// Test a normal response:
	logBuf.Reset()
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/", nil)
	req.RequestURI = req.URL.String()
	req.Header.Add("X-Real-IP", "123.456.789.0")
	TheUsual().With(addsNote).ServeHTTP(resp, req)
	validateLogMessage(t, logBuf.String(), _GREEN,
		`2001-02-03T04:05:06Z 123.456.789.0 "GET /" (200 8B 13ms)   a="x" b="y"`)

	// Test a slow response:
	logBuf.Reset()
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/slow", nil)
	req.RequestURI = req.URL.String()
	req.Header.Add("X-Forwarded-For", "<any string>")
	TheUsual().With(slowSendMsg).ServeHTTP(resp, req)
	validateLogMessage(t, logBuf.String(), _YELLOW,
		`2001-02-03T04:05:06Z <any string> "POST /slow" (200 8B 113ms)`)

	// Test a failed response:
	logBuf.Reset()
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("BOO!", "/fail", nil)
	req.RequestURI = req.URL.String()
	req.RemoteAddr = "[::1]:56596"
	TheUsual().With(fail).ServeHTTP(resp, req)
	validateLogMessage(t, logBuf.String(), _RED,
		`2001-02-03T04:05:06Z [::1]:56596 "BOO! /fail" (500 22B 13ms) `+"\n"+
			`  ERROR: (500) Failure: It went horribly wrong`)

	// Test a slow failed response (should still be red):
	logBuf.Reset()
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/slowfail", nil)
	req.RequestURI = req.URL.String()
	req.RemoteAddr = "[::1]:56596"
	req.Header.Add("X-Forwarded-For", "<any string>")
	req.Header.Add("X-Real-IP", "123.456.789.0") // takes precedence
	TheUsual().With(slowFail).ServeHTTP(resp, req)
	validateLogMessage(t, logBuf.String(), _RED,
		`2001-02-03T04:05:06Z 123.456.789.0 "PUT /slowfail" (500 22B 1.013s) `+"\n"+
			`  ERROR: (500) Failure: It went horribly wrong`)

	// Test a suppressed log.
	logBuf.Reset()
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/", nil)
	TheUsual().With(NoLog, addsNote).ServeHTTP(resp, req)
	if logBuf.String() != "" {
		t.Errorf("Expected no log output, but got [%s]", logBuf.String())
	}

	// Test that a panic should be recorded.
	var log LogEntry
	WriteLog = func(e LogEntry) { log = e }
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/slowfail", nil)
	req.RequestURI = req.URL.String()
	req.RemoteAddr = "<remote>"
	TheUsual().With(panics).ServeHTTP(resp, req)

	if err, ok := ToError(log.Error).Cause.(chain.PanicError); !ok {
		t.Errorf("log error should be a panic, but is: %#v", log.Error)
	} else if msg := err.Error(); !strings.Contains(msg, `Panic executing middleware`) {
		t.Errorf("Bad err message: %s", err)
	} else if !strings.Contains(msg, `oops`) {
		t.Errorf("Bad err message: %s", err)
	}

	if resp.Body.String() != "Hi thereInternal Server Error\n" {
		t.Errorf("Incorrect client response: %q", resp.Body.String())
	}
}
