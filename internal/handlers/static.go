// Package handlers — static.go serves the embedded frontend SPA.
//
// The actual embed.FS lives in github.com/skittercore-studio/friendslop/web
// because Go embed directives can't reach across package boundaries with
// "..". This file just wires that FS to chi routes and handles SPA fallback
// (any unknown path returns index.html so client-side routing works on
// direct navigation and refresh).
package handlers

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/skittercore-studio/friendslop/web"
)

// Static bundles the static-handler dependencies. The embedded SPA is held
// as an fs.FS so tests can swap in a fake filesystem without touching the
// real bundle.
type Static struct {
	dist fs.FS
}

// NewStatic returns a Static handler backed by the embedded frontend bundle.
func NewStatic() *Static { return &Static{dist: web.FS()} }

// NewStaticFS returns a Static handler backed by a caller-supplied FS. Used
// in tests to simulate missing bundles or specific assets.
func NewStaticFS(dist fs.FS) *Static { return &Static{dist: dist} }

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

	if data, err := fs.ReadFile(s.dist, urlPath); err == nil {
		writeStatic(w, urlPath, data)
		return
	}
	// SPA fallback — any unknown path returns index.html so client-side
	// routes work on direct navigation.
	if data, err := fs.ReadFile(s.dist, "index.html"); err == nil {
		writeStatic(w, "index.html", data)
		return
	}

	// Last-ditch placeholder so the server is functional even if the embed
	// is empty (shouldn't happen — embed fails at compile time if missing).
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
	case strings.HasSuffix(path, ".map"):
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
