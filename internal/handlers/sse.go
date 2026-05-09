// sse.go implements the per-room Server-Sent Events stream.
//
// Route: GET /api/v1/rooms/{code}/events
// Auth:  session cookie required; the cookie must belong to the room being
//
//	streamed (verified in-handler against the resolved session).
//
// Wire format follows the SSE spec (text/event-stream, "event: " + "\n" +
// "data: " + JSON + "\n\n"). Heartbeats fire every 25 seconds to keep idle
// proxies (Caddy, nginx, Cloudflare) from killing the connection.
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/skittercore-studio/friendslop/internal/events"
	"github.com/skittercore-studio/friendslop/internal/session"
)

// heartbeatInterval is how often we emit a "heartbeat" SSE event to keep
// proxies + load balancers from idle-killing the long-lived response.
const heartbeatInterval = 25 * time.Second

// SSE bundles the dependencies the events handler needs: the concrete pubsub
// to subscribe against, plus implicit session resolution via the standard
// RequireSession middleware applied at mount time.
type SSE struct {
	Pub *events.PerRoomPubsub
}

// NewSSE constructs an SSE handler bound to the given pubsub.
func NewSSE(pub *events.PerRoomPubsub) *SSE {
	return &SSE{Pub: pub}
}

// Stream is the http.HandlerFunc registered at /api/v1/rooms/{code}/events.
//
// It expects the request to have already passed the session-required
// middleware (so a Session is present on the context); it additionally
// verifies that the session's room matches the {code} in the URL to prevent
// cross-room leakage if a player has stale cookies for a different room.
func (h *SSE) Stream(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))

	s, err := session.FromContext(r.Context())
	if err != nil {
		http.Error(w, "no session", http.StatusUnauthorized)
		return
	}
	if s.RoomCode != code {
		http.Error(w, "session does not belong to this room", http.StatusForbidden)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		// Should never happen with net/http's default handler; if some
		// future middleware wraps the writer in a non-flushing type, fail
		// loudly rather than silently buffering.
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Headers MUST be set before WriteHeader/first flush.
	h.writeSSEHeaders(w)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, unsubscribe := h.Pub.Subscribe(s.RoomID)
	defer unsubscribe()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				// Pubsub closed our channel (e.g. from forced unsubscribe).
				return
			}
			if err := writeEvent(w, flusher, ev.Name, ev.Data); err != nil {
				log.Printf("sse: write event %q failed in room %s: %v", ev.Name, s.RoomID, err)
				return
			}
		case <-ticker.C:
			payload := map[string]int64{"ts": time.Now().UnixMilli()}
			if err := writeEvent(w, flusher, "heartbeat", payload); err != nil {
				// Client disconnected — exit cleanly; defer unsubscribes.
				return
			}
		}
	}
}

// writeSSEHeaders sets the canonical SSE response headers. X-Accel-Buffering
// is the nginx hint to disable response buffering; harmless in front of other
// proxies.
func (h *SSE) writeSSEHeaders(w http.ResponseWriter) {
	hdr := w.Header()
	hdr.Set("Content-Type", "text/event-stream")
	hdr.Set("Cache-Control", "no-cache")
	hdr.Set("Connection", "keep-alive")
	hdr.Set("X-Accel-Buffering", "no")
}

// writeEvent serialises a single SSE frame: "event: NAME\ndata: JSON\n\n".
// The double-newline is required by the spec to mark frame end. We flush
// immediately so the client receives the event without buffering.
func writeEvent(w http.ResponseWriter, flusher http.Flusher, name string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		// Fall back to a JSON null rather than corrupting the stream.
		payload = []byte("null")
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, payload); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
