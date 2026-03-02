package agentctx

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultWindowConfig(t *testing.T) {
	cfg := DefaultWindowConfig()
	assert.Equal(t, 20, cfg.MaxMessages)
	assert.Equal(t, 4000, cfg.MaxContentLen)
}

func TestWindowMessagesEmpty(t *testing.T) {
	result := WindowMessages(nil, DefaultWindowConfig())
	assert.Nil(t, result)

	result = WindowMessages([]*schema.Message{}, DefaultWindowConfig())
	assert.Empty(t, result)
}

func TestWindowMessagesWithinBudget(t *testing.T) {
	msgs := make([]*schema.Message, 5)
	for i := range msgs {
		msgs[i] = &schema.Message{Role: schema.User, Content: "hello"}
	}

	result := WindowMessages(msgs, WindowConfig{MaxMessages: 10, MaxContentLen: 4000})
	assert.Len(t, result, 5)
	for _, m := range result {
		assert.Equal(t, "hello", m.Content)
	}
}

func TestWindowMessagesExceedsBudget(t *testing.T) {
	msgs := make([]*schema.Message, 10)
	msgs[0] = &schema.Message{Role: schema.System, Content: "system prompt"}
	for i := 1; i < 10; i++ {
		msgs[i] = &schema.Message{Role: schema.User, Content: strings.Repeat("x", i)}
	}

	cfg := WindowConfig{MaxMessages: 4, MaxContentLen: 4000}
	result := WindowMessages(msgs, cfg)

	// first message (system) + placeholder + last 3 messages
	require.True(t, len(result) >= 4)
	assert.Equal(t, schema.System, result[0].Role)
	assert.Equal(t, "system prompt", result[0].Content)

	// placeholder message
	assert.Equal(t, schema.User, result[1].Role)
	assert.Contains(t, result[1].Content, "omitted")

	// last messages preserved
	lastResult := result[len(result)-1]
	lastOriginal := msgs[len(msgs)-1]
	assert.Equal(t, lastOriginal.Content, lastResult.Content)
}

func TestWindowMessagesDefaultsOnZeroConfig(t *testing.T) {
	msgs := make([]*schema.Message, 3)
	for i := range msgs {
		msgs[i] = &schema.Message{Role: schema.User, Content: "ok"}
	}

	result := WindowMessages(msgs, WindowConfig{})
	assert.Len(t, result, 3)
}

func TestWindowMessagesTruncatesLongContent(t *testing.T) {
	longContent := strings.Repeat("a", 5000)
	msgs := []*schema.Message{
		{Role: schema.User, Content: longContent},
	}

	cfg := WindowConfig{MaxMessages: 10, MaxContentLen: 100}
	result := WindowMessages(msgs, cfg)

	require.Len(t, result, 1)
	assert.True(t, len(result[0].Content) <= 100)
	assert.Contains(t, result[0].Content, "...[truncated]...")
}

func TestWindowMessagesPreservesToolCallGroups(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.System, Content: "system"},
		{Role: schema.User, Content: "msg1"},
		{Role: schema.User, Content: "msg2"},
		{Role: schema.User, Content: "msg3"},
		{Role: schema.Assistant, Content: "thinking", ToolCalls: []schema.ToolCall{{ID: "t1"}}},
		{Role: schema.Tool, Content: "result1", ToolCallID: "t1"},
		{Role: schema.User, Content: "msg4"},
		{Role: schema.User, Content: "msg5"},
	}

	// MaxMessages=4 would normally cut at index 4 (the assistant+ToolCalls msg)
	// but adjustForToolCallGroups should keep the group together
	cfg := WindowConfig{MaxMessages: 4, MaxContentLen: 4000}
	result := WindowMessages(msgs, cfg)

	// Verify no tool result message appears without its tool call
	for i, m := range result {
		if m.Role == schema.Tool {
			require.True(t, i > 0, "tool result should not be first message")
			prev := result[i-1]
			assert.Equal(t, schema.Assistant, prev.Role, "tool result should follow an assistant message")
		}
	}
}

func TestWindowMessagesToolResultAtCutPoint(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.System, Content: "system"},
		{Role: schema.User, Content: "old msg"},
		{Role: schema.Assistant, Content: "", ToolCalls: []schema.ToolCall{{ID: "t1"}, {ID: "t2"}}},
		{Role: schema.Tool, Content: "r1", ToolCallID: "t1"},
		{Role: schema.Tool, Content: "r2", ToolCallID: "t2"},
		{Role: schema.User, Content: "recent"},
	}

	// MaxMessages=3: tailStart would be at index 3 (a tool result)
	// Should back up to index 2 (the assistant+ToolCalls)
	cfg := WindowConfig{MaxMessages: 3, MaxContentLen: 4000}
	result := WindowMessages(msgs, cfg)

	// The assistant message with ToolCalls should be included
	hasAssistantWithTools := false
	for _, m := range result {
		if m.Role == schema.Assistant && len(m.ToolCalls) > 0 {
			hasAssistantWithTools = true
		}
	}
	assert.True(t, hasAssistantWithTools, "assistant message with ToolCalls should be preserved with its group")
}

// --- TruncateContent tests ---

func TestTruncateContentShortString(t *testing.T) {
	result := TruncateContent("short", 100)
	assert.Equal(t, "short", result)
}

func TestTruncateContentExactLength(t *testing.T) {
	s := strings.Repeat("x", 100)
	result := TruncateContent(s, 100)
	assert.Equal(t, s, result)
}

func TestTruncateContentZeroMaxLen(t *testing.T) {
	result := TruncateContent("anything", 0)
	assert.Equal(t, "anything", result)
}

func TestTruncateContentNegativeMaxLen(t *testing.T) {
	result := TruncateContent("anything", -1)
	assert.Equal(t, "anything", result)
}

func TestTruncateContentLongString(t *testing.T) {
	content := strings.Repeat("a", 1000)
	result := TruncateContent(content, 200)

	assert.True(t, len(result) <= 200, "result length %d exceeds maxLen 200", len(result))
	assert.Contains(t, result, "...[truncated]...")
	assert.True(t, strings.HasPrefix(result, "aaa"), "should start with original content")
	assert.True(t, strings.HasSuffix(result, "aaa"), "should end with original content")
}

func TestTruncateContentVerySmallMaxLen(t *testing.T) {
	content := strings.Repeat("x", 100)

	// maxLen smaller than marker
	result := TruncateContent(content, 5)
	assert.Len(t, result, 5)
	assert.Equal(t, "xxxxx", result)

	// maxLen just slightly larger than marker
	result = TruncateContent(content, 20)
	assert.True(t, len(result) <= 20)
}

func TestTruncateContentPreservesHeadAndTail(t *testing.T) {
	// Create content with distinct head and tail
	head := strings.Repeat("H", 500)
	tail := strings.Repeat("T", 500)
	content := head + tail

	result := TruncateContent(content, 200)
	assert.True(t, strings.HasPrefix(result, "HHH"), "should preserve head content")
	assert.True(t, strings.HasSuffix(result, "TTT"), "should preserve tail content")
}

// --- adjustForToolCallGroups tests ---

func TestAdjustForToolCallGroupsNoToolMessages(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.System},
		{Role: schema.User},
		{Role: schema.Assistant},
		{Role: schema.User},
	}
	assert.Equal(t, 2, adjustForToolCallGroups(msgs, 2))
}

func TestAdjustForToolCallGroupsAtToolResult(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.System},
		{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{ID: "t1"}}},
		{Role: schema.Tool, ToolCallID: "t1"},
		{Role: schema.User},
	}
	// tailStart=2 points to tool result, should back up to 1 (assistant)
	assert.Equal(t, 1, adjustForToolCallGroups(msgs, 2))
}

func TestAdjustForToolCallGroupsMultipleToolResults(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.System},
		{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{ID: "t1"}, {ID: "t2"}}},
		{Role: schema.Tool, ToolCallID: "t1"},
		{Role: schema.Tool, ToolCallID: "t2"},
		{Role: schema.User},
	}
	// tailStart=3 points to second tool result
	assert.Equal(t, 1, adjustForToolCallGroups(msgs, 3))
	// tailStart=2 points to first tool result
	assert.Equal(t, 1, adjustForToolCallGroups(msgs, 2))
}

func TestAdjustForToolCallGroupsBoundary(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.System},
		{Role: schema.User},
	}
	// tailStart <= 1 should be unchanged
	assert.Equal(t, 0, adjustForToolCallGroups(msgs, 0))
	assert.Equal(t, 1, adjustForToolCallGroups(msgs, 1))
	// tailStart >= len should be unchanged
	assert.Equal(t, 2, adjustForToolCallGroups(msgs, 2))
	assert.Equal(t, 5, adjustForToolCallGroups(msgs, 5))
}
