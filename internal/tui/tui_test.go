package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPromptStageAcceptsQAsText(t *testing.T) {
	m := newModel(Input{TokenReady: true})
	m.stage = stagePrompt
	m.promptInput.Focus()

	updated, _ := m.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'q'},
	})
	got := updated.(*model)

	if got.err != nil {
		t.Fatalf("typing q in prompt set error: %v", got.err)
	}
	if got.promptInput.Value() != "q" {
		t.Fatalf("prompt value = %q, want %q", got.promptInput.Value(), "q")
	}
}

func TestPromptStageHidesQuitHint(t *testing.T) {
	m := newModel(Input{TokenReady: true})
	m.stage = stagePrompt
	m.promptInput.Focus()

	view := m.View()

	if strings.Contains(view, "q quit") {
		t.Fatalf("prompt view contains quit hint:\n%s", view)
	}
	if !strings.Contains(view, "ctrl+d continue | esc back") {
		t.Fatalf("prompt view is missing prompt controls:\n%s", view)
	}
}
