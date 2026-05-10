// handlers.go — HTTP endpoints for game actions:
//
//	POST /api/v1/rooms/:code/start      (host only)
//	POST /api/v1/rooms/:code/character  (only valid in CHARCREATE)
//	POST /api/v1/rooms/:code/answer
//	POST /api/v1/rooms/:code/guess
//	POST /api/v1/rooms/:code/abandon    (host only)
//
// All five require a session cookie. Authorisation (host vs non-host) is
// enforced by checking the `is_host` flag carried in the resolved Session.
//
// The handlers compose with the pure logic in scoring.go / rounds.go /
// assignment.go / state.go. They own:
//   - request decoding + validation
//   - DB transactions
//   - event emission (state.changed, round.started, etc.)
//   - state-machine guards (e.g. /character only valid in CHARCREATE)
//
// The implementation lives in this package rather than internal/handlers to
// avoid an import cycle with the engine; see setup.go for the reasoning.
package game

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/skittercore-studio/friendslop/internal/events"
	"github.com/skittercore-studio/friendslop/internal/session"
)

// gameHandler bundles the handler dependencies.
type gameHandler struct {
	db             *sql.DB
	pub            events.Publisher
	content        *Content
	engine         *Engine
	requireSession func(http.Handler) http.Handler
}

func newHandler(db *sql.DB, pub events.Publisher, content *Content, engine *Engine, requireSession func(http.Handler) http.Handler) *gameHandler {
	if pub == nil {
		pub = events.NoopPublisher{}
	}
	return &gameHandler{
		db:             db,
		pub:            pub,
		content:        content,
		engine:         engine,
		requireSession: requireSession,
	}
}

// Mount installs the game-action routes on r. All endpoints require a
// resolved session.
func (h *gameHandler) Mount(r chi.Router) {
	r.Group(func(g chi.Router) {
		g.Use(h.requireSession)
		g.Post("/api/v1/rooms/{code}/start", h.start)
		g.Post("/api/v1/rooms/{code}/character", h.character)
		g.Post("/api/v1/rooms/{code}/answer", h.answer)
		g.Post("/api/v1/rooms/{code}/guess", h.guess)
		g.Post("/api/v1/rooms/{code}/abandon", h.abandon)
	})
}

// ----------------------------------------------------------------------------
// Wire types

type characterRequest struct {
	Name  string `json:"name"`
	Blurb string `json:"blurb"`
}

type answerRequest struct {
	RoundNumber int    `json:"round_number"`
	Text        string `json:"text"`
}

type guessRequest struct {
	RoundNumber int               `json:"round_number"`
	Mapping     map[string]string `json:"mapping"`
}

// ----------------------------------------------------------------------------
// Handlers

// start: LOBBY -> ANSWERING (curated) or LOBBY -> CHARCREATE (playerwritten).
func (h *gameHandler) start(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	s, err := requireRoomSession(r, code)
	if err != nil {
		writeErr(w, err)
		return
	}
	if !s.IsHost {
		writeError(w, http.StatusForbidden, "only the host can start the game")
		return
	}

	ctx := r.Context()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var (
		state, poolSrc string
	)
	if err := tx.QueryRowContext(ctx,
		`SELECT state, pool_source FROM rooms WHERE id = ?`, s.RoomID,
	).Scan(&state, &poolSrc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if RoomState(state) != StateLobby {
		writeError(w, http.StatusConflict, "room is not in lobby state")
		return
	}

	playerIDs, err := LivePlayerIDs(ctx, tx, s.RoomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(playerIDs) < MinPlayers {
		writeError(w, http.StatusBadRequest, ErrTooFewPlayers.Error())
		return
	}

	now := time.Now()
	switch poolSrc {
	case "curated":
		// Roll N characters, assign 1:1, open round 1.
		charIDs, err := RollCuratedPool(ctx, tx, s.RoomID, h.content.Characters, len(playerIDs))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := AssignCharacters(ctx, tx, playerIDs, charIDs); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		roundID, number, qtext, deadline, err := OpenRound(ctx, tx, s.RoomID, h.content.Questions, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = roundID
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		EmitStateChanged(h.pub, s.RoomID, StateAnswering, number)
		h.pub.Publish(s.RoomID, events.Event{
			Name: "round.started",
			Data: map[string]any{
				"round_number":    number,
				"question_text":   qtext,
				"answer_deadline": deadline,
			},
		})

	case "playerwritten":
		// Move into CHARCREATE; emit deadline if any.
		var charTO sql.NullInt64
		if err := tx.QueryRowContext(ctx,
			`SELECT charcreate_timeout_seconds FROM rooms WHERE id = ?`, s.RoomID,
		).Scan(&charTO); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		var deadline *int64
		if charTO.Valid {
			d := now.Add(time.Duration(charTO.Int64) * time.Second).UnixMilli()
			deadline = &d
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE rooms SET state = 'charcreate', last_activity_at = ?, started_at = COALESCE(started_at, ?)
			WHERE id = ?
		`, now.UnixMilli(), now.UnixMilli(), s.RoomID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		EmitStateChanged(h.pub, s.RoomID, StateCharcreate, 0)
		h.pub.Publish(s.RoomID, events.Event{
			Name: "charcreate.started",
			Data: map[string]any{"deadline": deadline},
		})

	default:
		writeError(w, http.StatusInternalServerError, "unknown pool_source: "+poolSrc)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// character: submit one (name, blurb) during CHARCREATE. When all live players
// have submitted, shuffle the pool, assign, and transition to ANSWERING(R1).
func (h *gameHandler) character(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	s, err := requireRoomSession(r, code)
	if err != nil {
		writeErr(w, err)
		return
	}

	var req characterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	blurb := strings.TrimSpace(req.Blurb)
	if l := len([]rune(name)); l < 1 || l > 60 {
		writeError(w, http.StatusBadRequest, "name must be 1-60 chars")
		return
	}
	if l := len([]rune(blurb)); l < 20 || l > 300 {
		writeError(w, http.StatusBadRequest, "blurb must be 20-300 chars")
		return
	}

	ctx := r.Context()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var state string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM rooms WHERE id = ?`, s.RoomID).Scan(&state); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if RoomState(state) != StateCharcreate {
		writeError(w, http.StatusConflict, "room is not in charcreate state")
		return
	}

	// Reject duplicate submissions.
	var existing int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM room_characters WHERE room_id = ? AND author_player_id = ?`,
		s.RoomID, s.PlayerID,
	).Scan(&existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing > 0 {
		writeError(w, http.StatusConflict, "you have already submitted a character")
		return
	}

	id := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO room_characters (id, room_id, template_id, author_player_id, name, blurb)
		VALUES (?, ?, NULL, ?, ?, ?)
	`, id, s.RoomID, s.PlayerID, name, blurb); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE rooms SET last_activity_at = ? WHERE id = ?`,
		time.Now().UnixMilli(), s.RoomID,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// All-submitted check + transition to ANSWERING.
	playerIDs, err := LivePlayerIDs(ctx, tx, s.RoomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	charIDs, err := PlayerWrittenCharacterIDs(ctx, tx, s.RoomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	submittedCount := len(charIDs)
	totalPlayers := len(playerIDs)
	allSubmitted := submittedCount == totalPlayers

	var (
		started      bool
		startedRound int
		startedQ     string
		startedDL    *int64
		revealChars  []PublicCharacter
	)

	if allSubmitted {
		// Deterministic authored assignment: every player performs the
		// character they wrote. No shuffle — the mechanic is "guess who
		// wrote (= played) which voice". A future "shuffled" room option
		// can branch here.
		if _, err := AssignAuthoredCharacters(ctx, tx, s.RoomID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		now := time.Now()
		_, number, qtext, deadline, err := OpenRound(ctx, tx, s.RoomID, h.content.Questions, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		started = true
		startedRound = number
		startedQ = qtext
		startedDL = deadline

		// Build the public-pool reveal payload (no authorship).
		rows, err := tx.QueryContext(ctx, `
			SELECT id, name, blurb FROM room_characters WHERE room_id = ? ORDER BY id ASC
		`, s.RoomID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for rows.Next() {
			var c PublicCharacter
			if err := rows.Scan(&c.ID, &c.Name, &c.Blurb); err != nil {
				rows.Close()
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			revealChars = append(revealChars, c)
		}
		rows.Close()
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Aggregate progress event.
	h.pub.Publish(s.RoomID, events.Event{
		Name: "charcreate.submitted",
		Data: map[string]any{
			"submitted_count": submittedCount,
			"total_players":   totalPlayers,
		},
	})

	if started {
		// Reveal pool, transition state, emit round.started.
		h.pub.Publish(s.RoomID, events.Event{
			Name: "charcreate.completed",
			Data: map[string]any{"characters": revealChars},
		})
		EmitStateChanged(h.pub, s.RoomID, StateAnswering, startedRound)
		h.pub.Publish(s.RoomID, events.Event{
			Name: "round.started",
			Data: map[string]any{
				"round_number":    startedRound,
				"question_text":   startedQ,
				"answer_deadline": startedDL,
			},
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// PublicCharacter mirrors handlers.PublicCharacter for the charcreate.completed
// payload — duplicated here to avoid a handlers import cycle.
type PublicCharacter struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Blurb string `json:"blurb"`
}

// answer: submit a per-round answer in ANSWERING. Stale round_numbers 409.
func (h *gameHandler) answer(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	s, err := requireRoomSession(r, code)
	if err != nil {
		writeErr(w, err)
		return
	}

	var req answerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	text := strings.TrimSpace(req.Text)
	if l := len([]rune(text)); l < 1 || l > 1000 {
		writeError(w, http.StatusBadRequest, "text must be 1-1000 chars")
		return
	}

	ctx := r.Context()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var state string
	var roundNum int
	if err := tx.QueryRowContext(ctx,
		`SELECT state, round_number FROM rooms WHERE id = ?`, s.RoomID,
	).Scan(&state, &roundNum); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if RoomState(state) != StateAnswering {
		writeError(w, http.StatusConflict, "room is not in answering state")
		return
	}
	if req.RoundNumber != roundNum {
		writeError(w, http.StatusConflict, fmt.Sprintf("stale round_number: have %d, want %d", req.RoundNumber, roundNum))
		return
	}

	var roundID, rstate string
	if err := tx.QueryRowContext(ctx, `
		SELECT id, state FROM rounds WHERE room_id = ? AND number = ?
	`, s.RoomID, roundNum).Scan(&roundID, &rstate); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if RoundState(rstate) != RoundAnswering {
		writeError(w, http.StatusConflict, "round is not accepting answers")
		return
	}

	now := time.Now().UnixMilli()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO answers (round_id, player_id, text, submitted_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(round_id, player_id) DO UPDATE SET text = excluded.text, submitted_at = excluded.submitted_at
	`, roundID, s.PlayerID, text, now); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE rooms SET last_activity_at = ? WHERE id = ?`, now, s.RoomID,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// All-submitted check?
	var submittedCount, totalPlayers int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT a.player_id)
		FROM answers a
		JOIN players p ON p.id = a.player_id
		WHERE a.round_id = ? AND p.left_at IS NULL
	`, roundID).Scan(&submittedCount); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM players WHERE room_id = ? AND left_at IS NULL
	`, s.RoomID).Scan(&totalPlayers); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var (
		closed       bool
		reveal       []RevealAnswer
		guessDL      *int64
		closedRound  int
	)
	if submittedCount == totalPlayers {
		var err error
		reveal, guessDL, err = CloseAnswering(ctx, tx, s.RoomID, roundID, time.Now())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		closedRound = roundNum
		closed = true
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.pub.Publish(s.RoomID, events.Event{
		Name: "answer.submitted",
		Data: map[string]any{
			"submitted_count": submittedCount,
			"total_players":   totalPlayers,
		},
	})
	if closed {
		EmitStateChanged(h.pub, s.RoomID, StateGuessing, closedRound)
		h.pub.Publish(s.RoomID, events.Event{
			Name: "round.answers_revealed",
			Data: map[string]any{
				"round_number":   closedRound,
				"answers":        reveal,
				"guess_deadline": guessDL,
			},
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// guess: submit a full mapping in GUESSING. When all guesses are in, score
// the round and either advance to scoring (then schedule next round) or end
// the game.
func (h *gameHandler) guess(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	s, err := requireRoomSession(r, code)
	if err != nil {
		writeErr(w, err)
		return
	}

	var req guessRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Mapping == nil {
		writeError(w, http.StatusBadRequest, "mapping required")
		return
	}

	ctx := r.Context()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var state string
	var roundNum int
	if err := tx.QueryRowContext(ctx,
		`SELECT state, round_number FROM rooms WHERE id = ?`, s.RoomID,
	).Scan(&state, &roundNum); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if RoomState(state) != StateGuessing {
		writeError(w, http.StatusConflict, "room is not in guessing state")
		return
	}
	if req.RoundNumber != roundNum {
		writeError(w, http.StatusConflict, fmt.Sprintf("stale round_number: have %d, want %d", req.RoundNumber, roundNum))
		return
	}

	var roundID, rstate string
	if err := tx.QueryRowContext(ctx, `
		SELECT id, state FROM rounds WHERE room_id = ? AND number = ?
	`, s.RoomID, roundNum).Scan(&roundID, &rstate); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if RoundState(rstate) != RoundGuessing {
		writeError(w, http.StatusConflict, "round is not accepting guesses")
		return
	}

	// Validate the mapping against the live roster + character pool.
	otherPlayerIDs, err := otherLivePlayerIDs(ctx, tx, s.RoomID, s.PlayerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	poolCharacterIDs, err := poolCharacterIDs(ctx, tx, s.RoomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := ValidateMapping(s.PlayerID, req.Mapping, otherPlayerIDs, poolCharacterIDs); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Reject duplicate submissions.
	var existing int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM guesses WHERE round_id = ? AND guesser_player_id = ?`,
		roundID, s.PlayerID,
	).Scan(&existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing > 0 {
		writeError(w, http.StatusConflict, "you have already submitted a guess for this round")
		return
	}

	now := time.Now().UnixMilli()
	guessID := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO guesses (id, round_id, guesser_player_id, correct_count, submitted_at)
		VALUES (?, ?, ?, 0, ?)
	`, guessID, roundID, s.PlayerID, now); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for target, charID := range req.Mapping {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO guess_entries (guess_id, target_player_id, character_id)
			VALUES (?, ?, ?)
		`, guessID, target, charID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE rooms SET last_activity_at = ? WHERE id = ?`, now, s.RoomID,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// All-submitted check?
	var submittedCount, totalPlayers int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT g.guesser_player_id)
		FROM guesses g
		JOIN players p ON p.id = g.guesser_player_id
		WHERE g.round_id = ? AND p.left_at IS NULL
	`, roundID).Scan(&submittedCount); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM players WHERE room_id = ? AND left_at IS NULL
	`, s.RoomID).Scan(&totalPlayers); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var (
		closed bool
		res    *ScoringResult
	)
	if submittedCount == totalPlayers {
		res, err = CloseGuessing(ctx, tx, s.RoomID, roundID, time.Now())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		closed = true
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.pub.Publish(s.RoomID, events.Event{
		Name: "guess.submitted",
		Data: map[string]any{
			"submitted_count": submittedCount,
			"total_players":   totalPlayers,
		},
	})

	if closed {
		nextState := StateScoring
		if res.GameOver {
			nextState = StateWon
		}
		EmitStateChanged(h.pub, s.RoomID, nextState, res.RoundNumber)
		h.pub.Publish(s.RoomID, events.Event{
			Name: "round.scored",
			Data: map[string]any{
				"round_number":  res.RoundNumber,
				"public_scores": res.PublicScores,
				"next_round_at": res.NextRoundAt,
			},
		})
		if res.GameOver {
			h.pub.Publish(s.RoomID, events.Event{
				Name: "game.won",
				Data: map[string]any{
					"winner_player_id":  res.WinnerPlayerID,
					"true_assignments":  PublicAssignments(res.TrueAssignments),
				},
			})
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// abandon: host explicitly ends the room. Any non-terminal state -> ABANDONED.
func (h *gameHandler) abandon(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	s, err := requireRoomSession(r, code)
	if err != nil {
		writeErr(w, err)
		return
	}
	if !s.IsHost {
		writeError(w, http.StatusForbidden, "only the host can abandon the room")
		return
	}

	ctx := r.Context()
	now := time.Now().UnixMilli()

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var state string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM rooms WHERE id = ?`, s.RoomID).Scan(&state); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if IsTerminalRoomState(RoomState(state)) {
		writeError(w, http.StatusConflict, "room is already in a terminal state")
		return
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE rooms SET state = 'abandoned', ended_at = ?, last_activity_at = ?
		WHERE id = ?
	`, now, now, s.RoomID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	EmitStateChanged(h.pub, s.RoomID, StateAbandoned, 0)
	h.pub.Publish(s.RoomID, events.Event{
		Name: "game.abandoned",
		Data: map[string]any{"reason": "host_quit"},
	})

	w.WriteHeader(http.StatusNoContent)
}

// ----------------------------------------------------------------------------
// helpers

// requireRoomSession resolves the request's session and verifies it's bound to
// the room identified by `code` in the URL.
func requireRoomSession(r *http.Request, code string) (*session.Session, error) {
	s, err := session.FromContext(r.Context())
	if err != nil {
		return nil, &httpError{Status: http.StatusUnauthorized, Msg: "no session"}
	}
	if s.RoomCode != code {
		return nil, &httpError{Status: http.StatusForbidden, Msg: "session does not belong to this room"}
	}
	return s, nil
}

func otherLivePlayerIDs(ctx context.Context, tx *sql.Tx, roomID, selfID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM players
		WHERE room_id = ? AND left_at IS NULL AND id != ?
		ORDER BY joined_at ASC
	`, roomID, selfID)
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

func poolCharacterIDs(ctx context.Context, tx *sql.Tx, roomID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM room_characters WHERE room_id = ? ORDER BY id ASC
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

type httpError struct {
	Status int
	Msg    string
}

func (e *httpError) Error() string { return e.Msg }

func writeErr(w http.ResponseWriter, err error) {
	var he *httpError
	if errors.As(err, &he) {
		writeError(w, he.Status, he.Msg)
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func decodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
