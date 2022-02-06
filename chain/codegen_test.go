package chain

import (
	"bytes"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

func normalizeWhitespace(s string) string {
	return regexp.MustCompile(`\s+`).ReplaceAllLiteralString(strings.TrimSpace(s), " ")
}

type TestDb struct{}

func (t *TestDb) Validate(s string) {}

func TestCodeGen(t *testing.T) {
	var buf bytes.Buffer

	type User struct{}
	type Http struct{}

	New().
		Arg((*http.ResponseWriter)(nil)).
		Set("").
		Set(int64(0)).
		Set(int(1)).
		Set((*User)(nil)).
		Set(User{}).
		Set(Http{}).
		Set(&TestDb{}).
		Then((*TestDb).Validate).
		Then(a, b, c).
		Code("foo", "chain", &buf)

	const expected = `func foo(
        str string,
        i64 int64,
        i int,
        pUser *User,
        user User,
        chain_Http Http,
        pTestDb *TestDb,
      ) func(
        rw http.ResponseWriter,
      ) {
        return func(
          rw http.ResponseWriter,
        ) {
          (*TestDb).Validate(pTestDb, str)

          str = a()

          str, i = b(str)

          c(str, i)

        }
      }`
	if normalizeWhitespace(buf.String()) != normalizeWhitespace(expected) {
		t.Errorf("Wrong code generated: %s\nExp: %q\nGot: %q", buf.String(),
			normalizeWhitespace(expected), normalizeWhitespace(buf.String()))
	}
}
