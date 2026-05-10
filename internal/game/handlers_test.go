package game_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/skittercore-studio/friendslop/internal/db"
	"github.com/skittercore-studio/friendslop/internal/events"
	"github.com/skittercore-studio/friendslop/internal/game"
	"github.com/skittercore-studio/friendslop/internal/server"
)

// recordingPub captures every event for assertions.
type recordingPub struct {
	mu     sync.Mutex
	events []capturedEvent
}

type capturedEvent struct {
	RoomID string
	Name   string
	Data   any
}

func (p *recordingPub) Publish(roomID string, ev events.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, capturedEvent{RoomID: roomID, Name: ev.Name, Data: ev.Data})
}

func (p *recordingPub) names() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.events))
	for i, e := range p.events {
		out[i] = e.Name
	}
	return out
}

func (p *recordingPub) findFirst(name string) (capturedEvent, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.events {
		if e.Name == name {
			return e, true
		}
	}
	return capturedEvent{}, false
}

// gameTestServer brings up a chi router with the rooms handler + the gamelogic
// package wired in. The timer goroutine is created with a context that cancels
// at test cleanup.
func gameTestServer(t *testing.T) (*httptest.Server, *recordingPub) {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(context.Background(), filepath.Join(dir, "h.db"))
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	pub := &recordingPub{}
	s := server.NewServer(d, server.WithPublisher(pub))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := game.Setup(ctx, s); err != nil {
		t.Fatalf("setup: %v", err)
	}
	r := chi.NewRouter()
	s.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, pub
}

func newJar() *cookiejar.Jar {
	j, _ := cookiejar.New(nil)
	return j
}

func mustPost(t *testing.T, c *http.Client, url string, body any) (*http.Response, []byte) {
	t.Helper()
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, out
}

func mustGet(t *testing.T, c *http.Client, url string) (*http.Response, []byte) {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, out
}

type joinedClient struct {
	c        *http.Client
	playerID string
	token    string
}

func createRoom(t *testing.T, srv *httptest.Server, hostName, mode, poolSrc string) (string, *joinedClient) {
	t.Helper()
	jc := &joinedClient{c: &http.Client{Jar: newJar()}}
	resp, body := mustPost(t, jc.c, srv.URL+"/api/v1/rooms", map[string]any{
		"host_name":   hostName,
		"mode":        mode,
		"pool_source": poolSrc,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %s/%s: %d %s", mode, poolSrc, resp.StatusCode, body)
	}
	var v struct {
		RoomCode string `json:"room_code"`
		PlayerID string `json:"player_id"`
		Token    string `json:"session_token"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("create unmarshal: %v", err)
	}
	jc.playerID = v.PlayerID
	jc.token = v.Token
	return v.RoomCode, jc
}

func joinRoom(t *testing.T, srv *httptest.Server, code, name string) *joinedClient {
	t.Helper()
	jc := &joinedClient{c: &http.Client{Jar: newJar()}}
	resp, body := mustPost(t, jc.c, srv.URL+"/api/v1/rooms/"+code+"/join", map[string]any{"name": name})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("join %s: %d %s", name, resp.StatusCode, body)
	}
	var v struct {
		PlayerID string `json:"player_id"`
		Token    string `json:"session_token"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("join unmarshal: %v", err)
	}
	jc.playerID = v.PlayerID
	jc.token = v.Token
	return jc
}

// publicSnapshot returns the current public snapshot.
type publicSnap struct {
	State        string `json:"state"`
	RoundNumber  int    `json:"round_number"`
	Characters   []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Blurb string `json:"blurb"`
	} `json:"characters"`
	CurrentRound *struct {
		Number       int    `json:"number"`
		State        string `json:"state"`
		QuestionText string `json:"question_text"`
	} `json:"current_round"`
	WinnerPlayerID *string `json:"winner_player_id"`
}

func getSnap(t *testing.T, srv *httptest.Server, code string) publicSnap {
	t.Helper()
	resp, body := mustGet(t, &http.Client{}, srv.URL+"/api/v1/rooms/"+code)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("snap: %d %s", resp.StatusCode, body)
	}
	var s publicSnap
	if err := json.Unmarshal(body, &s); err != nil {
		t.Fatalf("snap unmarshal: %v body=%s", err, body)
	}
	return s
}

type meView struct {
	PlayerID            string  `json:"player_id"`
	YourCharacter       *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Blurb string `json:"blurb"`
	} `json:"your_character"`
	YourAuthoredCharID *string `json:"your_authored_character_id,omitempty"`
}

func getMe(t *testing.T, srv *httptest.Server, code string, jc *joinedClient) meView {
	t.Helper()
	resp, body := mustGet(t, jc.c, srv.URL+"/api/v1/rooms/"+code+"/me")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/me: %d %s", resp.StatusCode, body)
	}
	var v meView
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("/me unmarshal: %v body=%s", err, body)
	}
	return v
}

// TestStartRequiresMinPlayers — host can't start below MinPlayers (host alone < MinPlayers).
func TestStartRequiresMinPlayers(t *testing.T) {
	srv, _ := gameTestServer(t)
	code, host := createRoom(t, srv, "vex", "live", "curated")
	resp, body := mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/start", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, body)
	}
}

// TestStartRequiresHost — non-host gets 403.
func TestStartRequiresHost(t *testing.T) {
	srv, _ := gameTestServer(t)
	code, _ := createRoom(t, srv, "vex", "live", "curated")
	for _, n := range []string{"a", "b", "c"} {
		joinRoom(t, srv, code, n)
	}
	other := joinRoom(t, srv, code, "rival")
	resp, _ := mustPost(t, other.c, srv.URL+"/api/v1/rooms/"+code+"/start", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

// TestCuratedHappyPath — full live game with curated pool, 4 players,
// rigged to produce a winner in round 1.
func TestCuratedHappyPath(t *testing.T) {
	srv, pub := gameTestServer(t)
	code, host := createRoom(t, srv, "vex", "async", "curated")

	// Three more players for total 4.
	others := []*joinedClient{
		joinRoom(t, srv, code, "alice"),
		joinRoom(t, srv, code, "bob"),
		joinRoom(t, srv, code, "carol"),
	}

	// Host starts.
	resp, body := mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/start", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("start: %d %s", resp.StatusCode, body)
	}

	snap := getSnap(t, srv, code)
	if snap.State != "answering" || snap.RoundNumber != 1 {
		t.Fatalf("after start: state=%s round=%d", snap.State, snap.RoundNumber)
	}
	if len(snap.Characters) != 4 {
		t.Fatalf("expected 4 characters revealed, got %d", len(snap.Characters))
	}

	// Each player submits an answer for round 1.
	all := append([]*joinedClient{host}, others...)
	for i, p := range all {
		resp, body := mustPost(t, p.c, srv.URL+"/api/v1/rooms/"+code+"/answer", map[string]any{
			"round_number": 1,
			"text":         "answer from player " + p.playerID,
		})
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("answer p%d: %d %s", i, resp.StatusCode, body)
		}
	}

	// Now in guessing.
	snap = getSnap(t, srv, code)
	if snap.State != "guessing" {
		t.Fatalf("expected guessing, got %s", snap.State)
	}

	// Discover everyone's true assignment via /me.
	trueAssign := map[string]string{}
	for _, p := range all {
		me := getMe(t, srv, code, p)
		if me.YourCharacter == nil {
			t.Fatalf("player %s has no character", p.playerID)
		}
		trueAssign[p.playerID] = me.YourCharacter.ID
	}

	// Host submits a perfect guess for the other 3 — should win.
	hostMapping := map[string]string{}
	for pid, cid := range trueAssign {
		if pid == host.playerID {
			continue
		}
		hostMapping[pid] = cid
	}
	resp, body = mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/guess", map[string]any{
		"round_number": 1,
		"mapping":      hostMapping,
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("host guess: %d %s", resp.StatusCode, body)
	}

	// Other players submit deliberately wrong guesses.
	for _, p := range others {
		mapping := map[string]string{}
		// Rotate true assignments by one — guarantees no perfect match.
		var others []string
		for pid := range trueAssign {
			if pid == p.playerID {
				continue
			}
			others = append(others, pid)
		}
		// Use the pool minus this player's own char.
		var poolMinus []string
		for _, cid := range trueAssign {
			poolMinus = append(poolMinus, cid)
		}
		// Build mapping: assign each other player a different player's true char
		// (rotated by 1 → guaranteed all-wrong if N>2).
		for i, pid := range others {
			cid := trueAssign[others[(i+1)%len(others)]]
			if cid == trueAssign[pid] {
				// Fallback shouldn't happen for N=3 but stay safe.
				cid = trueAssign[host.playerID]
			}
			mapping[pid] = cid
		}
		resp, body := mustPost(t, p.c, srv.URL+"/api/v1/rooms/"+code+"/guess", map[string]any{
			"round_number": 1,
			"mapping":      mapping,
		})
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("guess p=%s: %d %s", p.playerID, resp.StatusCode, body)
		}
	}

	// Game should be won.
	snap = getSnap(t, srv, code)
	if snap.State != "won" {
		t.Fatalf("expected won, got %s", snap.State)
	}
	if snap.WinnerPlayerID == nil || *snap.WinnerPlayerID != host.playerID {
		t.Fatalf("expected winner=%s, got %v", host.playerID, snap.WinnerPlayerID)
	}

	// Events emitted.
	expected := []string{"state.changed", "round.started", "answer.submitted", "round.answers_revealed", "guess.submitted", "round.scored", "game.won"}
	got := pub.names()
	for _, name := range expected {
		found := false
		for _, g := range got {
			if g == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing event %q (got %v)", name, got)
		}
	}
}

// TestPlayerWrittenFlow — playerwritten pool source goes through CHARCREATE.
func TestPlayerWrittenFlow(t *testing.T) {
	srv, pub := gameTestServer(t)
	code, host := createRoom(t, srv, "vex", "async", "playerwritten")
	others := []*joinedClient{
		joinRoom(t, srv, code, "alice"),
		joinRoom(t, srv, code, "bob"),
		joinRoom(t, srv, code, "carol"),
	}
	all := append([]*joinedClient{host}, others...)

	resp, body := mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/start", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("start: %d %s", resp.StatusCode, body)
	}
	snap := getSnap(t, srv, code)
	if snap.State != "charcreate" {
		t.Fatalf("expected charcreate, got %s", snap.State)
	}

	// Reject too-short blurb.
	resp, _ = mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/character",
		map[string]any{"name": "X", "blurb": "short"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on short blurb, got %d", resp.StatusCode)
	}

	// Each player submits a character.
	for i, p := range all {
		resp, body := mustPost(t, p.c, srv.URL+"/api/v1/rooms/"+code+"/character", map[string]any{
			"name":  "Char " + p.playerID[:4],
			"blurb": "A character with enough blurb to pass validation rule for player " + string(rune('A'+i)),
		})
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("character p%d: %d %s", i, resp.StatusCode, body)
		}
	}

	// Second submit by same player: 409.
	resp, _ = mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/character",
		map[string]any{"name": "Y", "blurb": "Another long-enough blurb 12345 12345"})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate submit, got %d", resp.StatusCode)
	}

	// Should have advanced to answering.
	snap = getSnap(t, srv, code)
	if snap.State != "answering" {
		t.Fatalf("expected answering after all chars submitted, got %s", snap.State)
	}
	if len(snap.Characters) != 4 {
		t.Fatalf("expected 4 characters revealed, got %d", len(snap.Characters))
	}

	// /me for an author should expose your_authored_character_id.
	me := getMe(t, srv, code, host)
	if me.YourAuthoredCharID == nil {
		t.Errorf("expected your_authored_character_id on /me for author")
	}

	// Events: charcreate.started, charcreate.submitted (multiple), charcreate.completed.
	if _, ok := pub.findFirst("charcreate.started"); !ok {
		t.Error("missing charcreate.started")
	}
	if _, ok := pub.findFirst("charcreate.completed"); !ok {
		t.Error("missing charcreate.completed")
	}
}

// TestAbandonHostOnly — non-host cannot abandon.
func TestAbandonHostOnly(t *testing.T) {
	srv, _ := gameTestServer(t)
	code, _ := createRoom(t, srv, "vex", "live", "curated")
	other := joinRoom(t, srv, code, "rival")
	resp, _ := mustPost(t, other.c, srv.URL+"/api/v1/rooms/"+code+"/abandon", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

// TestAbandonHost — host abandon transitions to abandoned, fires event.
func TestAbandonHost(t *testing.T) {
	srv, pub := gameTestServer(t)
	code, host := createRoom(t, srv, "vex", "live", "curated")
	resp, _ := mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/abandon", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	snap := getSnap(t, srv, code)
	if snap.State != "abandoned" {
		t.Fatalf("expected abandoned, got %s", snap.State)
	}
	if _, ok := pub.findFirst("game.abandoned"); !ok {
		t.Error("missing game.abandoned event")
	}
}

// TestStaleRoundNumberAnswer — answer with wrong round_number is 409.
func TestStaleRoundNumberAnswer(t *testing.T) {
	srv, _ := gameTestServer(t)
	code, host := createRoom(t, srv, "vex", "async", "curated")
	for _, n := range []string{"a", "b", "c"} {
		joinRoom(t, srv, code, n)
	}
	if resp, _ := mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/start", nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("start: %d", resp.StatusCode)
	}
	resp, body := mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/answer", map[string]any{
		"round_number": 99,
		"text":         "stale",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", resp.StatusCode, body)
	}
}

// TestGuessRejectsSelfMapping — mapping containing the guesser is 400.
func TestGuessRejectsSelfMapping(t *testing.T) {
	srv, _ := gameTestServer(t)
	code, host := createRoom(t, srv, "vex", "async", "curated")
	others := []*joinedClient{
		joinRoom(t, srv, code, "alice"),
		joinRoom(t, srv, code, "bob"),
		joinRoom(t, srv, code, "carol"),
	}
	if resp, _ := mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/start", nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("start: %d", resp.StatusCode)
	}
	all := append([]*joinedClient{host}, others...)
	for _, p := range all {
		mustPost(t, p.c, srv.URL+"/api/v1/rooms/"+code+"/answer", map[string]any{
			"round_number": 1,
			"text":         "an answer of substantial length",
		})
	}
	// In guessing now. Host submits a mapping that includes themselves.
	mapping := map[string]string{
		host.playerID: "any-c",
	}
	resp, body := mustPost(t, host.c, srv.URL+"/api/v1/rooms/"+code+"/guess",
		map[string]any{"round_number": 1, "mapping": mapping})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, body)
	}
}
