package sandwich

import (
	"embed"
	"io/fs"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed examples
var examples embed.FS

func TestServeFS(t *testing.T) {
	helloworld := ServeFS(examples, "examples/0-helloworld", "path")
	w := httptest.NewRecorder()
	helloworld(w, httptest.NewRequest("", "/foo", nil), Params{"path": "main.go"})
	contents, err := fs.ReadFile(examples, "examples/0-helloworld/main.go")
	require.NoError(t, err)
	assert.Equal(t, string(contents), w.Body.String())
}
