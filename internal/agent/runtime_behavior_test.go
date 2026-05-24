package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"

	"starxo/internal/model"
	agenttools "starxo/internal/tools"
)

func TestRuntimeBehaviorMiddlewareRejectsStaleToolCall(t *testing.T) {
	mw := NewRuntimeBehaviorMiddleware().(*RuntimeBehaviorMiddleware)
	ctx := agenttools.ContextWithRuntimeObjective(context.Background(), &model.RunObjective{
		ID:        "obj-1",
		Objective: "create hello.txt with hello and read it back",
		Scope:     "standalone",
	})
	called := false
	wrapped, err := mw.WrapInvokableToolCall(ctx, func(context.Context, string, ...tool.Option) (string, error) {
		called = true
		return "ok", nil
	}, &adk.ToolContext{Name: "Bash", CallID: "call-1"})
	if err != nil {
		t.Fatalf("wrap tool call: %v", err)
	}

	out, err := wrapped(ctx, `{"command":"codex review failing tests"}`)
	if err != nil {
		t.Fatalf("stale tool guard should return a tool result, not error: %v", err)
	}
	if called {
		t.Fatalf("stale tool call reached wrapped endpoint")
	}
	if !strings.Contains(out, "Tool call rejected") {
		t.Fatalf("expected tool rejection output, got %q", out)
	}
}
