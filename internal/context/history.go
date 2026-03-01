package agentctx

import (
	"sync"

	"github.com/cloudwego/eino/schema"
)

// ConversationHistory manages the conversation message history with thread safety.
type ConversationHistory struct {
	mu       sync.RWMutex
	messages []*schema.Message
}

// NewConversationHistory creates an empty conversation history.
func NewConversationHistory() *ConversationHistory {
	return &ConversationHistory{
		messages: make([]*schema.Message, 0),
	}
}

// Add appends a message to the conversation history.
func (h *ConversationHistory) Add(msg *schema.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, msg)
}

// GetAll returns a copy of all messages in the history.
func (h *ConversationHistory) GetAll() []*schema.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*schema.Message, len(h.messages))
	copy(out, h.messages)
	return out
}

// GetRecent returns the last n messages. If n exceeds the total count,
// all messages are returned.
func (h *ConversationHistory) GetRecent(n int) []*schema.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := len(h.messages)
	if n >= total {
		out := make([]*schema.Message, total)
		copy(out, h.messages)
		return out
	}
	out := make([]*schema.Message, n)
	copy(out, h.messages[total-n:])
	return out
}

// Len returns the number of messages in the history.
func (h *ConversationHistory) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.messages)
}

// Clear removes all messages from the history.
func (h *ConversationHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = make([]*schema.Message, 0)
}

// SetAll replaces the entire message history (used for session restore).
func (h *ConversationHistory) SetAll(msgs []*schema.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = make([]*schema.Message, len(msgs))
	copy(h.messages, msgs)
}
