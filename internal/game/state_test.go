package game

import "testing"

func TestRoomStateValid(t *testing.T) {
	all := []RoomState{StateLobby, StateCharcreate, StateAnswering, StateGuessing, StateScoring, StateWon, StateAbandoned}
	for _, s := range all {
		if !IsValidRoomState(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if IsValidRoomState("garbage") {
		t.Error("garbage should not validate")
	}
}

func TestRoomStateTerminal(t *testing.T) {
	if !IsTerminalRoomState(StateWon) || !IsTerminalRoomState(StateAbandoned) {
		t.Error("won/abandoned must be terminal")
	}
	for _, s := range []RoomState{StateLobby, StateCharcreate, StateAnswering, StateGuessing, StateScoring} {
		if IsTerminalRoomState(s) {
			t.Errorf("%s must not be terminal", s)
		}
	}
}

func TestCanTransitionRoom(t *testing.T) {
	cases := []struct {
		from, to RoomState
		ok       bool
	}{
		// happy edges
		{StateLobby, StateCharcreate, true},
		{StateLobby, StateAnswering, true},
		{StateCharcreate, StateAnswering, true},
		{StateAnswering, StateGuessing, true},
		{StateGuessing, StateScoring, true},
		{StateScoring, StateAnswering, true},
		{StateScoring, StateWon, true},

		// abandon from any non-terminal
		{StateLobby, StateAbandoned, true},
		{StateCharcreate, StateAbandoned, true},
		{StateAnswering, StateAbandoned, true},
		{StateGuessing, StateAbandoned, true},
		{StateScoring, StateAbandoned, true},

		// terminal sinks
		{StateWon, StateAnswering, false},
		{StateAbandoned, StateLobby, false},
		{StateWon, StateAbandoned, false},

		// no-op
		{StateLobby, StateLobby, false},

		// illegal jumps
		{StateLobby, StateGuessing, false},
		{StateAnswering, StateWon, false},
		{StateAnswering, StateScoring, false},
		{StateGuessing, StateAnswering, false},
		{StateGuessing, StateWon, false},
	}
	for _, c := range cases {
		got := CanTransitionRoom(c.from, c.to)
		if got != c.ok {
			t.Errorf("CanTransitionRoom(%s, %s) = %v, want %v", c.from, c.to, got, c.ok)
		}
	}
}

func TestRoundTransitions(t *testing.T) {
	cases := []struct {
		from, to RoundState
		ok       bool
	}{
		{RoundAnswering, RoundGuessing, true},
		{RoundGuessing, RoundScoring, true},
		{RoundScoring, RoundDone, true},
		{RoundAnswering, RoundScoring, false},
		{RoundDone, RoundAnswering, false},
		{RoundAnswering, RoundAnswering, false},
	}
	for _, c := range cases {
		got := CanTransitionRound(c.from, c.to)
		if got != c.ok {
			t.Errorf("CanTransitionRound(%s, %s) = %v, want %v", c.from, c.to, got, c.ok)
		}
	}
}

func TestValidateRoomTransitionError(t *testing.T) {
	err := ValidateRoomTransition(StateLobby, StateGuessing)
	if err == nil {
		t.Fatal("expected error for illegal transition")
	}
	if _, ok := err.(ErrInvalidTransition); !ok {
		t.Errorf("expected ErrInvalidTransition, got %T", err)
	}
}
