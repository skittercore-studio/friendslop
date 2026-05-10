package game

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/skittercore-studio/friendslop/internal/db"
)

func TestRollCuratedPool_UniqueChars(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "g.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	roomID := mustSeedRoom(t, d, "lobby", "curated")
	for i := 0; i < 5; i++ {
		mustSeedPlayer(t, d, roomID, "p"+string(rune('A'+i)))
	}

	pool := []CharacterTemplate{
		{TemplateID: "t1", Name: "n1", Blurb: "b1"},
		{TemplateID: "t2", Name: "n2", Blurb: "b2"},
		{TemplateID: "t3", Name: "n3", Blurb: "b3"},
		{TemplateID: "t4", Name: "n4", Blurb: "b4"},
		{TemplateID: "t5", Name: "n5", Blurb: "b5"},
		{TemplateID: "t6", Name: "n6", Blurb: "b6"},
		{TemplateID: "t7", Name: "n7", Blurb: "b7"},
	}

	tx, _ := d.BeginTx(ctx, nil)
	ids, err := RollCuratedPool(ctx, tx, roomID, pool, 5)
	if err != nil {
		t.Fatalf("roll: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 5 {
		t.Fatalf("want 5, got %d", len(ids))
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate id %s", id)
		}
		seen[id] = true
	}
	// Verify rows exist and have template_id set.
	var cnt int
	d.QueryRowContext(ctx, `SELECT COUNT(*) FROM room_characters WHERE room_id = ? AND template_id IS NOT NULL`, roomID).Scan(&cnt)
	if cnt != 5 {
		t.Errorf("want 5 rows, got %d", cnt)
	}
}

func TestAssignCharacters_Bijective(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "g.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	roomID := mustSeedRoom(t, d, "lobby", "curated")
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

	tx, _ := d.BeginTx(ctx, nil)
	assigned, err := AssignCharacters(ctx, tx, playerIDs, charIDs)
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(assigned) != 4 {
		t.Fatalf("want 4, got %d", len(assigned))
	}
	chars := map[string]int{}
	for _, c := range assigned {
		chars[c]++
	}
	for c, n := range chars {
		if n != 1 {
			t.Errorf("character %s used %d times, want 1", c, n)
		}
	}
	// Each player has a character_id row in players table.
	for _, pid := range playerIDs {
		var cid string
		if err := d.QueryRowContext(ctx, `SELECT character_id FROM players WHERE id = ?`, pid).Scan(&cid); err != nil {
			t.Fatalf("player %s: %v", pid, err)
		}
		if cid == "" {
			t.Errorf("player %s missing character_id", pid)
		}
	}
}

func TestAssignAuthoredCharacters_PlaysOwn(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "g.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	roomID := mustSeedRoom(t, d, "charcreate", "playerwritten")
	playerIDs := []string{
		mustSeedPlayer(t, d, roomID, "alice"),
		mustSeedPlayer(t, d, roomID, "bob"),
		mustSeedPlayer(t, d, roomID, "carol"),
		mustSeedPlayer(t, d, roomID, "dave"),
	}
	// Each player authors exactly one character.
	authored := make(map[string]string, len(playerIDs))
	for _, pid := range playerIDs {
		cid := mustSeedAuthoredCharacter(t, d, roomID, pid)
		authored[pid] = cid
	}

	tx, _ := d.BeginTx(ctx, nil)
	got, err := AssignAuthoredCharacters(ctx, tx, roomID)
	if err != nil {
		t.Fatalf("assign authored: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(authored) {
		t.Fatalf("want %d pairs, got %d", len(authored), len(got))
	}
	// Every player must be assigned the character they themselves authored.
	for pid, wantCID := range authored {
		if gotCID, ok := got[pid]; !ok {
			t.Errorf("player %s missing from assignment", pid)
		} else if gotCID != wantCID {
			t.Errorf("player %s: assigned %s, want authored %s", pid, gotCID, wantCID)
		}
	}
	// players.character_id must be persisted to match.
	for pid, wantCID := range authored {
		var cid string
		if err := d.QueryRowContext(ctx, `SELECT character_id FROM players WHERE id = ?`, pid).Scan(&cid); err != nil {
			t.Fatalf("player %s: %v", pid, err)
		}
		if cid != wantCID {
			t.Errorf("player %s persisted character_id %s, want %s", pid, cid, wantCID)
		}
	}
}

// ---- helpers ----------------------------------------------------------------

func mustSeedRoom(t *testing.T, d *sql.DB, state, poolSrc string) string {
	t.Helper()
	id := uuid.NewString()
	code := uuid.NewString()[:4]
	now := time.Now().UnixMilli()
	if _, err := d.Exec(`INSERT INTO rooms (
		id, code, state, mode, pool_source, created_at, last_activity_at
	) VALUES (?, ?, ?, 'live', ?, ?, ?)`, id, code, state, poolSrc, now, now); err != nil {
		t.Fatalf("seed room: %v", err)
	}
	return id
}

func mustSeedPlayer(t *testing.T, d *sql.DB, roomID, name string) string {
	t.Helper()
	id := uuid.NewString()
	tok := uuid.NewString()
	now := time.Now().UnixMilli()
	if _, err := d.Exec(`INSERT INTO players (
		id, room_id, name, session_token, is_host, joined_at
	) VALUES (?, ?, ?, ?, 0, ?)`, id, roomID, name, tok, now); err != nil {
		t.Fatalf("seed player: %v", err)
	}
	return id
}

func mustSeedRoomCharacter(t *testing.T, d *sql.DB, roomID, templateID string) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := d.Exec(`INSERT INTO room_characters (
		id, room_id, template_id, author_player_id, name, blurb
	) VALUES (?, ?, ?, NULL, 'name', 'blurb')`, id, roomID, templateID); err != nil {
		t.Fatalf("seed room_character: %v", err)
	}
	return id
}

func mustSeedAuthoredCharacter(t *testing.T, d *sql.DB, roomID, authorPlayerID string) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := d.Exec(`INSERT INTO room_characters (
		id, room_id, template_id, author_player_id, name, blurb
	) VALUES (?, ?, NULL, ?, 'name', 'blurb')`, id, roomID, authorPlayerID); err != nil {
		t.Fatalf("seed authored room_character: %v", err)
	}
	return id
}
