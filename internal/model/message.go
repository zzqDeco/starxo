package model

// PersistedMessage is the on-disk representation of a conversation message.
// Uses a slim format to avoid coupling to the eino schema internal types.
type PersistedMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
}
