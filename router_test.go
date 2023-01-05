package sandwich

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shorthand for keeping tests concise below
type M = Params

func TestMuxRegisterAndMatch(t *testing.T) {
	const REGISTRATION_ERROR = "•ERR:"
	fail := func(reason string) string {
		return REGISTRATION_ERROR + reason
	}
	split := func(combined_pattern string) (pattern, errmsg string) {
		pos := strings.Index(combined_pattern, REGISTRATION_ERROR)
		if pos == -1 {
			return combined_pattern, ""
		}
		return combined_pattern[:pos], combined_pattern[pos+len(REGISTRATION_ERROR):]
	}
	patterns := []string{
		"/",
		"/a",
		"/a" + fail("repeated entry"),
		"/a/:x/:x" + fail("repeated param name"),
		"/a/",
		"/a/b",
		"/a/b/c",
		"/a/b/c" + fail("repeated entry"),
		"/a/b/c/d/e", // NOTE: /a/b/c/d not registered
		"/a/:x/c",
		"/a/:x/c" + fail("repeated entry"),
		"/a/:y/c" + fail("ambiguous param var"),
		"/a/:y/c2",
		"/a/:m*",
		"/a/:m*/",
		"/b/:a*/x",
		"/b/:b*/y",
		"/b/:b*/x" + fail("ambiguous greedy pattern"),
		"/c/:x/y",
		"/c/:x*/y" + fail("ambiguous param (greedy or not)"),
		"/:m*/b/c",
		"/:m*/:x/c",
		"/:m*/:x*/c" + fail("multiple greedy patterns"),
		"/:x*/b/c" + fail("ambiguous greedy var"),
		"/x/:x*/y/:y/z/:z*/blah" + fail("multiple greedy patterns"),

		// literal colon in static URL
		"/a/::x",
		"/a/::x/c",
	}

	var m mux

	for _, combo_pattern := range patterns {
		pattern, errmsg := split(combo_pattern)
		err := m.Register(pattern, noopHandler(pattern))
		if errmsg == "" {
			require.NoError(t, err)
		} else {
			require.Error(t, err, "Pattern %#q should have failed: %s", pattern, errmsg)
		}
	}

	// priority:
	//  - static routes
	//  - explicit parameter
	//  - greedy parameter

	testCases := []struct {
		uri             string
		expectedHandler noopHandler
		expectedParams  M
	}{
		{"/", "/", M{}},
		{"/a", "/a", M{}},
		{"/a/", "/a/", M{}},
		{"/a/b", "/a/b", M{}},
		{"/a/b/c", "/a/b/c", M{}},
		{"/a/b/c/d/e", "/a/b/c/d/e", M{}},
		{"/a/b/c/d", "/a/:m*", M{"m": "b/c/d"}},

		{"/a/foobar/c", "/a/:x/c", M{"x": "foobar"}},
		{"/a/foobar/c2", "/a/:y/c2", M{"y": "foobar"}},

		{"/a/foobar/blah", "/a/:m*", M{"m": "foobar/blah"}},
		{"/a/foobar/blah/", "/a/:m*/", M{"m": "foobar/blah"}},

		{"/b/mm/nn/", "", nil},
		{"/b/mm/nn/x", "/b/:a*/x", M{"a": "mm/nn"}},
		{"/b/mm/nn/y", "/b/:b*/y", M{"b": "mm/nn"}},

		{"/c/x/y", "/c/:x/y", M{"x": "x"}},

		{"/b/x/y/b/c", "/:m*/b/c", M{"m": "b/x/y"}},
		{"/b/x/y/bo/c", "/:m*/:x/c", M{"m": "b/x/y", "x": "bo"}},
	}

	for _, test := range testCases {
		t.Run(fmt.Sprintf("%s -> %s", test.uri, test.expectedHandler), func(t *testing.T) {
			if test.expectedHandler == "" {
				t.Logf("Testing input uri %#q --> should not match any pattern",
					test.uri)
			} else {
				t.Logf("Testing input uri %#q --> should match pattern %#q",
					test.uri, test.expectedHandler)
			}
			params := Params{}
			selected := m.Match(test.uri, params)
			if test.expectedHandler == "" {
				assert.Nil(t, selected, "should not match any pattern")
				assert.Empty(t, params)
			} else {
				require.NotNil(t, selected)
				assert.Equal(t, test.expectedHandler, selected)
				assert.Equal(t, test.expectedParams, params)
			}
		})
	}
}

type noopHandler string

func (h noopHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, p Params) {}

func TestRouter(t *testing.T) {
	r := TheUsual()

	type UserID string
	type User string
	type UserDB map[UserID]User

	theUserDB := UserDB{"1": "bob", "2": "alice"}
	r.Set(theUserDB)

	loadUser := func(db UserDB, p Params) (User, UserID, error) {
		if uid := UserID(p["userID"]); uid == "" {
			return "", "", Error{Code: 400, ClientMsg: "Must specify user ID"}
		} else if u := db[uid]; u == "" {
			return "", uid, Error{Code: 404, ClientMsg: "No such user"}
		} else {
			return u, uid, nil
		}
	}
	newUserFromRequest := func(r *http.Request) (UserID, User, error) {
		uid := UserID(r.FormValue("uid"))
		user := User(r.FormValue("name"))
		if uid == "" {
			return "", "", errors.New("missing user id")
		} else if user == "" {
			return "", "", errors.New("missing user info")
		}
		return uid, user, nil
	}

	r.Get("/user/:userID", loadUser,
		func(w http.ResponseWriter, u User) {
			fmt.Fprintf(w, "Hi user %#q", u)
		},
	)
	r.Post("/user/", newUserFromRequest,
		func(db UserDB, uid UserID, u User) { db[uid] = u },
		func(w http.ResponseWriter, uid UserID, u User) {
			fmt.Fprintf(w, "Made user %#q = %#q", uid, u)
		},
	)
	r.Any("/user/:userID/:cmd*", loadUser,
		func(w http.ResponseWriter, r *http.Request, p Params, u User) {
			fmt.Fprintf(w, "Doing %#q (%s) to user %#q", r.Method, p["cmd"], u)
		},
	)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/user/1", nil))
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.Equal(t, "Hi user `bob`", w.Body.String())

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/user/2", nil))
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.Equal(t, "Hi user `alice`", w.Body.String())

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/user/3", nil))
	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
	assert.Equal(t, "No such user\n", w.Body.String())

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/user/?uid=3&name=sid", nil))
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "Response: %s", w.Body.String())

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/user/3", nil))
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.Equal(t, "Hi user `sid`", w.Body.String())

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("EXPLODE", "/user/3/boom", nil))
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.Equal(t, "Doing `EXPLODE` (boom) to user `sid`", w.Body.String())
}

// func TestNodeMatch(t *testing.T) {
// 	testCases := []struct {
// 		path, pattern  string
// 		matches        bool
// 		expectedParams M
// 	}{
// 		// static paths
// 		{"/a/b/c", "/a/b/c", true, M{}},
// 		{"/a/b/c", "/x/b/c", false, nil},
// 		{"/a/b/c", "/a/b", false, nil},
// 		{"/a/b/c", "/a/b/c/d", false, nil},

// 		{"/a/b/c", "/a/b/:last", true, M{"last": "c"}},
// 		{"/a/b/c", "/a/:mid/c", true, M{"mid": "b"}},
// 		{"/a/b/c", "/:first/:mid/c", true, M{"first": "a", "mid": "b"}},
// 		{"/a/b/c", "/:first/:mid/:last", true, M{"first": "a", "mid": "b", "last": "c"}},
// 		{"/a/b/c", "/:first/:mid/:last/:missing", false, nil},
// 		{"/a/b/c", "/:first/:mid/:last/x", false, nil},

// 		{"/a/b/c/d/e/f/g", "/:path*", true, M{"path": "a/b/c/d/e/f/g"}},
// 		{"/a/b/c/d/e/f/g", "/a/:path*", true, M{"path": "b/c/d/e/f/g"}},
// 		{"/a/b/c/d/e/f/g", "/a/:path*/g", true, M{"path": "b/c/d/e/f"}},
// 		{"/a/b/c/d/e/f/g", "/a/:path*/f/g", true, M{"path": "b/c/d/e"}},
// 		{"/a/b/c/d/e/f/g", "/a/:path*/:last", true, M{"path": "b/c/d/e/f", "last": "g"}},
// 		{"/a/b/c/d/e/f/g", "/a/:first/:mid*/:last", true, M{"first": "b", "mid": "c/d/e/f", "last": "g"}},
// 		{"/a/b/c/d/e/f/g", "/a/:first*/:mid/:last", true, M{"first": "b/c/d/e", "mid": "f", "last": "g"}},
// 		{"/a/b/c/d/e/f/g", "/a/:first/:mid/:last*", true, M{"first": "b", "mid": "c", "last": "d/e/f/g"}},
// 	}

// 	for i, test := range testCases {
// 		t.Run(fmt.Sprintf("%d:%s", i, test.pattern), func(t *testing.T) {
// 			t.Logf("Test %d: Path %#q  Pattern %#q   Should match: %v",
// 				i, test.path, test.pattern, test.matches)
// 			root, err := makeMatchNodes(test.pattern)
// 			require.NoError(t, err)
// 			pathSegments := strings.Split(test.path, "/")
// 			params := Params{}
// 			match := root.match(pathSegments[1:], params)
// 			if test.matches {
// 				require.NotNil(t, match)
// 				assert.Equal(t, test.expectedParams, params)
// 			} else {
// 				assert.Nil(t, match)
// 			}
// 		})
// 	}
// }
