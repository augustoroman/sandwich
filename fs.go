package sandwich

import (
	"io/fs"
	"net/http"
)

// ServeFS is a simple helper that will serve static files from an fs.FS
// filesystem. It allows serving files identified by a sandwich path parameter
// out of a subdirectory of the filesystem. This is especially useful when
// embedding static files:
//
//	//go:embed server_files
//	var all_files embed.FS
//
//	mux.Get("/css/:path*", sandwich.ServeFS(all_files, "static/css", "path"))
//	mux.Get("/js/:path*", sandwich.ServeFS(all_files, "dist/js", "path"))
//	mux.Get("/i/:path*", sandwich.ServeFS(all_files, "static/images", "path"))
func ServeFS(
	f fs.FS,
	fsRoot string,
	pathParam string,
) func(w http.ResponseWriter, r *http.Request, p Params) {
	sub, err := fs.Sub(f, fsRoot)
	if err != nil {
		panic(err)
	}
	handler := http.FileServer(http.FS(sub))
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		r.URL.Path = p[pathParam]
		handler.ServeHTTP(w, r)
	}
}
