package scheduler

import (
	"reflect"
	"testing"
)

func TestClaudeArgsUsesModelForNewSession(t *testing.T) {
	entry := ScheduleEntry{
		NewSession:     true,
		Model:          "fable",
		PermissionMode: "acceptEdits",
		Prompt:         "continue the work",
	}
	want := []string{"-p", "--model", "fable", "--permission-mode", "acceptEdits", "continue the work"}

	if got := claudeArgs(entry); !reflect.DeepEqual(got, want) {
		t.Fatalf("claudeArgs() = %#v, want %#v", got, want)
	}
}

func TestClaudeArgsDoesNotOverrideResumedSessionModel(t *testing.T) {
	entry := ScheduleEntry{
		SessionID:      "session-123",
		Model:          "opus",
		PermissionMode: "plan",
		Prompt:         "continue the work",
	}
	want := []string{"-p", "--permission-mode", "plan", "--resume", "session-123", "continue the work"}

	if got := claudeArgs(entry); !reflect.DeepEqual(got, want) {
		t.Fatalf("claudeArgs() = %#v, want %#v", got, want)
	}
}
