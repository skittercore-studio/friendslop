// scoring.go — pure scoring logic: given a guess mapping
// {target_player_id -> guessed_character_id} and the true assignment
// {player_id -> character_id}, count matches.
//
// The guesser's own player_id MUST NOT appear in the mapping (self-row is
// excluded from the puzzle by design — see SPEC.md §1). ScoreGuess rejects
// any mapping that violates that invariant.
package game

import (
	"errors"
	"fmt"
)

// ErrSelfInMapping is returned when a guess mapping contains the guesser
// themselves as a target. Self is excluded from the puzzle by design.
var ErrSelfInMapping = errors.New("guess mapping must not contain the guesser as a target")

// ScoreGuess returns the number of correct (target -> character) pairs in
// `mapping` against the true `assignment` table.
//
// guesserID is the player who submitted the guess. Their own row must not
// appear in mapping.
//
// Entries in mapping whose target_player_id is not present in `assignment`
// are silently ignored (e.g. left players). Targets present in assignment
// but absent from mapping count as wrong (no contribution to correct_count).
func ScoreGuess(guesserID string, mapping map[string]string, assignment map[string]string) (int, error) {
	if _, found := mapping[guesserID]; found {
		return 0, ErrSelfInMapping
	}
	correct := 0
	for target, guessedChar := range mapping {
		if target == guesserID {
			return 0, ErrSelfInMapping
		}
		if trueChar, ok := assignment[target]; ok && trueChar == guessedChar {
			correct++
		}
	}
	return correct, nil
}

// IsPerfectGuess reports whether the guesser's mapping is fully correct over
// every other player in `assignment`. The guesser's own slot is excluded.
//
// Used to detect winners: at SCORING, the first guess (by submission time)
// that satisfies this for round R is the winner.
func IsPerfectGuess(guesserID string, mapping map[string]string, assignment map[string]string) bool {
	expected := 0
	for pid := range assignment {
		if pid == guesserID {
			continue
		}
		expected++
		guessed, ok := mapping[pid]
		if !ok {
			return false
		}
		if assignment[pid] != guessed {
			return false
		}
	}
	if len(mapping) != expected {
		return false
	}
	return true
}

// ValidateMapping enforces the structural invariants of a guess mapping:
//
//  1. mapping does not contain the guesser as a target,
//  2. mapping covers every other live player in the room (N-1 entries),
//  3. every target_player_id in mapping is a live player in the room,
//  4. every character_id in mapping is one of this room's pool characters,
//  5. the character_ids are all distinct (1:1 over the pool minus self... but
//     wait: per spec §4 "must be 1:1 over the pool minus your own". The
//     "your own" refers to the row, not the character. So characters MUST be
//     distinct across the N-1 entries — exactly one of the N pool characters
//     is unused, namely the one assigned to the guesser. We enforce
//     distinctness here.).
//
// Returns nil on success, otherwise a descriptive error suitable for surfacing
// as an HTTP 400 message.
func ValidateMapping(guesserID string, mapping map[string]string, otherPlayerIDs []string, poolCharacterIDs []string) error {
	if _, found := mapping[guesserID]; found {
		return ErrSelfInMapping
	}

	otherSet := make(map[string]struct{}, len(otherPlayerIDs))
	for _, p := range otherPlayerIDs {
		otherSet[p] = struct{}{}
	}
	poolSet := make(map[string]struct{}, len(poolCharacterIDs))
	for _, c := range poolCharacterIDs {
		poolSet[c] = struct{}{}
	}

	if len(mapping) != len(otherPlayerIDs) {
		return fmt.Errorf("mapping must contain %d entries (one per other player), got %d",
			len(otherPlayerIDs), len(mapping))
	}

	seenChar := make(map[string]struct{}, len(mapping))
	for target, char := range mapping {
		if _, ok := otherSet[target]; !ok {
			return fmt.Errorf("target %q is not a player in this room", target)
		}
		if _, ok := poolSet[char]; !ok {
			return fmt.Errorf("character %q is not in this room's pool", char)
		}
		if _, dup := seenChar[char]; dup {
			return fmt.Errorf("character %q used more than once in mapping", char)
		}
		seenChar[char] = struct{}{}
	}
	return nil
}
