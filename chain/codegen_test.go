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

func TestCodeGen(t *testing.T) {
	var buf bytes.Buffer

	type User struct{}
	type Http struct{}

	New().
		Reserve((*http.ResponseWriter)(nil)).
		Provide("").
		Provide(int64(0)).
		Provide(int(1)).
		Provide((*User)(nil)).
		Provide(User{}).
		Provide(Http{}).
		With(a, b, c).
		Code("foo", "bar", &buf)

	const expected = `func foo(
        str string,
        i64 int64,
        i int,
        pUser *chain.User,
        user chain.User,
        chain_Http chain.Http,
      ) func(
        rw http.ResponseWriter,
      ) {
        return func(
	      rw http.ResponseWriter,
        ) {
          str = chain.a()

          str, i = chain.b(str)

          chain.c(str, i)
        }
      }`
	if normalizeWhitespace(buf.String()) != normalizeWhitespace(expected) {
		t.Errorf("Wrong code generated: %s\nExp: %q\nGot: %q", buf.String(),
			normalizeWhitespace(expected), normalizeWhitespace(buf.String()))
	}
}
