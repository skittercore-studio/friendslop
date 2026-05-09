// rounds.go — round lifecycle helpers.
//
// OpenRound: pick a question (random from bank, avoiding repeats already used
// in this room when possible), insert a rounds row in `answering` state with
// an answer_deadline computed from the room's mode + timeout.
//
// CloseAnswering: auto-fill missing answers as "[no answer]", flip the round
// to `guessing`, and stamp guess_deadline. Caller is responsible for emitting
// the round.answers_revealed event with character-attributed payload.
//
// CloseGuessing: score every guess against the true assignment, write
// correct_count, detect a winner, flip the round to `done`. Caller emits
// round.scored and either game.won or round.started.
//
// All entry points expect to be wrapped in a transaction by the caller, and
// take *sql.Tx so the work composes with surrounding state changes.
package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/skittercore-studio/friendslop/internal/events"
)

// NoAnswerSentinel is the auto-filled text for missing answers when a round
// closes by timeout or all-submitted with a player who never answered (e.g.
// dropped). Matches SPEC.md §2.
const NoAnswerSentinel = "[no answer]"

// RoomConfig is the subset of the rooms row needed for round lifecycle
// decisions.
type RoomConfig struct {
	ID                     string
	Mode                   string // live|async
	AnswerTimeoutSeconds   sql.NullInt64
	GuessTimeoutSeconds    sql.NullInt64
	InterRoundPauseSeconds int
	QuestionBank           string
}

// LoadRoomConfig fetches the per-round configuration for a room.
func LoadRoomConfig(ctx context.Context, tx *sql.Tx, roomID string) (*RoomConfig, error) {
	c := &RoomConfig{ID: roomID}
	if err := tx.QueryRowContext(ctx, `
		SELECT mode, answer_timeout_seconds, guess_timeout_seconds,
		       inter_round_pause_seconds, question_bank
		FROM rooms WHERE id = ?
	`, roomID).Scan(&c.Mode, &c.AnswerTimeoutSeconds, &c.GuessTimeoutSeconds,
		&c.InterRoundPauseSeconds, &c.QuestionBank); err != nil {
		return nil, err
	}
	return c, nil
}

// pickQuestion chooses a question from `bank` avoiding texts already used in
// `room`. Falls back to any question if the room has cycled through the
// whole bank.
func pickQuestion(ctx context.Context, tx *sql.Tx, roomID string, bank []QuestionTemplate) (QuestionTemplate, error) {
	if len(bank) == 0 {
		return QuestionTemplate{}, errors.New("question bank is empty")
	}
	used := map[string]struct{}{}
	rows, err := tx.QueryContext(ctx, `SELECT question_text FROM rounds WHERE room_id = ?`, roomID)
	if err != nil {
		return QuestionTemplate{}, err
	}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return QuestionTemplate{}, err
		}
		used[t] = struct{}{}
	}
	rows.Close()

	candidates := make([]QuestionTemplate, 0, len(bank))
	for _, q := range bank {
		if _, seen := used[q.Text]; !seen {
			candidates = append(candidates, q)
		}
	}
	if len(candidates) == 0 {
		candidates = bank
	}
	idx, err := randInt(len(candidates))
	if err != nil {
		return QuestionTemplate{}, err
	}
	return candidates[idx], nil
}

// OpenRound creates the next round row for `roomID`. It bumps the rooms.round_number
// and rooms.state to 'answering', stamps started_at, and computes
// answer_deadline based on the room mode + answer_timeout_seconds.
//
// Returns (roundID, roundNumber, questionText, answerDeadline) for use by the
// caller to emit round.started.
func OpenRound(ctx context.Context, tx *sql.Tx, roomID string, bank []QuestionTemplate, now time.Time) (string, int, string, *int64, error) {
	cfg, err := LoadRoomConfig(ctx, tx, roomID)
	if err != nil {
		return "", 0, "", nil, fmt.Errorf("load room: %w", err)
	}

	q, err := pickQuestion(ctx, tx, roomID, bank)
	if err != nil {
		return "", 0, "", nil, err
	}

	var nextNumber int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(number), 0) + 1 FROM rounds WHERE room_id = ?`, roomID,
	).Scan(&nextNumber); err != nil {
		return "", 0, "", nil, err
	}

	deadline := answerDeadlineFor(cfg, now)
	roundID := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO rounds (id, room_id, number, question_text, state, answer_deadline, started_at)
		VALUES (?, ?, ?, ?, 'answering', ?, ?)
	`, roundID, roomID, nextNumber, q.Text, nullableInt64(deadline), now.UnixMilli()); err != nil {
		return "", 0, "", nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE rooms SET state = 'answering', round_number = ?, last_activity_at = ?, started_at = COALESCE(started_at, ?)
		WHERE id = ?
	`, nextNumber, now.UnixMilli(), now.UnixMilli(), roomID); err != nil {
		return "", 0, "", nil, err
	}
	return roundID, nextNumber, q.Text, deadline, nil
}

func answerDeadlineFor(cfg *RoomConfig, now time.Time) *int64 {
	if !cfg.AnswerTimeoutSeconds.Valid {
		return nil
	}
	d := now.Add(time.Duration(cfg.AnswerTimeoutSeconds.Int64) * time.Second).UnixMilli()
	return &d
}

func guessDeadlineFor(cfg *RoomConfig, now time.Time) *int64 {
	if !cfg.GuessTimeoutSeconds.Valid {
		return nil
	}
	d := now.Add(time.Duration(cfg.GuessTimeoutSeconds.Int64) * time.Second).UnixMilli()
	return &d
}

func nullableInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// RevealAnswer is one entry in the round.answers_revealed payload — character
// attribution only, no player_id, no submitted_at, sorted by character_id.
type RevealAnswer struct {
	CharacterID string `json:"character_id"`
	Text        string `json:"text"`
}

// CloseAnswering transitions a round from `answering` -> `guessing`.
//   - Fills missing answers with NoAnswerSentinel.
//   - Updates rounds.state, rounds.guess_deadline, rooms.state.
//   - Returns the reveal payload (sorted by character_id) and the new
//     guess_deadline.
//
// Caller is expected to publish round.answers_revealed and state.changed.
func CloseAnswering(ctx context.Context, tx *sql.Tx, roomID, roundID string, now time.Time) ([]RevealAnswer, *int64, error) {
	cfg, err := LoadRoomConfig(ctx, tx, roomID)
	if err != nil {
		return nil, nil, err
	}

	// Fill missing answers for live players.
	prows, err := tx.QueryContext(ctx, `
		SELECT id FROM players WHERE room_id = ? AND left_at IS NULL
	`, roomID)
	if err != nil {
		return nil, nil, err
	}
	var pids []string
	for prows.Next() {
		var pid string
		if err := prows.Scan(&pid); err != nil {
			prows.Close()
			return nil, nil, err
		}
		pids = append(pids, pid)
	}
	prows.Close()

	for _, pid := range pids {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO answers (round_id, player_id, text, submitted_at)
			VALUES (?, ?, ?, ?)
		`, roundID, pid, NoAnswerSentinel, now.UnixMilli()); err != nil {
			return nil, nil, err
		}
	}

	deadline := guessDeadlineFor(cfg, now)
	if _, err := tx.ExecContext(ctx, `
		UPDATE rounds SET state = 'guessing', guess_deadline = ?
		WHERE id = ?
	`, nullableInt64(deadline), roundID); err != nil {
		return nil, nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE rooms SET state = 'guessing', last_activity_at = ? WHERE id = ?
	`, now.UnixMilli(), roomID); err != nil {
		return nil, nil, err
	}

	// Build reveal payload sorted by character_id (SQL ORDER BY rc.id).
	arows, err := tx.QueryContext(ctx, `
		SELECT rc.id, a.text
		FROM answers a
		JOIN players p ON p.id = a.player_id
		JOIN room_characters rc ON rc.id = p.character_id
		WHERE a.round_id = ?
		ORDER BY rc.id ASC
	`, roundID)
	if err != nil {
		return nil, nil, err
	}
	var reveal []RevealAnswer
	for arows.Next() {
		var r RevealAnswer
		if err := arows.Scan(&r.CharacterID, &r.Text); err != nil {
			arows.Close()
			return nil, nil, err
		}
		reveal = append(reveal, r)
	}
	arows.Close()
	return reveal, deadline, nil
}

// ScoringResult is the outcome of CloseGuessing — public per-player counts,
// the winner if any, and the true assignment table (only used in the
// game.won payload).
type ScoringResult struct {
	RoundNumber     int
	PublicScores    map[string]int
	WinnerPlayerID  string // empty if no winner this round
	TrueAssignments []AssignmentEntry
	GameOver        bool
	NextRoundAt     *int64 // unix ms — when the next round will auto-start (nil if game over)
}

// AssignmentEntry is one row of the true {player -> character} mapping,
// emitted only inside game.won payloads at end of game.
type AssignmentEntry struct {
	PlayerID    string `json:"player_id"`
	CharacterID string `json:"character_id"`
}

// CloseGuessing transitions a round from `guessing` -> `scoring` -> `done`.
//
//   - Fills missing guesses as guesses with correct_count=0 (no entries).
//   - Scores each guess against the true assignment (excluding the guesser
//     themselves).
//   - Detects a winner: any guesser with correct_count == N-1 (a perfect
//     mapping over all other live players). On tie, earliest submitted_at
//     wins.
//   - On winner: room.state = 'won', winner_player_id set, ended_at stamped.
//   - On no winner: room remains in 'scoring' temporarily; caller schedules
//     the next round at now + inter_round_pause_seconds.
func CloseGuessing(ctx context.Context, tx *sql.Tx, roomID, roundID string, now time.Time) (*ScoringResult, error) {
	cfg, err := LoadRoomConfig(ctx, tx, roomID)
	if err != nil {
		return nil, err
	}
	var roundNumber int
	if err := tx.QueryRowContext(ctx,
		`SELECT number FROM rounds WHERE id = ?`, roundID,
	).Scan(&roundNumber); err != nil {
		return nil, err
	}

	// Pull the true assignment over live players.
	arows, err := tx.QueryContext(ctx, `
		SELECT id, character_id FROM players
		WHERE room_id = ? AND left_at IS NULL AND character_id IS NOT NULL
		ORDER BY joined_at ASC
	`, roomID)
	if err != nil {
		return nil, err
	}
	assignment := map[string]string{}
	var assignList []AssignmentEntry
	for arows.Next() {
		var pid, cid string
		if err := arows.Scan(&pid, &cid); err != nil {
			arows.Close()
			return nil, err
		}
		assignment[pid] = cid
		assignList = append(assignList, AssignmentEntry{PlayerID: pid, CharacterID: cid})
	}
	arows.Close()

	// Auto-fill missing guesses (zero-score placeholder rows so the scoreboard
	// shows a row per player). We add an empty guess: no entries, count 0.
	for pid := range assignment {
		var has int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM guesses WHERE round_id = ? AND guesser_player_id = ?`,
			roundID, pid,
		).Scan(&has); err != nil {
			return nil, err
		}
		if has == 0 {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO guesses (id, round_id, guesser_player_id, correct_count, submitted_at)
				VALUES (?, ?, ?, 0, ?)
			`, uuid.NewString(), roundID, pid, now.UnixMilli()); err != nil {
				return nil, err
			}
		}
	}

	// Score every guess.
	grows, err := tx.QueryContext(ctx, `
		SELECT id, guesser_player_id, submitted_at
		FROM guesses WHERE round_id = ?
		ORDER BY submitted_at ASC, id ASC
	`, roundID)
	if err != nil {
		return nil, err
	}
	type guessRow struct {
		id, guesser string
		submitted   int64
	}
	var gs []guessRow
	for grows.Next() {
		var g guessRow
		if err := grows.Scan(&g.id, &g.guesser, &g.submitted); err != nil {
			grows.Close()
			return nil, err
		}
		gs = append(gs, g)
	}
	grows.Close()

	publicScores := map[string]int{}
	var winner string
	var winnerSubmitted int64

	for _, g := range gs {
		// Load entries for this guess.
		erows, err := tx.QueryContext(ctx,
			`SELECT target_player_id, character_id FROM guess_entries WHERE guess_id = ?`,
			g.id)
		if err != nil {
			return nil, err
		}
		mapping := map[string]string{}
		for erows.Next() {
			var t, c string
			if err := erows.Scan(&t, &c); err != nil {
				erows.Close()
				return nil, err
			}
			mapping[t] = c
		}
		erows.Close()

		count, err := ScoreGuess(g.guesser, mapping, assignment)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE guesses SET correct_count = ? WHERE id = ?`, count, g.id,
		); err != nil {
			return nil, err
		}
		publicScores[g.guesser] = count

		// Winner detection: perfect over all OTHER live players, earliest submit wins.
		if IsPerfectGuess(g.guesser, mapping, assignment) {
			if winner == "" || g.submitted < winnerSubmitted {
				winner = g.guesser
				winnerSubmitted = g.submitted
			}
		}
	}

	// Mark round done.
	if _, err := tx.ExecContext(ctx, `
		UPDATE rounds SET state = 'done', closed_at = ? WHERE id = ?
	`, now.UnixMilli(), roundID); err != nil {
		return nil, err
	}

	res := &ScoringResult{
		RoundNumber:     roundNumber,
		PublicScores:    publicScores,
		TrueAssignments: assignList,
	}
	if winner != "" {
		res.WinnerPlayerID = winner
		res.GameOver = true
		if _, err := tx.ExecContext(ctx, `
			UPDATE rooms SET state = 'won', winner_player_id = ?, ended_at = ?, last_activity_at = ?
			WHERE id = ?
		`, winner, now.UnixMilli(), now.UnixMilli(), roomID); err != nil {
			return nil, err
		}
	} else {
		// Park in 'scoring' until the timer kicks the next round.
		if _, err := tx.ExecContext(ctx, `
			UPDATE rooms SET state = 'scoring', last_activity_at = ? WHERE id = ?
		`, now.UnixMilli(), roomID); err != nil {
			return nil, err
		}
		next := now.Add(time.Duration(cfg.InterRoundPauseSeconds) * time.Second).UnixMilli()
		res.NextRoundAt = &next
	}
	return res, nil
}

// PublicAssignments converts the internal assignList to a JSON-marshallable
// slice for the game.won event payload.
func PublicAssignments(in []AssignmentEntry) []map[string]string {
	out := make([]map[string]string, 0, len(in))
	for _, a := range in {
		out = append(out, map[string]string{
			"player_id":    a.PlayerID,
			"character_id": a.CharacterID,
		})
	}
	return out
}

// EmitStateChanged is a small helper for emitting the canonical state.changed
// event after a transition.
func EmitStateChanged(pub events.Publisher, roomID string, state RoomState, roundNumber int) {
	pub.Publish(roomID, events.Event{
		Name: "state.changed",
		Data: map[string]any{"state": string(state), "round_number": roundNumber},
	})
}
