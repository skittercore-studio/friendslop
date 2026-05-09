package session_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/skittercore-studio/friendslop/internal/db"
	"github.com/skittercore-studio/friendslop/internal/session"
)

func TestNewTokenIsRandomAndUrlSafe(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		tok, err := session.NewToken()
		if err != nil {
			t.Fatalf("NewToken: %v", err)
		}
		if len(tok) < 40 {
			t.Fatalf("token too short: %q", tok)
		}
		for _, r := range tok {
			ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
				(r >= '0' && r <= '9') || r == '-' || r == '_'
			if !ok {
				t.Fatalf("non-url-safe char %q in token %q", r, tok)
			}
		}
		if seen[tok] {
			t.Fatalf("duplicate token after %d iterations", i)
		}
		seen[tok] = true
	}
}

// Round-trip: a player row with a token must resolve back through Resolver.
func TestResolverRoundTrip(t *testing.T) {
	dir := t.TempDir()
	d, err := db.Open(context.Background(), filepath.Join(dir, "s.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer d.Close()

	if _, err := d.Exec(`INSERT INTO rooms (
		id, code, state, mode, pool_source, created_at, last_activity_at
	) VALUES ('r1','BRSK','lobby','live','curated',1,1)`); err != nil {
		t.Fatalf("seed room: %v", err)
	}
	tok, err := session.NewToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO players (
		id, room_id, name, session_token, is_host, joined_at
	) VALUES ('p1','r1','vex',?,1,1)`, tok); err != nil {
		t.Fatalf("seed player: %v", err)
	}

	rs := session.NewResolver(d)
	s, err := rs.Resolve(context.Background(), tok)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if s.PlayerID != "p1" || s.RoomID != "r1" || s.RoomCode != "BRSK" || !s.IsHost {
		t.Fatalf("unexpected session: %+v", s)
	}

	// Stale (left) cookie returns ErrNoSession.
	if _, err := d.Exec(`UPDATE players SET left_at = 2 WHERE id = 'p1'`); err != nil {
		t.Fatalf("mark left: %v", err)
	}
	if _, err := rs.Resolve(context.Background(), tok); err == nil {
		t.Fatalf("expected ErrNoSession after leave")
	}
}
