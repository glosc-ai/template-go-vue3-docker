// Package webassets embeds the built Vue frontend into the API binary so
// production ships as a single image with no separate web/nginx container.
// `dist/index.html` here is a placeholder for local `go run`/`go test`;
// the production Dockerfile overwrites dist/ with the real Vite build before
// compiling.
package webassets

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var distFS embed.FS

// Handler serves the embedded frontend with SPA fallback: unknown paths
// (Vue Router routes) resolve to index.html, while /api/ and /health/ paths
// that reach it (i.e. unmatched by the API mux) get a JSON 404 instead of HTML.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/health/") {
			writeNotFound(w)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "."
		}
		if info, err := fs.Stat(sub, path); err != nil || info.IsDir() {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}), nil
}

func writeNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"resource not found"}}`))
}
