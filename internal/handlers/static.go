// Package handlers — static.go serves the embedded frontend SPA.
//
// The frontend agent owns frontend/. Until they ship a dist/ tree, the embed
// directive points at a placeholder directory shipped in this package
// (dist_stub) so the build succeeds even on a fresh clone. When the frontend
// build pipeline starts producing frontend/dist, swap the embed to point at
// that path (or use a build tag to switch between dev/prod assets).
package handlers

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// frontendDist is populated from the dist_stub placeholder so the embed
// directive succeeds out of the box. The frontend agent will replace this
// with real assets via a build tag.
//
//go:embed all:dist_stub
var frontendDist embed.FS

// Static bundles the static-handler dependencies (currently none, kept for
// symmetry with Rooms and to give the frontend agent a hook to inject a
// production FS later).
type Static struct{}

// NewStatic returns a Static handler.
func NewStatic() *Static { return &Static{} }

// Mount serves "/" and any non-API path from the embedded SPA bundle. The
// router order in server.Server.Mount ensures /api/v1/... is matched first.
func (s *Static) Mount(r chi.Router) {
	r.Get("/*", s.serve)
}

func (s *Static) serve(w http.ResponseWriter, r *http.Request) {
	urlPath := strings.TrimPrefix(r.URL.Path, "/")
	if urlPath == "" {
		urlPath = "index.html"
	}

	if sub, err := fs.Sub(frontendDist, "dist_stub"); err == nil {
		if data, err := fs.ReadFile(sub, urlPath); err == nil {
			writeStatic(w, urlPath, data)
			return
		}
		// SPA fallback — any unknown path returns index.html so client-side
		// routes work on direct navigation.
		if data, err := fs.ReadFile(sub, "index.html"); err == nil {
			writeStatic(w, "index.html", data)
			return
		}
	}

	// Last-ditch placeholder so the server is functional pre-frontend.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(placeholderHTML))
}

func writeStatic(w http.ResponseWriter, path string, data []byte) {
	switch {
	case strings.HasSuffix(path, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(path, ".json"):
		w.Header().Set("Content-Type", "application/json")
	}
	_, _ = w.Write(data)
}

const placeholderHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>friendslop</title></head>
<body style="font-family: system-ui; padding: 2rem;">
<h1>friendslop</h1>
<p>API up. Frontend bundle not yet embedded — try <code>POST /api/v1/rooms</code>.</p>
</body></html>
`
