package game

import "testing"

func TestLoadDefaultContent(t *testing.T) {
	c, err := LoadDefaultContent()
	if err != nil {
		t.Fatalf("LoadDefaultContent: %v", err)
	}
	if len(c.Characters) < 20 {
		t.Errorf("expected ≥20 characters, got %d", len(c.Characters))
	}
	if len(c.Questions) < 10 {
		t.Errorf("expected ≥10 questions, got %d", len(c.Questions))
	}
	// Spot-check shape.
	for i, ch := range c.Characters {
		if ch.TemplateID == "" || ch.Name == "" || ch.Blurb == "" {
			t.Fatalf("character %d malformed: %+v", i, ch)
		}
	}
	for i, q := range c.Questions {
		if q.ID == "" || q.Text == "" {
			t.Fatalf("question %d malformed: %+v", i, q)
		}
	}
}
