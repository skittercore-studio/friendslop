package game

import (
	"errors"
	"testing"
)

func TestScoreGuess_Perfect(t *testing.T) {
	assignment := map[string]string{
		"p1": "c1",
		"p2": "c2",
		"p3": "c3",
		"p4": "c4",
	}
	mapping := map[string]string{
		"p2": "c2",
		"p3": "c3",
		"p4": "c4",
	}
	got, err := ScoreGuess("p1", mapping, assignment)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != 3 {
		t.Errorf("want 3 correct, got %d", got)
	}
	if !IsPerfectGuess("p1", mapping, assignment) {
		t.Error("expected perfect guess")
	}
}

func TestScoreGuess_Mixed(t *testing.T) {
	assignment := map[string]string{"p1": "c1", "p2": "c2", "p3": "c3", "p4": "c4"}
	mapping := map[string]string{
		"p2": "c2", // right
		"p3": "c4", // wrong
		"p4": "c3", // wrong
	}
	got, _ := ScoreGuess("p1", mapping, assignment)
	if got != 1 {
		t.Errorf("want 1 correct, got %d", got)
	}
	if IsPerfectGuess("p1", mapping, assignment) {
		t.Error("must not report perfect on partial")
	}
}

func TestScoreGuess_RejectsSelf(t *testing.T) {
	assignment := map[string]string{"p1": "c1", "p2": "c2"}
	mapping := map[string]string{"p1": "c1", "p2": "c2"}
	_, err := ScoreGuess("p1", mapping, assignment)
	if !errors.Is(err, ErrSelfInMapping) {
		t.Errorf("expected ErrSelfInMapping, got %v", err)
	}
}

func TestIsPerfectGuess_MissingTarget(t *testing.T) {
	assignment := map[string]string{"p1": "c1", "p2": "c2", "p3": "c3"}
	mapping := map[string]string{"p2": "c2"} // missing p3
	if IsPerfectGuess("p1", mapping, assignment) {
		t.Error("incomplete mapping cannot be perfect")
	}
}

func TestValidateMapping_OK(t *testing.T) {
	err := ValidateMapping("p1",
		map[string]string{"p2": "c2", "p3": "c3", "p4": "c4"},
		[]string{"p2", "p3", "p4"},
		[]string{"c1", "c2", "c3", "c4"},
	)
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
}

func TestValidateMapping_RejectsSelf(t *testing.T) {
	err := ValidateMapping("p1",
		map[string]string{"p1": "c1", "p2": "c2", "p3": "c3"},
		[]string{"p2", "p3"},
		[]string{"c1", "c2", "c3"},
	)
	if !errors.Is(err, ErrSelfInMapping) {
		t.Errorf("expected ErrSelfInMapping, got %v", err)
	}
}

func TestValidateMapping_IncompleteCoverage(t *testing.T) {
	err := ValidateMapping("p1",
		map[string]string{"p2": "c2"}, // missing p3
		[]string{"p2", "p3"},
		[]string{"c1", "c2", "c3"},
	)
	if err == nil {
		t.Error("expected incomplete coverage error")
	}
}

func TestValidateMapping_DuplicateChar(t *testing.T) {
	err := ValidateMapping("p1",
		map[string]string{"p2": "c2", "p3": "c2"},
		[]string{"p2", "p3"},
		[]string{"c1", "c2", "c3"},
	)
	if err == nil {
		t.Error("expected duplicate char error")
	}
}

func TestValidateMapping_UnknownTarget(t *testing.T) {
	err := ValidateMapping("p1",
		map[string]string{"px": "c2", "p3": "c3"},
		[]string{"p2", "p3"},
		[]string{"c1", "c2", "c3"},
	)
	if err == nil {
		t.Error("expected unknown target error")
	}
}

func TestValidateMapping_UnknownChar(t *testing.T) {
	err := ValidateMapping("p1",
		map[string]string{"p2": "c2", "p3": "cZ"},
		[]string{"p2", "p3"},
		[]string{"c1", "c2", "c3"},
	)
	if err == nil {
		t.Error("expected unknown character error")
	}
}
