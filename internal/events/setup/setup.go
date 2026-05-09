// Package setup wires the in-process pubsub + SSE handler into a Server.
//
// It lives in a sub-package rather than directly under internal/events because
// internal/server already imports internal/events (for the Publisher
// interface), so a Setup function inside internal/events that takes
// *server.Server would create an import cycle. Putting the wiring in a sibling
// sub-package keeps the cycle broken while still letting cmd/friendslop/main.go
// call a single entry point.
//
// Wiring order: callers MUST invoke setup.Wire(s) BEFORE any other Setup
// functions that publish through s.Pub (notably internal/game.Setup), so that
// gamelogic publishes through the real pubsub instead of the bootstrap
// NoopPublisher. setup.Wire MUST also run before s.Mount(r), because Mount
// constructs handlers.NewRooms(s.DB, s.Pub) at call time and snapshots the
// publisher.
package setup

import (
	"github.com/go-chi/chi/v5"

	"github.com/skittercore-studio/friendslop/internal/events"
	"github.com/skittercore-studio/friendslop/internal/handlers"
	"github.com/skittercore-studio/friendslop/internal/server"
)

// Wire installs the concrete PerRoomPubsub on s and registers the SSE handler
// at GET /api/v1/rooms/{code}/events behind the session-required middleware.
//
// Returns an error in case future variants of this function need to fail
// (e.g. config validation). The current implementation is infallible.
func Wire(s *server.Server) error {
	pub := events.NewPubsub()
	s.Pub = pub

	sse := handlers.NewSSE(pub)
	s.AddRoutes(func(r chi.Router) {
		r.With(s.Sessions.RequireSession).
			Get("/api/v1/rooms/{code}/events", sse.Stream)
	})

	return nil
}
