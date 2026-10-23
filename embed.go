// Package situkang holds the build artifacts embedded into the server binary.
package situkang

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:web/dist
var webDist embed.FS

//go:embed migrations/*.sql
var migrationFiles embed.FS

// StaticFS returns the built frontend rooted at web/dist.
func StaticFS() (fs.FS, error) {
	return fs.Sub(webDist, "web/dist")
}

// Migrations returns the embedded SQL migrations rooted at migrations/.
func Migrations() (fs.FS, error) {
	return fs.Sub(migrationFiles, "migrations")
}

// NewSPAHandler serves files from dist and falls back to index.html for
// client-side routes. Paths that look like assets (they have an extension)
// return 404 instead of the SPA shell.
func NewSPAHandler(dist fs.FS) http.Handler {
	files := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}

		if info, err := fs.Stat(dist, name); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}

		if path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}

		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(index)
	})
}
