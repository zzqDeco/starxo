package model

// SessionData is the unified on-disk format that combines LLM conversation
// history (messages) and frontend display data (timeline turns) into a single
// atomically-written file. This eliminates the dual-source split between
// messages.json (backend) and display.json (frontend).
type SessionData struct {
	Version         int                    `json:"version"`                   // format version for future migrations
	Messages        []PersistedMessage     `json:"messages"`                  // LLM conversation history
	Display         []DisplayTurn          `json:"display"`                   // frontend-renderable timeline turns
	Streaming       *StreamingState        `json:"streaming,omitempty"`       // non-nil when interrupted mid-stream
	DiscoveredTools []DiscoveredToolRecord `json:"discoveredTools,omitempty"` // MCP deferred discovery state
}

// DiscoveredToolRecord is the persisted discovery record for a deferred MCP tool.
// CanonicalName is the primary key across in-memory and on-disk state.
type DiscoveredToolRecord struct {
	CanonicalName string `json:"canonicalName"`
	Server        string `json:"server"`
	Kind          string `json:"kind"`
	DiscoveredAt  int64  `json:"discoveredAt"`
}

// DisplayTurn represents a single turn (user or assistant) in the chat timeline,
// including all sub-events (tool calls, transfers, etc.).
type DisplayTurn struct {
	ID        string         `json:"id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Agent     string         `json:"agent,omitempty"`
	Timestamp int64          `json:"timestamp"`
	Events    []DisplayEvent `json:"events"`
}

// DisplayEvent is a single timeline event within a turn.
type DisplayEvent struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // "message" | "tool_call" | "tool_result" | "transfer" | "info" | "interrupt"
	Agent       string `json:"agent,omitempty"`
	Content     string `json:"content,omitempty"`
	ToolName    string `json:"toolName,omitempty"`
	ToolArgs    string `json:"toolArgs,omitempty"`
	ToolID      string `json:"toolId,omitempty"`
	ToolResult  string `json:"toolResult,omitempty"`
	Timestamp   int64  `json:"timestamp"`
	IsStreaming bool   `json:"isStreaming,omitempty"`
}

// StreamingState captures partial streaming content so that mid-stream saves
// can be restored as incomplete messages on reload.
type StreamingState struct {
	PartialContent string `json:"partialContent,omitempty"`
	AgentName      string `json:"agentName,omitempty"`
}
