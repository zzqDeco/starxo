package agentctx

import (
	"sort"
	"sync"

	"github.com/cloudwego/eino/schema"

	"starxo/internal/model"
)

// Engine coordinates all context sources (conversation history, file context)
// and prepares the message list for the LLM.
type Engine struct {
	mu           sync.RWMutex
	history      *ConversationHistory
	fileCtx      *FileContext
	maxTokens    int // approximate token budget for context window
	systemPrompt string
}

// RepairResult summarizes synthetic or moved tool results inserted to make
// assistant tool_call history valid for OpenAI-compatible providers.
type RepairResult struct {
	Count       int
	ToolCallIDs []string
}

// NewEngine creates a new context engine with the given system prompt and
// approximate token budget.
func NewEngine(systemPrompt string, maxTokens int) *Engine {
	return &Engine{
		history:      NewConversationHistory(),
		fileCtx:      NewFileContext(),
		maxTokens:    maxTokens,
		systemPrompt: systemPrompt,
	}
}

// AddUserMessage appends a user message to the conversation history.
func (e *Engine) AddUserMessage(content string) {
	e.history.Add(schema.UserMessage(content))
}

// AddAssistantMessage appends an assistant message to the conversation history.
func (e *Engine) AddAssistantMessage(content string) {
	e.history.Add(schema.AssistantMessage(content, nil))
}

// AddToolResult appends a tool result message to the conversation history.
func (e *Engine) AddToolResult(toolCallID, content string) {
	msg := &schema.Message{
		Role:       schema.Tool,
		Content:    content,
		ToolCallID: toolCallID,
	}
	e.history.Add(msg)
}

// PrepareMessages builds the full message list for the agent:
//  1. System message (with file context injected if files are present)
//  2. Windowed conversation history (recent messages kept, older ones summarized)
func (e *Engine) PrepareMessages() []*schema.Message {
	return e.PrepareMessagesWithPinnedPrefix(nil)
}

// PrepareMessagesWithPinnedPrefix builds the full message list for the agent:
//  1. System message
//  2. Synthetic pinned prefix messages
//  3. Windowed conversation history
func (e *Engine) PrepareMessagesWithPinnedPrefix(pinnedPrefix []*schema.Message) []*schema.Message {
	return e.PrepareMessagesWithCompact(pinnedPrefix, nil)
}

// PrepareMessagesWithCompact builds the full message list using token-aware
// windowing and an optional runtime compact summary.
func (e *Engine) PrepareMessagesWithCompact(pinnedPrefix []*schema.Message, compact *model.RuntimeContextCompact) []*schema.Message {
	return e.PrepareMessagesWithCompactFrom(pinnedPrefix, compact, 0)
}

// PrepareMessagesWithCompactFrom builds the prompt from a bounded history slice.
// historyStart is used by runtime objective isolation so a standalone user turn
// cannot accidentally continue older unrelated work.
func (e *Engine) PrepareMessagesWithCompactFrom(pinnedPrefix []*schema.Message, compact *model.RuntimeContextCompact, historyStart int) []*schema.Message {
	e.mu.RLock()
	sysPrompt := e.systemPrompt
	maxTokens := e.maxTokens
	e.mu.RUnlock()

	// Build the system message, optionally enriched with file context.
	fileDesc := e.fileCtx.FormatForSystemMessage()
	systemContent := sysPrompt
	if fileDesc != "" {
		systemContent = sysPrompt + "\n\n" + fileDesc
	}

	sysMsg := schema.SystemMessage(systemContent)

	// Get conversation history and apply windowing.
	historyMsgs := e.history.GetAll()
	if historyStart > 0 {
		if historyStart > len(historyMsgs) {
			historyStart = len(historyMsgs)
		}
		historyMsgs = historyMsgs[historyStart:]
	}
	historyMsgs, _ = repairOrphanToolCallsWithReason(historyMsgs, "Error: tool execution was interrupted before this request")

	prefix := make([]*schema.Message, 0, 1+len(pinnedPrefix))
	prefix = append(prefix, sysMsg)
	prefix = append(prefix, pinnedPrefix...)

	cfg := DefaultTokenWindowConfig()
	if maxTokens > 0 {
		cfg.MaxTokens = maxTokens
	}

	return WindowMessagesTokenAwareWithPinnedPrefix(prefix, historyMsgs, compact, cfg)
}

// FileContext returns the file context manager.
func (e *Engine) FileContext() *FileContext {
	return e.fileCtx
}

// History returns the conversation history manager.
func (e *Engine) History() *ConversationHistory {
	return e.history
}

// ClearHistory resets the conversation history.
func (e *Engine) ClearHistory() {
	e.history.Clear()
}

// SessionValues returns session metadata suitable for ADK runner options.
// Keys returned:
//   - "workspace_files": list of workspace file names
//   - "uploaded_files": list of recently uploaded file names
func (e *Engine) SessionValues() map[string]any {
	wsFiles := e.fileCtx.GetWorkspaceFiles()
	upFiles := e.fileCtx.GetUploadedFiles()

	wsNames := make([]string, len(wsFiles))
	for i, f := range wsFiles {
		wsNames[i] = f.Name
	}

	upNames := make([]string, len(upFiles))
	for i, f := range upFiles {
		upNames[i] = f.Name
	}

	return map[string]any{
		"workspace_files": wsNames,
		"uploaded_files":  upNames,
	}
}

// AddMessage appends a complete message (including ToolCalls) to the conversation history.
func (e *Engine) AddMessage(msg *schema.Message) {
	e.history.Add(msg)
}

// ExportMessages converts the conversation history to a serializable format for persistence.
func (e *Engine) ExportMessages() []model.PersistedMessage {
	msgs := e.history.GetAll()
	result := make([]model.PersistedMessage, 0, len(msgs))
	for _, msg := range msgs {
		pm := model.PersistedMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}
		for _, tc := range msg.ToolCalls {
			pm.ToolCalls = append(pm.ToolCalls, model.PersistedToolCall{
				ID: tc.ID,
				Function: model.PersistedToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		result = append(result, pm)
	}
	return result
}

// ImportMessages restores conversation history from persisted messages.
// Clears existing history first. Also repairs orphaned tool_calls that
// were persisted without matching tool results (e.g., due to mid-stream crash).
func (e *Engine) ImportMessages(messages []model.PersistedMessage) RepairResult {
	msgs := make([]*schema.Message, 0, len(messages))
	for _, pm := range messages {
		msg := &schema.Message{
			Role:       schema.RoleType(pm.Role),
			Content:    pm.Content,
			Name:       pm.Name,
			ToolCallID: pm.ToolCallID,
		}
		for _, ptc := range pm.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, schema.ToolCall{
				ID: ptc.ID,
				Function: schema.FunctionCall{
					Name:      ptc.Function.Name,
					Arguments: ptc.Function.Arguments,
				},
			})
		}
		msgs = append(msgs, msg)
	}

	msgs, repair := repairOrphanToolCallsWithReason(msgs, "Error: tool execution was interrupted before the session was restored")
	e.history.SetAll(msgs)
	return repair
}

// RepairOrphanToolCalls fixes the persisted history in place so assistant
// tool_calls are immediately followed by matching tool results.
func (e *Engine) RepairOrphanToolCalls(reason string) RepairResult {
	msgs := e.history.GetAll()
	repaired, result := repairOrphanToolCallsWithReason(msgs, reason)
	if result.Count > 0 {
		e.history.SetAll(repaired)
	}
	return result
}

func repairOrphanToolCalls(msgs []*schema.Message) []*schema.Message {
	repaired, _ := repairOrphanToolCallsWithReason(msgs, "Error: tool execution was interrupted")
	return repaired
}

type toolResultRef struct {
	index int
	msg   *schema.Message
}

// repairOrphanToolCallsWithReason enforces the OpenAI-compatible invariant that
// every assistant message with tool_calls is immediately followed by one tool
// message for each tool_call_id. Existing non-adjacent tool results are moved
// into the correct position; missing results are synthesized.
func repairOrphanToolCallsWithReason(msgs []*schema.Message, reason string) ([]*schema.Message, RepairResult) {
	if len(msgs) == 0 {
		return msgs, RepairResult{}
	}
	if reason == "" {
		reason = "Error: tool execution was interrupted"
	}

	toolResults := make(map[string][]toolResultRef)
	for i, msg := range msgs {
		if msg == nil || msg.Role != schema.Tool || msg.ToolCallID == "" {
			continue
		}
		toolResults[msg.ToolCallID] = append(toolResults[msg.ToolCallID], toolResultRef{index: i, msg: msg})
	}

	usedToolResults := make(map[int]bool)
	out := make([]*schema.Message, 0, len(msgs))
	result := RepairResult{}
	recordRepair := func(id string) {
		if id == "" {
			return
		}
		result.Count++
		result.ToolCallIDs = append(result.ToolCallIDs, id)
	}

	for i := 0; i < len(msgs); i++ {
		if usedToolResults[i] {
			continue
		}
		msg := msgs[i]
		if msg == nil {
			continue
		}
		if len(msg.ToolCalls) == 0 {
			if msg.Role == schema.Tool && msg.ToolCallID != "" {
				// A tool message outside an assistant tool_call group is invalid
				// for provider APIs. It is either moved by the group repair above
				// or dropped as stale synthetic output.
				recordRepair(msg.ToolCallID)
				continue
			}
			out = append(out, msg)
			continue
		}

		out = append(out, msg)
		expected := make([]string, 0, len(msg.ToolCalls))
		expectedSet := make(map[string]struct{}, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			if tc.ID == "" {
				continue
			}
			if _, ok := expectedSet[tc.ID]; ok {
				continue
			}
			expectedSet[tc.ID] = struct{}{}
			expected = append(expected, tc.ID)
		}

		immediate := make(map[string]*schema.Message, len(expected))
		j := i + 1
		for j < len(msgs) {
			next := msgs[j]
			if next == nil || next.Role != schema.Tool || next.ToolCallID == "" {
				break
			}
			usedToolResults[j] = true
			if _, ok := expectedSet[next.ToolCallID]; ok {
				if _, exists := immediate[next.ToolCallID]; !exists {
					immediate[next.ToolCallID] = next
				} else {
					recordRepair(next.ToolCallID)
				}
			} else {
				recordRepair(next.ToolCallID)
			}
			j++
		}

		for _, id := range expected {
			if existing := immediate[id]; existing != nil {
				out = append(out, existing)
				continue
			}
			if moved := firstUnusedToolResultAfter(toolResults[id], i, usedToolResults); moved != nil {
				usedToolResults[moved.index] = true
				out = append(out, moved.msg)
				recordRepair(id)
				continue
			}
			out = append(out, &schema.Message{
				Role:       schema.Tool,
				Content:    reason,
				ToolCallID: id,
			})
			recordRepair(id)
		}
		i = j - 1
	}

	if len(result.ToolCallIDs) > 1 {
		sort.Strings(result.ToolCallIDs)
		result.ToolCallIDs = compactStrings(result.ToolCallIDs)
		result.Count = len(result.ToolCallIDs)
	}
	return out, result
}

func firstUnusedToolResultAfter(refs []toolResultRef, assistantIndex int, used map[int]bool) *toolResultRef {
	for _, ref := range refs {
		if ref.index <= assistantIndex || used[ref.index] {
			continue
		}
		cp := ref
		return &cp
	}
	return nil
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:0]
	last := ""
	for i, value := range values {
		if i > 0 && value == last {
			continue
		}
		out = append(out, value)
		last = value
	}
	return out
}

// MessageCount returns the number of messages in the conversation history.
func (e *Engine) MessageCount() int {
	return e.history.Len()
}
