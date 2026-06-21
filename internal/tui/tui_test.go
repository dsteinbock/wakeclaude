package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"wakeclaude/internal/app"
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

func TestDefaultModelOptionsExcludeDefaultAndBest(t *testing.T) {
	m := newModel(Input{TokenReady: true})
	want := []app.ModelOption{
		{Label: "Fable", Value: "fable"},
		{Label: "Opus", Value: "opus"},
		{Label: "Sonnet", Value: "sonnet"},
		{Label: "Haiku", Value: "haiku"},
	}

	if !reflect.DeepEqual(m.models, want) {
		t.Fatalf("models = %#v, want %#v", m.models, want)
	}
}

func TestResumedSessionSkipsModelSelection(t *testing.T) {
	m := newModel(Input{TokenReady: true})
	m.stage = stagePrompt
	m.selectedSess = &app.Session{
		ID:    "session-123",
		Path:  "/tmp/session-123.jsonl",
		Model: "claude-opus-4-8",
	}
	m.promptInput.SetValue("continue the work")
	m.promptInput.Focus()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	got := updated.(*model)

	if got.stage != stagePermissionMode {
		t.Fatalf("stage = %v, want stagePermissionMode", got.stage)
	}
	if got.modelLabel() != "Claude Opus 4.8" {
		t.Fatalf("model label = %q, want %q", got.modelLabel(), "Claude Opus 4.8")
	}
}

func TestNewSessionContinuesToModelSelection(t *testing.T) {
	m := newModel(Input{TokenReady: true})
	m.stage = stagePrompt
	m.selectedNew = true
	m.promptInput.SetValue("start the work")
	m.promptInput.Focus()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	got := updated.(*model)

	if got.stage != stageModels {
		t.Fatalf("stage = %v, want stageModels", got.stage)
	}
}
