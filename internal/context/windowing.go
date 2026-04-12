package agentctx

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// WindowConfig controls how conversation messages are windowed to fit
// within the model's context limits.
type WindowConfig struct {
	MaxMessages   int // max messages to keep in full (default 20)
	MaxContentLen int // max content length per message before truncation (default 4000)
}

// DefaultWindowConfig returns the default windowing configuration.
func DefaultWindowConfig() WindowConfig {
	return WindowConfig{
		MaxMessages:   20,
		MaxContentLen: 4000,
	}
}

// WindowMessages applies legacy windowing to a flat slice of messages.
// The first message is treated as pinned prefix and preserved.
func WindowMessages(messages []*schema.Message, cfg WindowConfig) []*schema.Message {
	if len(messages) == 0 {
		return messages
	}
	return WindowMessagesWithPinnedPrefix(messages[:1], messages[1:], cfg)
}

// WindowMessagesWithPinnedPrefix preserves pinnedPrefix in full and windows only the history tail.
func WindowMessagesWithPinnedPrefix(pinnedPrefix []*schema.Message, history []*schema.Message, cfg WindowConfig) []*schema.Message {
	if cfg.MaxMessages <= 0 {
		cfg.MaxMessages = DefaultWindowConfig().MaxMessages
	}
	if cfg.MaxContentLen <= 0 {
		cfg.MaxContentLen = DefaultWindowConfig().MaxContentLen
	}

	prefix := truncateAll(pinnedPrefix, cfg.MaxContentLen)
	if len(history) == 0 {
		return prefix
	}

	total := len(prefix) + len(history)

	// If within budget, just truncate long content.
	if total <= cfg.MaxMessages {
		result := make([]*schema.Message, 0, total)
		result = append(result, prefix...)
		result = append(result, truncateAll(history, cfg.MaxContentLen)...)
		return result
	}

	// Preserve the entire prefix and keep the newest history entries within the remaining budget.
	keepTail := cfg.MaxMessages - len(prefix)
	if keepTail < 1 {
		keepTail = 1
	}

	tailStart := len(history) - keepTail
	if tailStart < 0 {
		tailStart = 0
	}

	// Adjust tailStart to avoid splitting tool call groups.
	// A tool call group = assistant message with ToolCalls + subsequent tool result messages.
	tailStart = adjustForToolCallGroups(history, tailStart)

	omitted := tailStart

	result := make([]*schema.Message, 0, len(prefix)+1+keepTail)
	result = append(result, prefix...)

	if omitted > 0 {
		placeholder := schema.UserMessage(
			fmt.Sprintf("[Earlier conversation with %d messages omitted for brevity]", omitted),
		)
		result = append(result, placeholder)
	}

	for _, msg := range history[tailStart:] {
		result = append(result, truncateMsg(msg, cfg.MaxContentLen))
	}

	return result
}

// adjustForToolCallGroups ensures the window cut point does not land inside
// a tool call group (assistant message with ToolCalls + its tool result messages).
// If tailStart points to a tool result message, it moves backward to include
// the entire group.
func adjustForToolCallGroups(messages []*schema.Message, tailStart int) int {
	if tailStart <= 1 || tailStart >= len(messages) {
		return tailStart
	}
	// If the message at tailStart is a tool result, scan backward past all
	// consecutive tool results to find the group's assistant+ToolCalls start.
	for tailStart > 1 && messages[tailStart].Role == schema.Tool {
		tailStart--
	}
	return tailStart
}

// TruncateContent truncates content that exceeds maxLen, keeping the first 60%
// and last 20% with a marker in between.
func TruncateContent(content string, maxLen int) string {
	if maxLen <= 0 || len(content) <= maxLen {
		return content
	}

	marker := "...[truncated]..."
	markerLen := len(marker)

	// Minimum meaningful truncation needs room for marker + some content.
	if maxLen <= markerLen+10 {
		if maxLen > markerLen {
			return content[:maxLen-markerLen] + marker
		}
		return content[:maxLen]
	}

	available := maxLen - markerLen
	headLen := available * 60 / 100 // 60% for head
	tailLen := available * 20 / 100 // 20% for tail

	// Ensure we use remaining budget on the head.
	if headLen+tailLen+markerLen < maxLen {
		headLen = maxLen - markerLen - tailLen
	}

	head := content[:headLen]
	tail := content[len(content)-tailLen:]

	return head + marker + tail
}

// truncateAll returns a new slice with all messages truncated.
func truncateAll(messages []*schema.Message, maxContentLen int) []*schema.Message {
	out := make([]*schema.Message, len(messages))
	for i, msg := range messages {
		out[i] = truncateMsg(msg, maxContentLen)
	}
	return out
}

// truncateMsg returns a copy of the message with content truncated if needed.
// If no truncation is needed, the original pointer is returned.
func truncateMsg(msg *schema.Message, maxContentLen int) *schema.Message {
	if len(msg.Content) <= maxContentLen {
		return msg
	}
	cp := *msg
	cp.Content = TruncateContent(msg.Content, maxContentLen)
	return &cp
}
