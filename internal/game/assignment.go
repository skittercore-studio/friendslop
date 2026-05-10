// assignment.go — character pool roll & 1:1 player→character assignment.
//
// Two pool sources, both end the same way:
//   curated:        pick N random templates from the default pool, persist as
//                   room_characters with template_id set, author_player_id NULL.
//   playerwritten:  the room_characters rows already exist (one per author).
//                   No insertion needed — just shuffle them and assign 1:1.
//
// In both cases, after rolling the pool we shuffle the player order and the
// character order, then zip them: players[i] -> characters[i]. This keeps the
// assignment uniform-random over permutations.
package game

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

// RollCuratedPool selects N templates from the global pool uniformly at
// random without replacement, inserts them as room_characters, and returns
// the inserted character IDs in shuffled order.
//
// Caller is responsible for owning the surrounding transaction so the pool
// roll, the assignment, and the room state transition all commit atomically.
func RollCuratedPool(ctx context.Context, tx *sql.Tx, roomID string, pool []CharacterTemplate, n int) ([]string, error) {
	if n <= 0 {
		return nil, fmt.Errorf("pool size must be > 0")
	}
	if n > len(pool) {
		return nil, fmt.Errorf("requested pool size %d exceeds available templates (%d)", n, len(pool))
	}

	// Fisher-Yates partial shuffle: pick N distinct indices.
	indices := make([]int, len(pool))
	for i := range indices {
		indices[i] = i
	}
	for i := 0; i < n; i++ {
		j, err := randInt(len(indices) - i)
		if err != nil {
			return nil, err
		}
		j += i
		indices[i], indices[j] = indices[j], indices[i]
	}

	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		t := pool[indices[i]]
		id := uuid.NewString()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO room_characters (id, room_id, template_id, author_player_id, name, blurb)
			VALUES (?, ?, ?, NULL, ?, ?)
		`, id, roomID, t.TemplateID, t.Name, t.Blurb); err != nil {
			return nil, fmt.Errorf("insert room_characters: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// AssignCharacters performs a uniform-random 1:1 assignment of characters to
// players, persisting the result by setting players.character_id.
//
// playerIDs and characterIDs MUST be the same length. Returns the resulting
// assignment map for convenience (player_id -> character_id).
func AssignCharacters(ctx context.Context, tx *sql.Tx, playerIDs []string, characterIDs []string) (map[string]string, error) {
	if len(playerIDs) != len(characterIDs) {
		return nil, fmt.Errorf("player count %d != character count %d", len(playerIDs), len(characterIDs))
	}
	if len(playerIDs) == 0 {
		return nil, errors.New("no players to assign")
	}

	// Shuffle a copy of characterIDs (Fisher-Yates).
	shuffled := make([]string, len(characterIDs))
	copy(shuffled, characterIDs)
	for i := len(shuffled) - 1; i > 0; i-- {
		j, err := randInt(i + 1)
		if err != nil {
			return nil, err
		}
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	out := make(map[string]string, len(playerIDs))
	for i, pid := range playerIDs {
		cid := shuffled[i]
		if _, err := tx.ExecContext(ctx, `
			UPDATE players SET character_id = ? WHERE id = ?
		`, cid, pid); err != nil {
			return nil, fmt.Errorf("assign player %s: %w", pid, err)
		}
		out[pid] = cid
	}
	return out, nil
}

// AssignAuthoredCharacters performs the deterministic playerwritten assignment:
// every player is dealt the character they themselves authored. No shuffle,
// no derangement — the mechanic is "you write a character, you perform it,
// friends try to recognise your voice." The shuffle in AssignCharacters is
// reserved for curated mode where there's no authorship to honour.
//
// Operates on the players table directly via a JOIN: for every live player in
// the room, set character_id to the room_characters row where
// author_player_id = player.id. Returns the resulting map for callers that
// want to inspect the assignment.
func AssignAuthoredCharacters(ctx context.Context, tx *sql.Tx, roomID string) (map[string]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT p.id, c.id
		FROM players p
		JOIN room_characters c
		  ON c.room_id = p.room_id AND c.author_player_id = p.id
		WHERE p.room_id = ? AND p.left_at IS NULL
	`, roomID)
	if err != nil {
		return nil, fmt.Errorf("collect authored pairs: %w", err)
	}
	defer rows.Close()

	pairs := make(map[string]string)
	for rows.Next() {
		var pid, cid string
		if err := rows.Scan(&pid, &cid); err != nil {
			return nil, err
		}
		pairs[pid] = cid
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, errors.New("no authored characters to assign")
	}

	for pid, cid := range pairs {
		if _, err := tx.ExecContext(ctx, `
			UPDATE players SET character_id = ? WHERE id = ?
		`, cid, pid); err != nil {
			return nil, fmt.Errorf("assign player %s: %w", pid, err)
		}
	}
	return pairs, nil
}

// PlayerWrittenCharacterIDs reads back the room_characters that authors
// inserted during CHARCREATE, in stable id order. Used to feed
// AssignCharacters once all authors have submitted.
func PlayerWrittenCharacterIDs(ctx context.Context, tx *sql.Tx, roomID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM room_characters
		WHERE room_id = ? AND author_player_id IS NOT NULL
		ORDER BY id ASC
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LivePlayerIDs returns the IDs of all not-left players in a room, in
// joined_at order.
func LivePlayerIDs(ctx context.Context, tx *sql.Tx, roomID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM players
		WHERE room_id = ? AND left_at IS NULL
		ORDER BY joined_at ASC
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// randInt returns a uniform random int in [0, n) using crypto/rand.
func randInt(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("randInt: n must be > 0")
	}
	bi, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(bi.Int64()), nil
}
