package agentctx

import (
	"testing"

	"github.com/cloudwego/eino/schema"

	"starxo/internal/model"
)

func TestRepairOrphanToolCallsInsertsResultAfterAssistantGroup(t *testing.T) {
	msgs := []*schema.Message{
		schema.UserMessage("debug tests"),
		{
			Role:    schema.Assistant,
			Content: "I need more input.",
			ToolCalls: []schema.ToolCall{{
				ID:       "call-ask",
				Function: schema.FunctionCall{Name: "ask_user", Arguments: `{}`},
			}},
		},
		schema.UserMessage("new standalone task"),
	}

	repaired, result := repairOrphanToolCallsWithReason(msgs, "Error: superseded")
	if result.Count != 1 || len(result.ToolCallIDs) != 1 || result.ToolCallIDs[0] != "call-ask" {
		t.Fatalf("unexpected repair result %#v", result)
	}
	assertProviderToolPairing(t, repaired)
	if len(repaired) != 4 {
		t.Fatalf("expected synthetic tool result to be inserted, got %#v", repaired)
	}
	if repaired[2].Role != schema.Tool || repaired[2].ToolCallID != "call-ask" || repaired[2].Content != "Error: superseded" {
		t.Fatalf("expected inserted tool result immediately after assistant call, got %#v", repaired[2])
	}
	if repaired[3].Role != schema.User || repaired[3].Content != "new standalone task" {
		t.Fatalf("expected user message to remain after repaired tool group, got %#v", repaired[3])
	}
}

func TestRepairOrphanToolCallsMovesNonAdjacentToolResult(t *testing.T) {
	msgs := []*schema.Message{
		{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{{
				ID:       "call-late",
				Function: schema.FunctionCall{Name: "ask_user", Arguments: `{}`},
			}},
		},
		schema.UserMessage("new task"),
		{Role: schema.Tool, ToolCallID: "call-late", Content: "Error: old appended repair"},
	}

	repaired, result := repairOrphanToolCallsWithReason(msgs, "Error: synthetic")
	if result.Count != 1 || result.ToolCallIDs[0] != "call-late" {
		t.Fatalf("unexpected repair result %#v", result)
	}
	assertProviderToolPairing(t, repaired)
	if len(repaired) != 3 {
		t.Fatalf("expected late result to be moved instead of duplicated, got %#v", repaired)
	}
	if repaired[1].Role != schema.Tool || repaired[1].Content != "Error: old appended repair" {
		t.Fatalf("expected old tool result to be moved after assistant call, got %#v", repaired[1])
	}
	if repaired[2].Role != schema.User {
		t.Fatalf("expected user message after repaired group, got %#v", repaired[2])
	}
}

func TestRepairOrphanToolCallsHandlesPartialMultiToolGroup(t *testing.T) {
	msgs := []*schema.Message{
		{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{ID: "call-a", Function: schema.FunctionCall{Name: "Read", Arguments: `{}`}},
				{ID: "call-b", Function: schema.FunctionCall{Name: "Bash", Arguments: `{}`}},
			},
		},
		{Role: schema.Tool, ToolCallID: "call-b", Content: "b result"},
	}

	repaired, result := repairOrphanToolCallsWithReason(msgs, "Error: missing")
	if result.Count != 1 || result.ToolCallIDs[0] != "call-a" {
		t.Fatalf("expected only missing call-a to be repaired, got %#v", result)
	}
	assertProviderToolPairing(t, repaired)
	if len(repaired) != 3 {
		t.Fatalf("expected assistant plus two tool results, got %#v", repaired)
	}
	if repaired[1].ToolCallID != "call-a" || repaired[2].ToolCallID != "call-b" {
		t.Fatalf("expected tool results in tool call order, got %#v", repaired)
	}
}

func TestPrepareMessagesWithCompactFromRepairsSlicedHistory(t *testing.T) {
	engine := NewEngine("system", 8000)
	engine.ImportMessages([]model.PersistedMessage{
		{Role: string(schema.User), Content: "old task"},
		{
			Role:    string(schema.Assistant),
			Content: "need input",
			ToolCalls: []model.PersistedToolCall{{
				ID: "call-old",
				Function: model.PersistedToolCallFunction{
					Name:      "ask_user",
					Arguments: `{}`,
				},
			}},
		},
		{Role: string(schema.User), Content: "new standalone task"},
	})

	msgs := engine.PrepareMessagesWithCompactFrom(nil, nil, 2)
	assertProviderToolPairing(t, msgs)
	for _, msg := range msgs {
		if msg.Role == schema.Tool && msg.ToolCallID == "call-old" {
			t.Fatalf("expected sliced prompt view to drop stale stray tool result, got %#v", msgs)
		}
	}
	if !messagesContainContent(msgs, "new standalone task") {
		t.Fatalf("expected prompt to keep current user request, got %#v", msgs)
	}
}

func assertProviderToolPairing(t *testing.T, msgs []*schema.Message) {
	t.Helper()
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		if msg == nil {
			continue
		}
		if msg.Role == schema.Tool && msg.ToolCallID != "" {
			t.Fatalf("tool message %q at index %d is not immediately attached to an assistant tool_call group: %#v", msg.ToolCallID, i, msgs)
		}
		if len(msg.ToolCalls) == 0 {
			continue
		}
		for _, tc := range msg.ToolCalls {
			i++
			if i >= len(msgs) {
				t.Fatalf("missing tool result for %q at end of messages: %#v", tc.ID, msgs)
			}
			result := msgs[i]
			if result == nil || result.Role != schema.Tool || result.ToolCallID != tc.ID {
				t.Fatalf("expected tool result for %q at index %d, got %#v in %#v", tc.ID, i, result, msgs)
			}
		}
	}
}

func messagesContainContent(msgs []*schema.Message, content string) bool {
	for _, msg := range msgs {
		if msg != nil && msg.Content == content {
			return true
		}
	}
	return false
}
