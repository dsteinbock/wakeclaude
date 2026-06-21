package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSessionModelUsesLatestNonSyntheticAssistantModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	content := "" +
		`{"type":"assistant","message":{"role":"assistant","model":"claude-sonnet-4-6","content":[]}}` + "\n" +
		`{"type":"user","message":{"role":"user","content":"switch models"}}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-8","content":[]}}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","model":"<synthetic>","content":[]}}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := ExtractSessionModel(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "claude-opus-4-8" {
		t.Fatalf("model = %q, want %q", got, "claude-opus-4-8")
	}
}

func TestModelDisplayName(t *testing.T) {
	tests := map[string]string{
		"":                              "Session model unknown",
		"claude-fable-5":                "Claude Fable 5",
		"claude-opus-4-8":               "Claude Opus 4.8",
		"claude-haiku-4-5-20251001":     "Claude Haiku 4.5",
		"provider-specific-model-value": "provider-specific-model-value",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := ModelDisplayName(input); got != want {
				t.Fatalf("ModelDisplayName(%q) = %q, want %q", input, got, want)
			}
		})
	}
}
