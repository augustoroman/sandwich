package chain

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func normalizeWhitespace(s string) string {
	return regexp.MustCompile(`\s+`).ReplaceAllLiteralString(strings.TrimSpace(s), " ")
}

func TestCodeGen(t *testing.T) {
	var buf bytes.Buffer
	New().Reserve("").With(a, b, c).Code("foo", "bar", &buf)

	const expected = `func foo(
      ) func(
        string_val string,
      ) {
        return func(
          string_val string,
        ) {
          string_val = chain.a()

          var int_val int
          string_val, int_val = chain.b(string_val)

          chain.c(string_val, int_val)

        }
      }`
	if normalizeWhitespace(buf.String()) != normalizeWhitespace(expected) {
		t.Errorf("Wrong code generated: %s\nExp: %q\nGot: %q", buf.String(),
			normalizeWhitespace(expected), normalizeWhitespace(buf.String()))
	}
}
