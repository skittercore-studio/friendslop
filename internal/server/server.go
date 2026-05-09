// Package server wires friendslop's HTTP surface together.
//
// Extension hooks for sibling agents:
//
//  1. Server exposes (s *Server).AddRoutes(fn func(chi.Router)) — gamelogic
//     and events agents call this from their own Setup function before
//     Server.Mount runs. Their handlers can therefore register additional
//     /api/v1/... routes without editing this file.
//
//  2. Server.Pub satisfies events.Publisher. Gamelogic publishes via
//     `s.Pub.Publish(roomID, ev)`. The events agent will swap the
//     NoopPublisher default for its concrete PerRoomPubsub via
//     server.WithPublisher when constructing Server, or by reassigning
//     s.Pub before Mount.
//
//  3. DB access pattern: Server.DB is a public *sql.DB so other handler
//     constructors can take it as a parameter. Pass it explicitly into
//     handler structs — do NOT depend on package-level globals.
package server

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/skittercore-studio/friendslop/internal/events"
	"github.com/skittercore-studio/friendslop/internal/handlers"
	"github.com/skittercore-studio/friendslop/internal/session"
)

// Server is the composition root. It owns the DB handle, the events publisher,
// the session resolver, and an extension list other agents can append to.
type Server struct {
	DB       *sql.DB
	Pub      events.Publisher
	Sessions *session.Resolver

	// extra holds late-registered route registrar callbacks. AddRoutes
	// appends here; Mount drains them in registration order.
	extra []func(chi.Router)
}

// Option is a functional option for NewServer.
type Option func(*Server)

// WithPublisher overrides the default NoopPublisher.
func WithPublisher(p events.Publisher) Option {
	return func(s *Server) { s.Pub = p }
}

// NewServer builds a Server bound to a DB handle. Use Option to swap the
// publisher; otherwise events are dropped silently.
func NewServer(d *sql.DB, opts ...Option) *Server {
	s := &Server{
		DB:       d,
		Pub:      events.NoopPublisher{},
		Sessions: session.NewResolver(d),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// AddRoutes registers a callback to be invoked at Mount time. Other agents call
// this from their package's Setup(s *Server) function. Callbacks run in
// registration order, after the room handlers are mounted.
func (s *Server) AddRoutes(fn func(chi.Router)) {
	if fn == nil {
		return
	}
	s.extra = append(s.extra, fn)
}

// Mount installs all friendslop routes onto r. The caller is responsible for
// serving r — this method does not start a listener.
//
// Route order:
//  1. /healthz (no middleware)
//  2. session-resolving optional middleware on the whole tree
//  3. room CRUD (this package's handlers/rooms.go)
//  4. extension callbacks (gamelogic, events)
//  5. static SPA catch-all (matches last so /api/v1/... wins)
func (s *Server) Mount(r chi.Router) {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.Sessions.OptionalSession)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	rooms := handlers.NewRooms(s.DB, s.Pub)
	rooms.Mount(r, s.Sessions.RequireSession)

	for _, fn := range s.extra {
		fn(r)
	}

	static := handlers.NewStatic()
	static.Mount(r)
}
