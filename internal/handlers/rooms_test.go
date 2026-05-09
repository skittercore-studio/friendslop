package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/skittercore-studio/friendslop/internal/db"
	"github.com/skittercore-studio/friendslop/internal/server"
)

// testServer spins up an in-memory server with a tempdir SQLite store.
func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(context.Background(), filepath.Join(dir, "h.db"))
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	r := chi.NewRouter()
	s := server.NewServer(d)
	s.Mount(r)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, c *http.Client, url string, body interface{}) (*http.Response, []byte) {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, out
}

func getJSON(t *testing.T, c *http.Client, url string) (*http.Response, []byte) {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, out
}

func TestRoomCreateJoinSnapshot(t *testing.T) {
	srv := testServer(t)
	jar, _ := newJar()
	hostClient := &http.Client{Jar: jar}

	// Create room as host vex.
	resp, body := postJSON(t, hostClient, srv.URL+"/api/v1/rooms", map[string]any{
		"host_name":   "vex",
		"mode":        "live",
		"pool_source": "curated",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", resp.StatusCode, body)
	}
	var created struct {
		RoomCode     string `json:"room_code"`
		PlayerID     string `json:"player_id"`
		SessionToken string `json:"session_token"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("unmarshal create: %v body=%s", err, body)
	}
	if len(created.RoomCode) != 4 {
		t.Fatalf("expected 4-letter code, got %q", created.RoomCode)
	}
	if created.SessionToken == "" || created.PlayerID == "" {
		t.Fatalf("missing fields: %+v", created)
	}
	for _, r := range created.RoomCode {
		if r == 'I' || r == 'O' {
			t.Fatalf("forbidden char in code %q", created.RoomCode)
		}
	}

	// Public snapshot before any joins beyond host.
	resp, body = getJSON(t, hostClient, srv.URL+"/api/v1/rooms/"+created.RoomCode)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("snapshot status=%d body=%s", resp.StatusCode, body)
	}
	var snap struct {
		Code    string `json:"code"`
		State   string `json:"state"`
		Players []struct {
			Name   string `json:"name"`
			IsHost bool   `json:"is_host"`
		} `json:"players"`
	}
	if err := json.Unmarshal(body, &snap); err != nil {
		t.Fatalf("unmarshal snapshot: %v body=%s", err, body)
	}
	if snap.State != "lobby" {
		t.Fatalf("want state=lobby got %s", snap.State)
	}
	if len(snap.Players) != 1 || snap.Players[0].Name != "vex" || !snap.Players[0].IsHost {
		t.Fatalf("unexpected players: %+v", snap.Players)
	}

	// A second player joins with a fresh client / cookie jar.
	jar2, _ := newJar()
	toyClient := &http.Client{Jar: jar2}
	resp, body = postJSON(t, toyClient, srv.URL+"/api/v1/rooms/"+created.RoomCode+"/join",
		map[string]any{"name": "toy"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("join status=%d body=%s", resp.StatusCode, body)
	}
	var joined struct {
		PlayerID     string `json:"player_id"`
		SessionToken string `json:"session_token"`
	}
	if err := json.Unmarshal(body, &joined); err != nil {
		t.Fatalf("unmarshal join: %v body=%s", err, body)
	}
	if joined.PlayerID == "" || joined.SessionToken == created.SessionToken {
		t.Fatalf("token must differ between players: created=%s joined=%s",
			created.SessionToken, joined.SessionToken)
	}

	// Snapshot now has both players.
	resp, body = getJSON(t, hostClient, srv.URL+"/api/v1/rooms/"+created.RoomCode)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("snapshot2 status=%d body=%s", resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, &snap); err != nil {
		t.Fatalf("unmarshal snapshot2: %v", err)
	}
	if len(snap.Players) != 2 {
		t.Fatalf("want 2 players, got %d", len(snap.Players))
	}

	// Duplicate name should 409.
	jar3, _ := newJar()
	dupClient := &http.Client{Jar: jar3}
	resp, _ = postJSON(t, dupClient, srv.URL+"/api/v1/rooms/"+created.RoomCode+"/join",
		map[string]any{"name": "toy"})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("dup name should 409, got %d", resp.StatusCode)
	}

	// /me on host returns 200 with cookie.
	resp, body = getJSON(t, hostClient, srv.URL+"/api/v1/rooms/"+created.RoomCode+"/me")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/me status=%d body=%s", resp.StatusCode, body)
	}
	var me struct {
		PlayerID string `json:"player_id"`
		Name     string `json:"name"`
		IsHost   bool   `json:"is_host"`
	}
	if err := json.Unmarshal(body, &me); err != nil {
		t.Fatalf("unmarshal /me: %v body=%s", err, body)
	}
	if me.PlayerID != created.PlayerID || me.Name != "vex" || !me.IsHost {
		t.Fatalf("unexpected /me: %+v", me)
	}

	// /me without cookie is 401.
	bare := &http.Client{}
	resp, _ = getJSON(t, bare, srv.URL+"/api/v1/rooms/"+created.RoomCode+"/me")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bare /me should 401, got %d", resp.StatusCode)
	}

	// /abandon is now owned by the gamelogic package (internal/game) and is
	// not mounted by the bare server.Server used in this test. We don't
	// exercise it here — see internal/game/handlers_test.go for the real
	// abandon coverage.
}

func TestRoomCreateValidation(t *testing.T) {
	srv := testServer(t)
	jar, _ := newJar()
	c := &http.Client{Jar: jar}

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing host_name", map[string]any{"mode": "live", "pool_source": "curated"}},
		{"bad mode", map[string]any{"host_name": "x", "mode": "telegraph", "pool_source": "curated"}},
		{"bad pool_source", map[string]any{"host_name": "x", "mode": "live", "pool_source": "wiki"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := postJSON(t, c, srv.URL+"/api/v1/rooms", tc.body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("want 400, got %d body=%s", resp.StatusCode, body)
			}
		})
	}
}
