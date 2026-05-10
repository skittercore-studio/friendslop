// Package web embeds the production frontend SPA bundle.
//
// The embed lives at the repository root because Go's embed directive cannot
// traverse up the package tree with "..". Putting the embed.FS in this
// sibling-of-frontend package lets us pull frontend/dist/ into the binary
// without bolting Go source onto the JS project tree.
//
// Build pipeline:
//
//  1. cd frontend && npm run build           (produces frontend/dist/)
//  2. cp -r frontend/dist web/frontend-dist  (or `make assets`)
//  3. go build ./...                         (embeds web/frontend-dist/)
//
// web/frontend-dist/ is gitignored — it's a build artifact mirror. The
// canonical source is frontend/dist/. If web/frontend-dist/ is missing, the
// build fails loudly at compile time, which is the right behaviour: we never
// want to ship a server binary without the SPA inside it.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend-dist
var distFS embed.FS

// FS returns a sub-filesystem rooted at the SPA's index.html, so callers can
// read "index.html" / "assets/foo.js" rather than "frontend-dist/...".
func FS() fs.FS {
	sub, err := fs.Sub(distFS, "frontend-dist")
	if err != nil {
		// fs.Sub only fails on bad input — this is unreachable as long as the
		// embed directive resolves, which it must for the binary to compile.
		panic("web: fs.Sub on embedded dist: " + err.Error())
	}
	return sub
}
