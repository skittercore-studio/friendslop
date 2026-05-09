// setup.go — package wiring.
//
// Setup is the single entry point that the orchestrator (cmd/friendslop/main.go)
// calls to bolt the gamelogic package onto a Server. It loads the embedded
// content, builds an Engine, registers the game-action route callback via
// s.AddRoutes, and starts the background timer goroutine.
//
// The timer respects the supplied ctx — cancel ctx and the goroutine returns.
//
// The handlers themselves live in internal/game/handlers.go (NOT
// internal/handlers/game.go as originally sketched in the brief) to avoid an
// import cycle: internal/handlers imports internal/events and the rooms
// handler depends on those types, while internal/game depends on
// internal/events too — putting the game handlers inside internal/game keeps
// us off a circular reference. The HTTP surface is identical.
package game

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"

	"github.com/skittercore-studio/friendslop/internal/server"
)

// Setup wires the gamelogic surface onto s. ctx controls the timer goroutine
// lifetime — pass the rootCtx from main so SIGTERM stops the engine.
//
// Caller is responsible for ensuring s.Mount is called AFTER this function so
// the registered routes are picked up.
func Setup(ctx context.Context, s *server.Server) error {
	content, err := LoadDefaultContent()
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}

	engine := NewEngine(s.DB, s.Pub, content)
	h := newHandler(s.DB, s.Pub, content, engine, s.Sessions.RequireSession)

	s.AddRoutes(func(r chi.Router) {
		h.Mount(r)
	})

	go engine.Run(ctx)
	return nil
}
