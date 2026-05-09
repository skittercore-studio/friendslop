package game

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/skittercore-studio/friendslop/internal/db"
	"github.com/skittercore-studio/friendslop/internal/events"
)

// inMemoryPub captures events for tests inside this package.
type inMemoryPub struct {
	mu     sync.Mutex
	events []events.Event
}

func (p *inMemoryPub) Publish(_ string, ev events.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, ev)
}

func (p *inMemoryPub) names() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.events))
	for i, e := range p.events {
		out[i] = e.Name
	}
	return out
}

// TestTickAdvancesExpiredAnsweringRound seeds a room mid-answer with a
// past-expired deadline and verifies one Tick advances it to guessing.
func TestTickAdvancesExpiredAnsweringRound(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	roomID := mustSeedRoom(t, d, "answering", "curated")
	playerIDs := []string{
		mustSeedPlayer(t, d, roomID, "alice"),
		mustSeedPlayer(t, d, roomID, "bob"),
		mustSeedPlayer(t, d, roomID, "carol"),
		mustSeedPlayer(t, d, roomID, "dave"),
	}
	charIDs := []string{
		mustSeedRoomCharacter(t, d, roomID, "tA"),
		mustSeedRoomCharacter(t, d, roomID, "tB"),
		mustSeedRoomCharacter(t, d, roomID, "tC"),
		mustSeedRoomCharacter(t, d, roomID, "tD"),
	}
	for i, pid := range playerIDs {
		if _, err := d.Exec(`UPDATE players SET character_id = ? WHERE id = ?`, charIDs[i], pid); err != nil {
			t.Fatalf("set char: %v", err)
		}
	}
	if _, err := d.Exec(`UPDATE rooms SET round_number = 1 WHERE id = ?`, roomID); err != nil {
		t.Fatal(err)
	}
	// Round with deadline in the past.
	roundID := uuid.NewString()
	past := time.Now().Add(-1 * time.Minute).UnixMilli()
	if _, err := d.Exec(`INSERT INTO rounds (id, room_id, number, question_text, state, answer_deadline, started_at)
		VALUES (?, ?, 1, 'q?', 'answering', ?, ?)`, roundID, roomID, past, past); err != nil {
		t.Fatalf("seed round: %v", err)
	}

	pub := &inMemoryPub{}
	content, err := LoadDefaultContent()
	if err != nil {
		t.Fatal(err)
	}
	eng := NewEngine(d, pub, content)

	if err := eng.Tick(ctx, time.Now()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	// Round should now be guessing.
	var rstate, ostate string
	d.QueryRow(`SELECT state FROM rounds WHERE id = ?`, roundID).Scan(&rstate)
	d.QueryRow(`SELECT state FROM rooms WHERE id = ?`, roomID).Scan(&ostate)
	if rstate != "guessing" {
		t.Errorf("round state = %s, want guessing", rstate)
	}
	if ostate != "guessing" {
		t.Errorf("room state = %s, want guessing", ostate)
	}
	gotEvents := pub.names()
	wantOne := func(name string) {
		for _, n := range gotEvents {
			if n == name {
				return
			}
		}
		t.Errorf("missing event %q in %v", name, gotEvents)
	}
	wantOne("state.changed")
	wantOne("round.answers_revealed")

	// All four players should now have an answer (auto-filled).
	var cnt int
	d.QueryRow(`SELECT COUNT(*) FROM answers WHERE round_id = ?`, roundID).Scan(&cnt)
	if cnt != 4 {
		t.Errorf("expected 4 auto-filled answers, got %d", cnt)
	}
}
