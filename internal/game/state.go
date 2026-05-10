// Package game implements the friendslop game state machine, scoring,
// assignment, round lifecycle, content loading, and the HTTP handlers that
// mutate game state.
//
// Wiring: cmd/friendslop/main.go calls game.Setup(s, ctx) after the server
// has been constructed but before Mount. Setup loads default content, starts
// the timer goroutine, and registers route callbacks via s.AddRoutes.
package game

import (
	"errors"
	"fmt"
)

// RoomState mirrors the `state` column of the rooms table. The string values
// match SPEC.md §3 exactly — stored as TEXT in SQLite.
type RoomState string

const (
	StateLobby      RoomState = "lobby"
	StateCharcreate RoomState = "charcreate"
	StateAnswering  RoomState = "answering"
	StateGuessing   RoomState = "guessing"
	StateScoring    RoomState = "scoring"
	StateWon        RoomState = "won"
	StateAbandoned  RoomState = "abandoned"
)

// IsValidRoomState reports whether s is one of the known room states.
func IsValidRoomState(s RoomState) bool {
	switch s {
	case StateLobby, StateCharcreate, StateAnswering, StateGuessing,
		StateScoring, StateWon, StateAbandoned:
		return true
	}
	return false
}

// IsTerminalRoomState reports whether s represents an end-of-life room.
// No further mutations should happen once a room enters one of these states.
func IsTerminalRoomState(s RoomState) bool {
	return s == StateWon || s == StateAbandoned
}

// validRoomTransitions enumerates legal RoomState edges. Transitions to
// StateAbandoned are allowed from any non-terminal state; that case is
// expressed in CanTransitionRoom rather than enumerated here.
var validRoomTransitions = map[RoomState]map[RoomState]bool{
	StateLobby: {
		StateCharcreate: true, // pool_source = playerwritten
		StateAnswering:  true, // pool_source = curated
	},
	StateCharcreate: {
		StateAnswering: true,
	},
	StateAnswering: {
		StateGuessing: true,
	},
	StateGuessing: {
		StateScoring: true,
	},
	StateScoring: {
		StateAnswering: true, // next round
		StateWon:       true,
	},
}

// CanTransitionRoom reports whether moving the room from `from` to `to` is
// allowed. Abandoning is permitted from any non-terminal state. Terminal
// states (won, abandoned) are sinks except for the explicit /restart edge
// back to lobby — that's the only way out, and it's host-driven.
func CanTransitionRoom(from, to RoomState) bool {
	if from == to {
		return false
	}
	if IsTerminalRoomState(from) {
		// Restart: terminal states may transition back to lobby only.
		return to == StateLobby
	}
	if to == StateAbandoned {
		return true
	}
	allowed, ok := validRoomTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// ErrInvalidTransition is returned when a state transition is rejected.
type ErrInvalidTransition struct {
	From RoomState
	To   RoomState
}

func (e ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid room state transition: %s -> %s", e.From, e.To)
}

// ValidateRoomTransition wraps CanTransitionRoom in error form.
func ValidateRoomTransition(from, to RoomState) error {
	if !CanTransitionRoom(from, to) {
		return ErrInvalidTransition{From: from, To: to}
	}
	return nil
}

// RoundState mirrors the `state` column of the rounds table.
type RoundState string

const (
	RoundAnswering RoundState = "answering"
	RoundGuessing  RoundState = "guessing"
	RoundScoring   RoundState = "scoring"
	RoundDone      RoundState = "done"
)

// validRoundTransitions captures legal round substate moves.
var validRoundTransitions = map[RoundState]map[RoundState]bool{
	RoundAnswering: {RoundGuessing: true},
	RoundGuessing:  {RoundScoring: true},
	RoundScoring:   {RoundDone: true},
}

// CanTransitionRound reports whether the round may move from from -> to.
func CanTransitionRound(from, to RoundState) bool {
	if from == to {
		return false
	}
	allowed, ok := validRoundTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// ValidateRoundTransition wraps CanTransitionRound as an error.
func ValidateRoundTransition(from, to RoundState) error {
	if !CanTransitionRound(from, to) {
		return fmt.Errorf("invalid round state transition: %s -> %s", from, to)
	}
	return nil
}

// MinPlayers is the hard minimum player count enforced at /start.
// PLAYTEST: lowered from 4 to 2 for early playtesting. RESTORE TO 4 BEFORE PUBLIC LAUNCH.
const MinPlayers = 2

// ErrTooFewPlayers indicates a /start was attempted below MinPlayers.
var ErrTooFewPlayers = errors.New("at least 2 players required to start")
