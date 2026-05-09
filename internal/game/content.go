// content.go — loads the default character pool and question bank from
// embedded JSON. The JSON files live under content/ at the repository root and
// are baked into the binary via embed.FS so the server has zero on-disk
// runtime asset dependencies.
package game

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed content/characters_default.json content/questions_default.json
var contentFS embed.FS

// CharacterTemplate is one entry in the curated character pool.
type CharacterTemplate struct {
	TemplateID string `json:"template_id"`
	Name       string `json:"name"`
	Blurb      string `json:"blurb"`
}

// QuestionTemplate is one entry in the curated question bank.
type QuestionTemplate struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Content is the parsed in-memory copy of the static JSON files. Treat as
// immutable once loaded.
type Content struct {
	Characters []CharacterTemplate
	Questions  []QuestionTemplate
}

// LoadDefaultContent parses the embedded JSON assets. Returns an error if the
// files are missing, malformed, or empty (any of which would make the game
// unrunnable).
func LoadDefaultContent() (*Content, error) {
	chars, err := loadCharacters()
	if err != nil {
		return nil, err
	}
	qs, err := loadQuestions()
	if err != nil {
		return nil, err
	}
	return &Content{Characters: chars, Questions: qs}, nil
}

func loadCharacters() ([]CharacterTemplate, error) {
	raw, err := contentFS.ReadFile("content/characters_default.json")
	if err != nil {
		return nil, fmt.Errorf("read characters_default.json: %w", err)
	}
	var out []CharacterTemplate
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse characters_default.json: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("characters_default.json is empty")
	}
	for i, c := range out {
		if c.TemplateID == "" || c.Name == "" || c.Blurb == "" {
			return nil, fmt.Errorf("character %d missing required field", i)
		}
	}
	return out, nil
}

func loadQuestions() ([]QuestionTemplate, error) {
	raw, err := contentFS.ReadFile("content/questions_default.json")
	if err != nil {
		return nil, fmt.Errorf("read questions_default.json: %w", err)
	}
	var out []QuestionTemplate
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse questions_default.json: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("questions_default.json is empty")
	}
	for i, q := range out {
		if q.ID == "" || q.Text == "" {
			return nil, fmt.Errorf("question %d missing required field", i)
		}
	}
	return out, nil
}
