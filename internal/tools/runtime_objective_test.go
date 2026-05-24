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

func TestRuntimeObjectivePromptGuardAllowsContinuationReferences(t *testing.T) {
	ctx := ContextWithRuntimeObjective(context.Background(), &model.RunObjective{
		ID:        "obj-1",
		Objective: "继续",
		Scope:     "continuation",
	})

	if msg, ok := RuntimeObjectivePromptGuard(ctx, "Which failing tests should I debug first?"); !ok {
		t.Fatalf("expected continuation question to be allowed, msg=%q", msg)
	}
	if msg, ok := RuntimeObjectiveToolGuard(ctx, "Bash", `{"command":"go test ./..."}`); !ok {
		t.Fatalf("expected continuation tool call to be allowed, msg=%q", msg)
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

func TestRuntimeObjectiveGuardUsesWordBoundaries(t *testing.T) {
	ctx := ContextWithRuntimeObjective(context.Background(), &model.RunObjective{
		ID:        "obj-1",
		Objective: "create preview.txt and show it",
		Scope:     "standalone",
	})

	if msg, ok := RuntimeObjectiveToolGuard(ctx, "Bash", `{"command":"cat preview.txt"}`); !ok {
		t.Fatalf("expected preview to avoid matching review, msg=%q", msg)
	}
	if msg, ok := RuntimeObjectiveToolGuard(ctx, "Bash", `{"command":"git stage preview.txt"}`); !ok {
		t.Fatalf("expected stage to avoid matching tag, msg=%q", msg)
	}
}

func TestRuntimeObjectiveToolGuardAllowsPlanReviewText(t *testing.T) {
	ctx := ContextWithRuntimeObjective(context.Background(), &model.RunObjective{
		ID:        "obj-1",
		Objective: "implement a small file change",
		Scope:     "standalone",
	})

	args := `{"plan":"1. Review current files\n2. Edit hello.txt\n3. Run verification"}`
	if msg, ok := RuntimeObjectiveToolGuard(ctx, "ExitPlanMode", args); !ok {
		t.Fatalf("expected normal plan review wording to be allowed, msg=%q", msg)
	}
}

func TestRuntimeObjectiveToolGuardKeepsHighConfidenceReviewStale(t *testing.T) {
	ctx := ContextWithRuntimeObjective(context.Background(), &model.RunObjective{
		ID:        "obj-1",
		Objective: "implement a small file change",
		Scope:     "standalone",
	})

	if msg, ok := RuntimeObjectiveToolGuard(ctx, "Bash", `{"command":"codex review"}`); ok {
		t.Fatalf("expected codex review stale call to be rejected, msg=%q", msg)
	}
}
