package agentctx

import (
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"

	"starxo/internal/model"
)

// WindowConfig controls how conversation messages are windowed to fit
// within the model's context limits.
type WindowConfig struct {
	MaxMessages   int // max messages to keep in full (default 20)
	MaxContentLen int // max content length per message before truncation (default 4000)
}

// TokenWindowConfig controls token-aware prompt compaction.
type TokenWindowConfig struct {
	MaxTokens         int
	MaxContentLen     int
	MinRecentMessages int
}

// DefaultWindowConfig returns the default windowing configuration.
func DefaultWindowConfig() WindowConfig {
	return WindowConfig{
		MaxMessages:   20,
		MaxContentLen: 4000,
	}
}

func DefaultTokenWindowConfig() TokenWindowConfig {
	return TokenWindowConfig{
		MaxTokens:         8000,
		MaxContentLen:     4000,
		MinRecentMessages: 6,
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

// WindowMessagesTokenAwareWithPinnedPrefix preserves pinnedPrefix, injects a
// compact runtime summary when available, and keeps the newest history tail
// within an approximate token budget.
func WindowMessagesTokenAwareWithPinnedPrefix(pinnedPrefix []*schema.Message, history []*schema.Message, compact *model.RuntimeContextCompact, cfg TokenWindowConfig) []*schema.Message {
	defaults := DefaultTokenWindowConfig()
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = defaults.MaxTokens
	}
	if cfg.MaxContentLen <= 0 {
		cfg.MaxContentLen = defaults.MaxContentLen
	}
	if cfg.MinRecentMessages <= 0 {
		cfg.MinRecentMessages = defaults.MinRecentMessages
	}

	prefix := truncateAll(pinnedPrefix, cfg.MaxContentLen)
	result := make([]*schema.Message, 0, len(prefix)+len(history)+1)
	result = append(result, prefix...)

	compactMsg := runtimeCompactMessage(compact, cfg.MaxContentLen)
	if compactMsg != nil {
		result = append(result, compactMsg)
	}
	if len(history) == 0 {
		return result
	}

	remaining := cfg.MaxTokens - EstimateMessagesTokens(result)
	if remaining <= 0 {
		remaining = cfg.MaxTokens / 4
	}

	tailStart := len(history)
	tailTokens := 0
	minStart := len(history) - cfg.MinRecentMessages
	if minStart < 0 {
		minStart = 0
	}
	for i := len(history) - 1; i >= 0; i-- {
		msg := truncateMsg(history[i], cfg.MaxContentLen)
		msgTokens := EstimateMessageTokens(msg)
		if tailTokens+msgTokens > remaining && i < minStart {
			break
		}
		tailStart = i
		tailTokens += msgTokens
	}
	tailStart = adjustForToolCallGroups(history, tailStart)

	if tailStart > 0 && compactMsg == nil {
		result = append(result, schema.UserMessage(fmt.Sprintf("[Earlier conversation with %d messages omitted for brevity]", tailStart)))
	}
	for _, msg := range history[tailStart:] {
		result = append(result, truncateMsg(msg, cfg.MaxContentLen))
	}
	return result
}

// EstimateMessageTokens is a cheap, deterministic token approximation. It is
// intentionally conservative enough for windowing decisions without coupling
// this package to a tokenizer for every model.
func EstimateMessageTokens(msg *schema.Message) int {
	if msg == nil {
		return 0
	}
	tokens := estimateTextTokens(string(msg.Role)) + estimateTextTokens(msg.Name) + estimateTextTokens(msg.ToolCallID) + estimateTextTokens(msg.Content)
	for _, tc := range msg.ToolCalls {
		tokens += estimateTextTokens(tc.ID)
		tokens += estimateTextTokens(tc.Function.Name)
		tokens += estimateTextTokens(tc.Function.Arguments)
	}
	if tokens < 1 {
		return 1
	}
	return tokens
}

func EstimateMessagesTokens(messages []*schema.Message) int {
	total := 0
	for _, msg := range messages {
		total += EstimateMessageTokens(msg)
	}
	return total
}

func EstimatePersistedMessagesTokens(messages []model.PersistedMessage) int {
	total := 0
	for _, msg := range messages {
		total += estimateTextTokens(msg.Role)
		total += estimateTextTokens(msg.Name)
		total += estimateTextTokens(msg.ToolCallID)
		total += estimateTextTokens(msg.Content)
		for _, tc := range msg.ToolCalls {
			total += estimateTextTokens(tc.ID)
			total += estimateTextTokens(tc.Function.Name)
			total += estimateTextTokens(tc.Function.Arguments)
		}
	}
	return total
}

func estimateTextTokens(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	// A rough English/code average of 4 chars per token, plus a small message
	// overhead so many short turns are not undercounted.
	return len(text)/4 + 1
}

func runtimeCompactMessage(compact *model.RuntimeContextCompact, maxContentLen int) *schema.Message {
	if compact == nil {
		return nil
	}
	content := FormatRuntimeContextCompact(compact)
	if strings.TrimSpace(content) == "" {
		return nil
	}
	return schema.UserMessage(TruncateContent(content, maxContentLen))
}

func FormatRuntimeContextCompact(compact *model.RuntimeContextCompact) string {
	if compact == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("[Runtime context compact]\n")
	if compact.Summary != "" {
		b.WriteString(compact.Summary)
		b.WriteString("\n")
	}
	if compact.ActiveObjective != nil && strings.TrimSpace(compact.ActiveObjective.Objective) != "" {
		scope := compact.ActiveObjective.Scope
		if scope == "" {
			scope = "standalone"
		}
		b.WriteString("- Active objective: ")
		b.WriteString(oneLine(compact.ActiveObjective.Objective, 320))
		b.WriteString(" (")
		b.WriteString(scope)
		b.WriteString(")\n")
	}
	if compact.OmittedMessageCount > 0 {
		b.WriteString(fmt.Sprintf("Earlier conversation messages omitted from the prompt: %d of %d.\n", compact.OmittedMessageCount, compact.OriginalMessageCount))
	}
	if len(compact.ToolSearch.DiscoveredTools) > 0 {
		names := make([]string, 0, len(compact.ToolSearch.DiscoveredTools))
		for _, record := range compact.ToolSearch.DiscoveredTools {
			if record.CanonicalName != "" {
				names = append(names, record.CanonicalName)
			}
		}
		if len(names) > 0 {
			b.WriteString("- Discovered tools: ")
			b.WriteString(strings.Join(names, ", "))
			b.WriteString("\n")
		}
	}
	if len(compact.PermissionGrants) > 0 {
		names := make([]string, 0, len(compact.PermissionGrants))
		for _, grant := range compact.PermissionGrants {
			if grant.ToolName != "" {
				names = append(names, grant.ToolName)
			}
		}
		if len(names) > 0 {
			b.WriteString("- Session permission grants: ")
			b.WriteString(strings.Join(names, ", "))
			b.WriteString("\n")
		}
	}
	if compact.PlanDocument != nil && strings.TrimSpace(compact.PlanDocument.Markdown) != "" {
		b.WriteString("- Active plan: ")
		b.WriteString(oneLine(compact.PlanDocument.Markdown, 280))
		b.WriteString("\n")
	}
	if len(compact.Todos) > 0 {
		b.WriteString("- Todos: ")
		b.WriteString(formatTodoCounts(compact.Todos))
		b.WriteString("\n")
	}
	if compact.Workspace != nil && compact.Workspace.Active {
		b.WriteString("- Active worktree: ")
		b.WriteString(compact.Workspace.WorktreePath)
		if compact.Workspace.WorktreeBranch != "" {
			b.WriteString(" on ")
			b.WriteString(compact.Workspace.WorktreeBranch)
		}
		b.WriteString("\n")
	}
	if len(compact.Tasks) > 0 {
		b.WriteString("- Runtime tasks:\n")
		for _, task := range compact.Tasks {
			if task.ID == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("  - %s [%s] %s output=%s\n", task.ID, task.Status, oneLine(task.Description, 120), task.OutputPath))
		}
	}
	if len(compact.FileReadState) > 0 {
		b.WriteString("- Recent file reads:\n")
		for _, file := range compact.FileReadState {
			if file.FilePath == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("  - %s lines %d-%d/%d hash=%s\n", file.FilePath, file.StartLine, file.StartLine+max(0, file.NumLines)-1, file.TotalLines, file.ContentHash))
		}
	}
	if len(compact.DiffSummaries) > 0 {
		b.WriteString("- Recent edits:\n")
		for _, diff := range compact.DiffSummaries {
			if diff.FilePath == "" {
				continue
			}
			summary := diff.Summary
			if summary == "" {
				summary = fmt.Sprintf("%s +%d -%d", diff.ToolName, diff.LinesAdded, diff.LinesRemoved)
			}
			b.WriteString(fmt.Sprintf("  - %s: %s\n", diff.FilePath, oneLine(summary, 180)))
		}
	}
	return strings.TrimSpace(b.String())
}

func formatTodoCounts(todos []model.RuntimeTodoItem) string {
	counts := map[string]int{}
	for _, todo := range todos {
		counts[todo.Status]++
	}
	parts := make([]string, 0, 5)
	for _, status := range []string{"pending", "in_progress", "done", "failed", "blocked"} {
		if counts[status] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[status], status))
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d items", len(todos))
	}
	return strings.Join(parts, ", ")
}

func oneLine(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return TruncateContent(text, limit)
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
