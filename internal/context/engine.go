package agentctx

import (
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
	e.mu.RLock()
	sysPrompt := e.systemPrompt
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

	// Combine system message + history, then window.
	all := make([]*schema.Message, 0, 1+len(historyMsgs))
	all = append(all, sysMsg)
	all = append(all, historyMsgs...)

	// Estimate a reasonable message count from token budget.
	// Rough heuristic: ~200 tokens per message on average.
	cfg := DefaultWindowConfig()
	if e.maxTokens > 0 {
		estimated := e.maxTokens / 200
		if estimated > 0 && estimated < cfg.MaxMessages {
			cfg.MaxMessages = estimated
		}
	}

	return WindowMessages(all, cfg)
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
		result = append(result, pm)
	}
	return result
}

// ImportMessages restores conversation history from persisted messages.
// Clears existing history first.
func (e *Engine) ImportMessages(messages []model.PersistedMessage) {
	msgs := make([]*schema.Message, 0, len(messages))
	for _, pm := range messages {
		msg := &schema.Message{
			Role:       schema.RoleType(pm.Role),
			Content:    pm.Content,
			Name:       pm.Name,
			ToolCallID: pm.ToolCallID,
		}
		msgs = append(msgs, msg)
	}
	e.history.SetAll(msgs)
}

// MessageCount returns the number of messages in the conversation history.
func (e *Engine) MessageCount() int {
	return e.history.Len()
}
