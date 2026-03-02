package model

// PersistedToolCall is the on-disk representation of a tool call request.
type PersistedToolCall struct {
	ID       string                     `json:"id"`
	Function PersistedToolCallFunction  `json:"function"`
}

// PersistedToolCallFunction holds the function name and arguments of a tool call.
type PersistedToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// PersistedMessage is the on-disk representation of a conversation message.
// Uses a slim format to avoid coupling to the eino schema internal types.
type PersistedMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	Name       string               `json:"name,omitempty"`
	ToolCallID string               `json:"toolCallId,omitempty"`
	ToolCalls  []PersistedToolCall   `json:"toolCalls,omitempty"`
}
