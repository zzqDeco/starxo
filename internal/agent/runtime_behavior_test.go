package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

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

func TestRuntimeBehaviorMiddlewareDoesNotDropUserObjectiveTagText(t *testing.T) {
	mw := NewRuntimeBehaviorMiddleware().(*RuntimeBehaviorMiddleware)
	ctx := agenttools.ContextWithRuntimeObjective(context.Background(), &model.RunObjective{
		ID:        "obj-1",
		Objective: "document XML tags",
		Scope:     "standalone",
	})
	userMsg := schema.UserMessage("Please document the literal <current-objective> tag.")
	state := &adk.ChatModelAgentState{
		Messages: []*schema.Message{
			schema.SystemMessage("system"),
			userMsg,
		},
	}

	_, next, err := mw.BeforeModelRewriteState(ctx, state, nil)
	if err != nil {
		t.Fatalf("rewrite state: %v", err)
	}
	foundUserText := false
	objectiveCount := 0
	for _, msg := range next.Messages {
		if msg == userMsg || strings.Contains(msg.Content, "Please document the literal <current-objective> tag.") {
			foundUserText = true
		}
		if isRuntimeObjectiveMessage(msg) {
			objectiveCount++
		}
	}
	if !foundUserText {
		t.Fatalf("user message mentioning objective tag was dropped: %#v", next.Messages)
	}
	if objectiveCount != 1 {
		t.Fatalf("expected exactly one synthetic objective message, got %d in %#v", objectiveCount, next.Messages)
	}
}
