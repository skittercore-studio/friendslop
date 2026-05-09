// Package handlers contains the HTTP handler implementations for friendslop.
//
// rooms.go owns room CRUD: create, join, public snapshot, private /me view,
// leave. The host abandon endpoint is stubbed pending the gamelogic agent.
//
// Other agents will add files in this package (game.go, character.go,
// answer.go, guess.go, sse.go) and register their routes via
// server.Server.AddRoutes. They MUST NOT modify this file.
package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/skittercore-studio/friendslop/internal/events"
	"github.com/skittercore-studio/friendslop/internal/session"
)

// codeAlphabet excludes I and O to avoid 1/0 confusion when read aloud.
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ"
const codeLength = 4
const maxCodeRetries = 12

// Rooms bundles the dependencies used by the room handlers.
type Rooms struct {
	DB  *sql.DB
	Pub events.Publisher
}

// NewRooms constructs a Rooms handler set. nil Pub is replaced by NoopPublisher
// so handlers can call Publish unconditionally.
func NewRooms(d *sql.DB, p events.Publisher) *Rooms {
	if p == nil {
		p = events.NoopPublisher{}
	}
	return &Rooms{DB: d, Pub: p}
}

// Mount registers the room CRUD routes onto r. Caller is expected to wrap
// /me and /leave with a session-requiring middleware before mounting; see
// server.Server.Mount.
func (h *Rooms) Mount(r chi.Router, requireSession func(http.Handler) http.Handler) {
	r.Post("/api/v1/rooms", h.Create)
	r.Post("/api/v1/rooms/{code}/join", h.Join)
	r.Get("/api/v1/rooms/{code}", h.PublicSnapshot)

	r.Group(func(g chi.Router) {
		g.Use(requireSession)
		g.Get("/api/v1/rooms/{code}/me", h.PrivateMe)
		g.Post("/api/v1/rooms/{code}/leave", h.Leave)
		// /abandon is registered by internal/game (gamelogic agent) once the
		// game package is wired in; bootstrap leaves it unrouted so chi does
		// not panic on duplicate registrations. The AbandonStub method is
		// retained below for backward-compatibility while the gamelogic agent
		// completes wiring.
	})
}

// ----------------------------------------------------------------------------
// Wire types

type createRequest struct {
	HostName                 string `json:"host_name"`
	Mode                     string `json:"mode"`
	PoolSource               string `json:"pool_source"`
	AnswerTimeoutSeconds     *int   `json:"answer_timeout_seconds,omitempty"`
	GuessTimeoutSeconds      *int   `json:"guess_timeout_seconds,omitempty"`
	CharcreateTimeoutSeconds *int   `json:"charcreate_timeout_seconds,omitempty"`
}

type createResponse struct {
	RoomCode     string `json:"room_code"`
	PlayerID     string `json:"player_id"`
	SessionToken string `json:"session_token"`
}

type joinRequest struct {
	Name string `json:"name"`
}

type joinResponse struct {
	RoomCode     string `json:"room_code"`
	PlayerID     string `json:"player_id"`
	SessionToken string `json:"session_token"`
}

// PublicPlayer is the redacted view appearing in the public snapshot.
type PublicPlayer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	IsHost bool   `json:"is_host"`
	Left   bool   `json:"left"`
}

// PublicCharacter is a room-scoped character with no authorship leak.
type PublicCharacter struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Blurb string `json:"blurb"`
}

// PublicSnapshotResponse mirrors SPEC.md §4.1.
type PublicSnapshotResponse struct {
	Code           string            `json:"code"`
	State          string            `json:"state"`
	Mode           string            `json:"mode"`
	PoolSource     string            `json:"pool_source"`
	RoundNumber    int               `json:"round_number"`
	Players        []PublicPlayer    `json:"players"`
	Characters     []PublicCharacter `json:"characters,omitempty"`
	CurrentRound   *currentRoundView `json:"current_round,omitempty"`
	Scoreboard     scoreboardView    `json:"scoreboard"`
	WinnerPlayerID *string           `json:"winner_player_id"`
}

type currentRoundView struct {
	Number         int              `json:"number"`
	State          string           `json:"state"`
	QuestionText   string           `json:"question_text"`
	AnswerDeadline *int64           `json:"answer_deadline"`
	GuessDeadline  *int64           `json:"guess_deadline"`
	Answers        []revealedAnswer `json:"answers,omitempty"`
}

type revealedAnswer struct {
	CharacterID string `json:"character_id"`
	Text        string `json:"text"`
}

type scoreboardView struct {
	Rounds []roundScores `json:"rounds"`
}

type roundScores struct {
	RoundNumber int            `json:"round_number"`
	Scores      map[string]int `json:"scores"`
}

// PrivateMeResponse mirrors SPEC.md §4.2.
type PrivateMeResponse struct {
	PlayerID                string            `json:"player_id"`
	Name                    string            `json:"name"`
	IsHost                  bool              `json:"is_host"`
	YourCharacter           *PublicCharacter  `json:"your_character"`
	YourAuthoredCharacterID *string           `json:"your_authored_character_id,omitempty"`
	YourAnswerForCurrent    *string           `json:"your_answer_for_current_round"`
	YourGuessForCurrent     map[string]string `json:"your_guess_for_current_round"`
	YourPastGuesses         []pastGuess       `json:"your_past_guesses"`
}

type pastGuess struct {
	RoundNumber  int               `json:"round_number"`
	Mapping      map[string]string `json:"mapping"`
	CorrectCount int               `json:"correct_count"`
}

// ----------------------------------------------------------------------------
// Handlers

// Create handles POST /api/v1/rooms — creates a room and host player.
func (h *Rooms) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hostName := strings.TrimSpace(req.HostName)
	if !validName(hostName) {
		writeError(w, http.StatusBadRequest, "host_name must be 1-32 chars")
		return
	}
	if req.Mode != "live" && req.Mode != "async" {
		writeError(w, http.StatusBadRequest, "mode must be live or async")
		return
	}
	if req.PoolSource != "curated" && req.PoolSource != "playerwritten" {
		writeError(w, http.StatusBadRequest, "pool_source must be curated or playerwritten")
		return
	}

	answerTO := defaultTimeout(req.AnswerTimeoutSeconds, req.Mode, 120, 24*60*60)
	guessTO := defaultTimeout(req.GuessTimeoutSeconds, req.Mode, 120, 24*60*60)
	var charTO *int
	if req.PoolSource == "playerwritten" {
		v := defaultTimeout(req.CharcreateTimeoutSeconds, req.Mode, 300, 24*60*60)
		charTO = &v
	}

	roomID := uuid.NewString()
	playerID := uuid.NewString()
	token, err := session.NewToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token gen failed")
		return
	}
	now := time.Now().UnixMilli()

	ctx := r.Context()
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tx begin failed")
		return
	}
	defer tx.Rollback()

	code, err := allocateCode(ctx, tx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO rooms (
			id, code, state, mode, pool_source, charcreate_timeout_seconds,
			host_player_id, round_number,
			answer_timeout_seconds, guess_timeout_seconds,
			inter_round_pause_seconds, question_bank, character_pool,
			created_at, last_activity_at
		) VALUES (?, ?, 'lobby', ?, ?, ?, ?, 0, ?, ?, 10, 'default', 'default', ?, ?)
	`, roomID, code, req.Mode, req.PoolSource, nullableInt(charTO),
		playerID, answerTO, guessTO,
		now, now,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "room insert failed: "+err.Error())
		return
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO players (id, room_id, name, session_token, is_host, joined_at)
		VALUES (?, ?, ?, ?, 1, ?)
	`, playerID, roomID, hostName, token, now); err != nil {
		writeError(w, http.StatusInternalServerError, "player insert failed: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "tx commit failed")
		return
	}

	session.SetCookie(w, r, token)
	writeJSON(w, http.StatusCreated, createResponse{
		RoomCode:     code,
		PlayerID:     playerID,
		SessionToken: token,
	})
}

// Join handles POST /api/v1/rooms/{code}/join.
func (h *Rooms) Join(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	var req joinRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if !validName(name) {
		writeError(w, http.StatusBadRequest, "name must be 1-32 chars")
		return
	}

	ctx := r.Context()
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tx begin failed")
		return
	}
	defer tx.Rollback()

	var roomID, state string
	if err := tx.QueryRowContext(ctx,
		`SELECT id, state FROM rooms WHERE code = ?`, code,
	).Scan(&roomID, &state); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if state != "lobby" {
		writeError(w, http.StatusConflict, "room not accepting joins")
		return
	}

	playerID := uuid.NewString()
	token, err := session.NewToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token gen failed")
		return
	}
	now := time.Now().UnixMilli()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO players (id, room_id, name, session_token, is_host, joined_at)
		VALUES (?, ?, ?, ?, 0, ?)
	`, playerID, roomID, name, token, now); err != nil {
		// UNIQUE (room_id, name) collision is a 409, not a 500.
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "name already taken in this room")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE rooms SET last_activity_at = ? WHERE id = ?`, now, roomID,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.Pub.Publish(roomID, events.Event{
		Name: "player.joined",
		Data: map[string]any{"player_id": playerID, "name": name},
	})

	session.SetCookie(w, r, token)
	writeJSON(w, http.StatusOK, joinResponse{
		RoomCode:     code,
		PlayerID:     playerID,
		SessionToken: token,
	})
}

// PublicSnapshot handles GET /api/v1/rooms/{code} — anonymous read.
func (h *Rooms) PublicSnapshot(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	ctx := r.Context()

	snap, err := h.buildPublicSnapshot(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (h *Rooms) buildPublicSnapshot(ctx context.Context, code string) (*PublicSnapshotResponse, error) {
	var snap PublicSnapshotResponse
	var roomID string
	var winner sql.NullString

	if err := h.DB.QueryRowContext(ctx, `
		SELECT id, code, state, mode, pool_source, round_number, winner_player_id
		FROM rooms WHERE code = ?
	`, code).Scan(&roomID, &snap.Code, &snap.State, &snap.Mode, &snap.PoolSource, &snap.RoundNumber, &winner); err != nil {
		return nil, err
	}
	if winner.Valid {
		s := winner.String
		snap.WinnerPlayerID = &s
	}

	// Players
	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, name, is_host, left_at
		FROM players WHERE room_id = ?
		ORDER BY joined_at ASC
	`, roomID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var pp PublicPlayer
		var isHost int
		var leftAt sql.NullInt64
		if err := rows.Scan(&pp.ID, &pp.Name, &isHost, &leftAt); err != nil {
			rows.Close()
			return nil, err
		}
		pp.IsHost = isHost != 0
		pp.Left = leftAt.Valid
		snap.Players = append(snap.Players, pp)
	}
	rows.Close()
	if snap.Players == nil {
		snap.Players = []PublicPlayer{}
	}

	// Characters — only revealed once we've left lobby/charcreate.
	if snap.State != "lobby" && snap.State != "charcreate" {
		crows, err := h.DB.QueryContext(ctx, `
			SELECT id, name, blurb FROM room_characters
			WHERE room_id = ? ORDER BY id ASC
		`, roomID)
		if err != nil {
			return nil, err
		}
		for crows.Next() {
			var c PublicCharacter
			if err := crows.Scan(&c.ID, &c.Name, &c.Blurb); err != nil {
				crows.Close()
				return nil, err
			}
			snap.Characters = append(snap.Characters, c)
		}
		crows.Close()
	}

	// Current round summary (if any).
	if snap.RoundNumber > 0 {
		var roundID, rstate, qtext string
		var rnum int
		var ad, gd sql.NullInt64
		err := h.DB.QueryRowContext(ctx, `
			SELECT id, number, state, question_text, answer_deadline, guess_deadline
			FROM rounds WHERE room_id = ? AND number = ?
		`, roomID, snap.RoundNumber).Scan(&roundID, &rnum, &rstate, &qtext, &ad, &gd)
		if err == nil {
			cr := &currentRoundView{
				Number:       rnum,
				State:        rstate,
				QuestionText: qtext,
			}
			if ad.Valid {
				v := ad.Int64
				cr.AnswerDeadline = &v
			}
			if gd.Valid {
				v := gd.Int64
				cr.GuessDeadline = &v
			}
			// Reveal answers only in guessing/scoring states. Sort by character_id
			// to avoid leaking submission order — see SPEC.md §1.
			if rstate == "guessing" || rstate == "scoring" {
				arows, err := h.DB.QueryContext(ctx, `
					SELECT rc.id, a.text
					FROM answers a
					JOIN players p ON p.id = a.player_id
					JOIN room_characters rc ON rc.id = p.character_id
					WHERE a.round_id = ?
					ORDER BY rc.id ASC
				`, roundID)
				if err == nil {
					for arows.Next() {
						var ra revealedAnswer
						if err := arows.Scan(&ra.CharacterID, &ra.Text); err == nil {
							cr.Answers = append(cr.Answers, ra)
						}
					}
					arows.Close()
				}
			}
			snap.CurrentRound = cr
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	// Scoreboard — public counts per round.
	srows, err := h.DB.QueryContext(ctx, `
		SELECT rd.number, g.guesser_player_id, g.correct_count
		FROM guesses g
		JOIN rounds rd ON rd.id = g.round_id
		WHERE rd.room_id = ?
		ORDER BY rd.number ASC
	`, roomID)
	if err != nil {
		return nil, err
	}
	rounds := map[int]map[string]int{}
	var order []int
	for srows.Next() {
		var rn, cc int
		var pid string
		if err := srows.Scan(&rn, &pid, &cc); err != nil {
			srows.Close()
			return nil, err
		}
		if _, ok := rounds[rn]; !ok {
			rounds[rn] = map[string]int{}
			order = append(order, rn)
		}
		rounds[rn][pid] = cc
	}
	srows.Close()
	for _, rn := range order {
		snap.Scoreboard.Rounds = append(snap.Scoreboard.Rounds, roundScores{RoundNumber: rn, Scores: rounds[rn]})
	}
	if snap.Scoreboard.Rounds == nil {
		snap.Scoreboard.Rounds = []roundScores{}
	}
	return &snap, nil
}

// PrivateMe handles GET /api/v1/rooms/{code}/me.
func (h *Rooms) PrivateMe(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	s, err := session.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "no session")
		return
	}
	if s.RoomCode != code {
		// Cookie does not match the room being requested.
		writeError(w, http.StatusForbidden, "session does not belong to this room")
		return
	}

	ctx := r.Context()
	var resp PrivateMeResponse
	resp.PlayerID = s.PlayerID
	resp.IsHost = s.IsHost
	resp.YourGuessForCurrent = map[string]string{}
	resp.YourPastGuesses = []pastGuess{}

	var (
		name      string
		charID    sql.NullString
		roomState string
		roundNum  int
		poolSrc   string
	)
	if err := h.DB.QueryRowContext(ctx, `
		SELECT p.name, p.character_id, r.state, r.round_number, r.pool_source
		FROM players p JOIN rooms r ON r.id = p.room_id
		WHERE p.id = ?
	`, s.PlayerID).Scan(&name, &charID, &roomState, &roundNum, &poolSrc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Name = name

	if charID.Valid {
		var pc PublicCharacter
		if err := h.DB.QueryRowContext(ctx,
			`SELECT id, name, blurb FROM room_characters WHERE id = ?`, charID.String,
		).Scan(&pc.ID, &pc.Name, &pc.Blurb); err == nil {
			resp.YourCharacter = &pc
		}
	}

	if poolSrc == "playerwritten" && roomState != "lobby" && roomState != "charcreate" {
		var authored sql.NullString
		_ = h.DB.QueryRowContext(ctx,
			`SELECT id FROM room_characters WHERE author_player_id = ?`, s.PlayerID,
		).Scan(&authored)
		if authored.Valid {
			v := authored.String
			resp.YourAuthoredCharacterID = &v
		}
	}

	if roundNum > 0 {
		var roundID string
		if err := h.DB.QueryRowContext(ctx,
			`SELECT id FROM rounds WHERE room_id = ? AND number = ?`, s.RoomID, roundNum,
		).Scan(&roundID); err == nil {
			var ans sql.NullString
			_ = h.DB.QueryRowContext(ctx,
				`SELECT text FROM answers WHERE round_id = ? AND player_id = ?`,
				roundID, s.PlayerID,
			).Scan(&ans)
			if ans.Valid {
				v := ans.String
				resp.YourAnswerForCurrent = &v
			}

			var guessID sql.NullString
			_ = h.DB.QueryRowContext(ctx,
				`SELECT id FROM guesses WHERE round_id = ? AND guesser_player_id = ?`,
				roundID, s.PlayerID,
			).Scan(&guessID)
			if guessID.Valid {
				erows, err := h.DB.QueryContext(ctx,
					`SELECT target_player_id, character_id FROM guess_entries WHERE guess_id = ?`,
					guessID.String,
				)
				if err == nil {
					for erows.Next() {
						var tp, cid string
						if err := erows.Scan(&tp, &cid); err == nil {
							resp.YourGuessForCurrent[tp] = cid
						}
					}
					erows.Close()
				}
			}
		}
	}

	// Past guesses: every closed round before current.
	prows, err := h.DB.QueryContext(ctx, `
		SELECT rd.number, g.id, g.correct_count
		FROM guesses g
		JOIN rounds rd ON rd.id = g.round_id
		WHERE rd.room_id = ? AND g.guesser_player_id = ? AND rd.number < ?
		ORDER BY rd.number ASC
	`, s.RoomID, s.PlayerID, roundNum)
	if err == nil {
		for prows.Next() {
			var rn, cc int
			var gid string
			if err := prows.Scan(&rn, &gid, &cc); err != nil {
				continue
			}
			pg := pastGuess{RoundNumber: rn, Mapping: map[string]string{}, CorrectCount: cc}
			erows, err := h.DB.QueryContext(ctx,
				`SELECT target_player_id, character_id FROM guess_entries WHERE guess_id = ?`,
				gid)
			if err == nil {
				for erows.Next() {
					var tp, cid string
					if err := erows.Scan(&tp, &cid); err == nil {
						pg.Mapping[tp] = cid
					}
				}
				erows.Close()
			}
			resp.YourPastGuesses = append(resp.YourPastGuesses, pg)
		}
		prows.Close()
	}

	writeJSON(w, http.StatusOK, resp)
}

// Leave handles POST /api/v1/rooms/{code}/leave.
func (h *Rooms) Leave(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	s, err := session.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "no session")
		return
	}
	if s.RoomCode != code {
		writeError(w, http.StatusForbidden, "session does not belong to this room")
		return
	}

	now := time.Now().UnixMilli()
	if _, err := h.DB.ExecContext(r.Context(),
		`UPDATE players SET left_at = ? WHERE id = ? AND left_at IS NULL`,
		now, s.PlayerID,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = h.DB.ExecContext(r.Context(),
		`UPDATE rooms SET last_activity_at = ? WHERE id = ?`, now, s.RoomID)

	h.Pub.Publish(s.RoomID, events.Event{
		Name: "player.left",
		Data: map[string]any{"player_id": s.PlayerID},
	})
	session.ClearCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// AbandonStub returns 501 — the gamelogic agent owns the real implementation.
func (h *Rooms) AbandonStub(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "abandon not implemented yet (gamelogic agent)")
}

// ----------------------------------------------------------------------------
// helpers

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

func validName(s string) bool {
	if l := len(s); l < 1 || l > 32 {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func defaultTimeout(provided *int, mode string, liveDefault, asyncDefault int) int {
	if provided != nil {
		return *provided
	}
	if mode == "live" {
		return liveDefault
	}
	return asyncDefault
}

func nullableInt(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func isUniqueViolation(err error) bool {
	// The mattn/go-sqlite3 driver's Error type carries an ExtendedCode that can
	// be 2067 for UNIQUE; we keep the check string-based to avoid coupling here.
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// allocateCode finds a fresh unused 4-letter code, retrying on collision.
func allocateCode(ctx context.Context, tx *sql.Tx) (string, error) {
	for i := 0; i < maxCodeRetries; i++ {
		c, err := randomCode()
		if err != nil {
			return "", err
		}
		var exists int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM rooms WHERE code = ?`, c).Scan(&exists); err != nil {
			return "", err
		}
		if exists == 0 {
			return c, nil
		}
	}
	return "", errors.New("could not allocate unique room code")
}

func randomCode() (string, error) {
	out := make([]byte, codeLength)
	mod := big.NewInt(int64(len(codeAlphabet)))
	for i := 0; i < codeLength; i++ {
		n, err := rand.Int(rand.Reader, mod)
		if err != nil {
			return "", err
		}
		out[i] = codeAlphabet[n.Int64()]
	}
	return string(out), nil
}
