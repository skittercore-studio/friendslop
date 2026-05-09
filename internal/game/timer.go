// timer.go — single background goroutine that wakes every ~5s, looks for
// past-deadline rounds, and advances them.
//
// Behaviour:
//   - For 'answering' rounds with answer_deadline < now: close the round
//     (auto-fill missing answers), advance to 'guessing', emit
//     round.answers_revealed + state.changed.
//   - For 'guessing' rounds with guess_deadline < now: close the round,
//     score, advance to 'scoring', emit round.scored. If a winner: emit
//     game.won. Otherwise: schedule next round at now + inter_round_pause_seconds.
//   - For 'scoring' rooms whose last_activity_at + inter_round_pause_seconds < now,
//     and whose latest round is `done`: open a fresh round.
//
// async-mode rounds DO have deadlines (24h default per spec §2). The ticker
// therefore handles both modes uniformly.
//
// All DB work happens in a transaction. Each round is processed in its own
// transaction so a failure on one room does not stall others.
package game

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/skittercore-studio/friendslop/internal/events"
)

// TickInterval is how often the background goroutine wakes up.
const TickInterval = 5 * time.Second

// Engine bundles the dependencies required to advance rounds.
type Engine struct {
	DB      *sql.DB
	Pub     events.Publisher
	Content *Content
}

// NewEngine returns an Engine bound to db + publisher + content.
func NewEngine(db *sql.DB, pub events.Publisher, content *Content) *Engine {
	if pub == nil {
		pub = events.NoopPublisher{}
	}
	return &Engine{DB: db, Pub: pub, Content: content}
}

// Run starts the timer loop. Blocks until ctx is cancelled. Intended to be
// invoked as `go engine.Run(ctx)` from Setup.
func (e *Engine) Run(ctx context.Context) {
	t := time.NewTicker(TickInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := e.Tick(ctx, time.Now()); err != nil {
				log.Printf("game.engine tick: %v", err)
			}
		}
	}
}

// Tick is one pass of the deadline scan. Exported for tests so they can drive
// the engine deterministically without waiting for the ticker.
func (e *Engine) Tick(ctx context.Context, now time.Time) error {
	if err := e.advanceAnswering(ctx, now); err != nil {
		return err
	}
	if err := e.advanceGuessing(ctx, now); err != nil {
		return err
	}
	if err := e.advanceInterRound(ctx, now); err != nil {
		return err
	}
	return nil
}

func (e *Engine) advanceAnswering(ctx context.Context, now time.Time) error {
	rows, err := e.DB.QueryContext(ctx, `
		SELECT r.id, r.room_id
		FROM rounds r
		JOIN rooms ro ON ro.id = r.room_id
		WHERE r.state = 'answering'
		  AND r.answer_deadline IS NOT NULL
		  AND r.answer_deadline <= ?
		  AND ro.state = 'answering'
	`, now.UnixMilli())
	if err != nil {
		return err
	}
	type pair struct{ roundID, roomID string }
	var todo []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.roundID, &p.roomID); err != nil {
			rows.Close()
			return err
		}
		todo = append(todo, p)
	}
	rows.Close()

	for _, p := range todo {
		if err := e.closeAnsweringOne(ctx, p.roomID, p.roundID, now); err != nil {
			log.Printf("close answering room=%s round=%s: %v", p.roomID, p.roundID, err)
		}
	}
	return nil
}

func (e *Engine) closeAnsweringOne(ctx context.Context, roomID, roundID string, now time.Time) error {
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	reveal, deadline, err := CloseAnswering(ctx, tx, roomID, roundID, now)
	if err != nil {
		return err
	}
	var roundNumber int
	if err := tx.QueryRowContext(ctx, `SELECT number FROM rounds WHERE id = ?`, roundID).Scan(&roundNumber); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	EmitStateChanged(e.Pub, roomID, StateGuessing, roundNumber)
	e.Pub.Publish(roomID, events.Event{
		Name: "round.answers_revealed",
		Data: map[string]any{
			"round_number":   roundNumber,
			"answers":        reveal,
			"guess_deadline": deadline,
		},
	})
	return nil
}

func (e *Engine) advanceGuessing(ctx context.Context, now time.Time) error {
	rows, err := e.DB.QueryContext(ctx, `
		SELECT r.id, r.room_id
		FROM rounds r
		JOIN rooms ro ON ro.id = r.room_id
		WHERE r.state = 'guessing'
		  AND r.guess_deadline IS NOT NULL
		  AND r.guess_deadline <= ?
		  AND ro.state = 'guessing'
	`, now.UnixMilli())
	if err != nil {
		return err
	}
	type pair struct{ roundID, roomID string }
	var todo []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.roundID, &p.roomID); err != nil {
			rows.Close()
			return err
		}
		todo = append(todo, p)
	}
	rows.Close()

	for _, p := range todo {
		if err := e.closeGuessingOne(ctx, p.roomID, p.roundID, now); err != nil {
			log.Printf("close guessing room=%s round=%s: %v", p.roomID, p.roundID, err)
		}
	}
	return nil
}

func (e *Engine) closeGuessingOne(ctx context.Context, roomID, roundID string, now time.Time) error {
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := CloseGuessing(ctx, tx, roomID, roundID, now)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	EmitStateChanged(e.Pub, roomID, func() RoomState {
		if res.GameOver {
			return StateWon
		}
		return StateScoring
	}(), res.RoundNumber)

	e.Pub.Publish(roomID, events.Event{
		Name: "round.scored",
		Data: map[string]any{
			"round_number":  res.RoundNumber,
			"public_scores": res.PublicScores,
			"next_round_at": res.NextRoundAt,
		},
	})

	if res.GameOver {
		e.Pub.Publish(roomID, events.Event{
			Name: "game.won",
			Data: map[string]any{
				"winner_player_id":  res.WinnerPlayerID,
				"true_assignments":  PublicAssignments(res.TrueAssignments),
			},
		})
	}
	return nil
}

func (e *Engine) advanceInterRound(ctx context.Context, now time.Time) error {
	// Rooms in 'scoring' whose latest round is 'done' and where now exceeds
	// last_activity_at + inter_round_pause_seconds → start next round.
	rows, err := e.DB.QueryContext(ctx, `
		SELECT id FROM rooms
		WHERE state = 'scoring'
		  AND (last_activity_at + (inter_round_pause_seconds * 1000)) <= ?
	`, now.UnixMilli())
	if err != nil {
		return err
	}
	var roomIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		roomIDs = append(roomIDs, id)
	}
	rows.Close()

	for _, rid := range roomIDs {
		if err := e.openNextRound(ctx, rid, now); err != nil {
			log.Printf("open next round room=%s: %v", rid, err)
		}
	}
	return nil
}

func (e *Engine) openNextRound(ctx context.Context, roomID string, now time.Time) error {
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	roundID, number, qtext, deadline, err := OpenRound(ctx, tx, roomID, e.Content.Questions, now)
	if err != nil {
		return err
	}
	_ = roundID
	if err := tx.Commit(); err != nil {
		return err
	}

	EmitStateChanged(e.Pub, roomID, StateAnswering, number)
	e.Pub.Publish(roomID, events.Event{
		Name: "round.started",
		Data: map[string]any{
			"round_number":    number,
			"question_text":   qtext,
			"answer_deadline": deadline,
		},
	})
	return nil
}
