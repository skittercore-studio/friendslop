package handlers_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/skittercore-studio/friendslop/internal/db"
	"github.com/skittercore-studio/friendslop/internal/events"
	"github.com/skittercore-studio/friendslop/internal/handlers"
	"github.com/skittercore-studio/friendslop/internal/server"
)

// sseTestServer mirrors testServer() from rooms_test.go but additionally
// installs the concrete pubsub + SSE handler that the events package owns.
// It returns the running server and the pubsub (so tests can publish directly).
func sseTestServer(t *testing.T) (*httptest.Server, *events.PerRoomPubsub) {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(context.Background(), filepath.Join(dir, "h.db"))
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	pub := events.NewPubsub()
	r := chi.NewRouter()
	s := server.NewServer(d, server.WithPublisher(pub))

	sseH := handlers.NewSSE(pub)
	s.AddRoutes(func(rt chi.Router) {
		rt.With(s.Sessions.RequireSession).Get("/api/v1/rooms/{code}/events", sseH.Stream)
	})
	s.Mount(r)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, pub
}

// createRoomAsHost creates a room as host and returns the join code plus a
// cookie-bearing http.Client.
func createRoomAsHost(t *testing.T, srv *httptest.Server) (code string, hostClient *http.Client) {
	t.Helper()
	jar, _ := newJar()
	hostClient = &http.Client{Jar: jar}

	resp, body := postJSON(t, hostClient, srv.URL+"/api/v1/rooms", map[string]any{
		"host_name":   "vex",
		"mode":        "live",
		"pool_source": "curated",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", resp.StatusCode, body)
	}
	var created struct {
		RoomCode string `json:"room_code"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return created.RoomCode, hostClient
}

// sseFrame is one parsed Server-Sent Events frame.
type sseFrame struct {
	Event string
	Data  string
}

// readSSEFrames spawns a goroutine that reads SSE frames off r and writes
// them to the returned channel. The channel is closed when the source EOFs
// or the underlying reader errors (typically when the response body is
// closed by the client).
func readSSEFrames(t *testing.T, r io.Reader) <-chan sseFrame {
	t.Helper()
	out := make(chan sseFrame, 16)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(r)
		var cur sseFrame
		for sc.Scan() {
			line := sc.Text()
			if line == "" {
				if cur.Event != "" || cur.Data != "" {
					out <- cur
				}
				cur = sseFrame{}
				continue
			}
			switch {
			case strings.HasPrefix(line, "event: "):
				cur.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				cur.Data = strings.TrimPrefix(line, "data: ")
			}
		}
	}()
	return out
}

func TestSSE_RequiresSession(t *testing.T) {
	srv, _ := sseTestServer(t)
	code, _ := createRoomAsHost(t, srv)

	resp, err := http.Get(srv.URL + "/api/v1/rooms/" + code + "/events")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
}

// TestSSE_RoundTrip opens an SSE stream as the host, then drives a publish
// from two angles:
//  1. Publishing indirectly via the /join handler (which the bootstrap
//     hooked up to call Pub.Publish on player.joined).
//  2. Publishing directly via pubsub.Publish using the room ID surfaced by
//     RoomIDs() once the host has subscribed.
//
// We verify SSE headers, frame format, and that context-cancel cleanly tears
// down the subscription.
func TestSSE_RoundTrip(t *testing.T) {
	srv, pub := sseTestServer(t)
	code, hostClient := createRoomAsHost(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		srv.URL+"/api/v1/rooms/"+code+"/events", nil)
	if err != nil {
		t.Fatalf("req: %v", err)
	}
	resp, err := hostClient.Do(req)
	if err != nil {
		t.Fatalf("sse open: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sse status=%d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content-type=%q want text/event-stream", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("cache-control=%q want no-cache", got)
	}
	if got := resp.Header.Get("X-Accel-Buffering"); got != "no" {
		t.Fatalf("x-accel-buffering=%q want no", got)
	}

	frames := readSSEFrames(t, resp.Body)

	// Wait briefly for the SSE handler to register its subscription so our
	// later publish can find a destination. We poll the pubsub's room set.
	deadline := time.Now().Add(2 * time.Second)
	var roomID string
	for time.Now().Before(deadline) {
		ids := pub.RoomIDs()
		if len(ids) == 1 {
			roomID = ids[0]
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if roomID == "" {
		t.Fatal("SSE subscription never registered with pubsub")
	}

	// (1) Indirect publish via /join — the rooms handler emits player.joined.
	jar2, _ := newJar()
	toy := &http.Client{Jar: jar2}
	jresp, jbody := postJSON(t, toy, srv.URL+"/api/v1/rooms/"+code+"/join",
		map[string]any{"name": "toy"})
	if jresp.StatusCode != http.StatusOK {
		t.Fatalf("join status=%d body=%s", jresp.StatusCode, jbody)
	}

	select {
	case fr := <-frames:
		if fr.Event != "player.joined" {
			t.Fatalf("got event=%q want player.joined", fr.Event)
		}
		var d map[string]any
		if err := json.Unmarshal([]byte(fr.Data), &d); err != nil {
			t.Fatalf("data not JSON: %v raw=%s", err, fr.Data)
		}
		if d["name"] != "toy" {
			t.Fatalf("data.name=%v want toy", d["name"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for player.joined event")
	}

	// (2) Direct publish via the pubsub the handler is subscribed to.
	pub.Publish(roomID, events.Event{
		Name: "round.started",
		Data: map[string]any{"round_number": 1, "question_text": "test?"},
	})

	select {
	case fr := <-frames:
		if fr.Event != "round.started" {
			t.Fatalf("got event=%q want round.started", fr.Event)
		}
		var d map[string]any
		if err := json.Unmarshal([]byte(fr.Data), &d); err != nil {
			t.Fatalf("data not JSON: %v raw=%s", err, fr.Data)
		}
		if d["round_number"].(float64) != 1 {
			t.Fatalf("data.round_number=%v want 1", d["round_number"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for round.started")
	}

	// Cancel: the handler should observe ctx.Done() and unsubscribe. After a
	// brief settle we expect the pubsub's room map to drop the entry.
	cancel()
	resp.Body.Close()

	settled := time.Now().Add(2 * time.Second)
	for time.Now().Before(settled) {
		if len(pub.RoomIDs()) == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("subscription not cleaned up after context cancel; rooms=%v", pub.RoomIDs())
}

// TestSSE_RejectsForeignRoom confirms a session bound to room A cannot tail
// events for room B even with a valid cookie.
func TestSSE_RejectsForeignRoom(t *testing.T) {
	srv, _ := sseTestServer(t)
	codeA, hostA := createRoomAsHost(t, srv)
	codeB, _ := createRoomAsHost(t, srv)
	if codeA == codeB {
		t.Skip("collision in random codes — extremely rare")
	}

	resp, err := hostA.Get(srv.URL + "/api/v1/rooms/" + codeB + "/events")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", resp.StatusCode)
	}
}
