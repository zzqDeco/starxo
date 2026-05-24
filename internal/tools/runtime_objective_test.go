package tools

import (
	"context"
	"strings"
	"testing"

	"starxo/internal/model"
)

func TestRuntimeObjectivePromptGuardRejectsStaleQuestion(t *testing.T) {
	ctx := ContextWithRuntimeObjective(context.Background(), &model.RunObjective{
		ID:        "obj-1",
		Objective: "create hello.txt with hello and read it back",
		Scope:     "standalone",
	})

	msg, ok := RuntimeObjectivePromptGuard(ctx, "Which failing tests should I debug next?")
	if ok {
		t.Fatalf("expected stale question to be rejected")
	}
	if !strings.Contains(msg, "current user objective") {
		t.Fatalf("expected actionable guard message, got %q", msg)
	}
}

func TestRuntimeObjectivePromptGuardAllowsRelevantQuestion(t *testing.T) {
	ctx := ContextWithRuntimeObjective(context.Background(), &model.RunObjective{
		ID:        "obj-1",
		Objective: "debug the failing tests in package alpha",
		Scope:     "standalone",
	})

	if msg, ok := RuntimeObjectivePromptGuard(ctx, "Which failing tests should I debug first?"); !ok {
		t.Fatalf("expected relevant question to be allowed, msg=%q", msg)
	}
}

func TestRuntimeObjectiveToolGuardRejectsStaleToolCall(t *testing.T) {
	ctx := ContextWithRuntimeObjective(context.Background(), &model.RunObjective{
		ID:        "obj-1",
		Objective: "create hello.txt with hello and read it back",
		Scope:     "standalone",
	})

	msg, ok := RuntimeObjectiveToolGuard(ctx, "Bash", `{"command":"codex review failing tests"}`)
	if ok {
		t.Fatalf("expected stale tool call to be rejected")
	}
	if !strings.Contains(msg, "Tool call rejected") {
		t.Fatalf("expected tool rejection message, got %q", msg)
	}
}
