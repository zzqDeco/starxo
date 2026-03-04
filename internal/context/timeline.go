package agentctx

import (
	"sync"

	"starxo/internal/model"
)

// TimelineCollector accumulates display turns in memory so the backend can
// persist them alongside conversation messages. It mirrors what the frontend
// chatStore does, but lives server-side where it can be saved reliably on
// shutdown, crash recovery, or session switch.
type TimelineCollector struct {
	mu    sync.RWMutex
	turns []model.DisplayTurn
}

// NewTimelineCollector creates an empty collector.
func NewTimelineCollector() *TimelineCollector {
	return &TimelineCollector{}
}

// StartTurn begins a new display turn (user or assistant).
func (tc *TimelineCollector) StartTurn(id, role, agent string, timestamp int64) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.turns = append(tc.turns, model.DisplayTurn{
		ID:        id,
		Role:      role,
		Agent:     agent,
		Timestamp: timestamp,
	})
}

// AddUserTurn adds a complete user turn.
func (tc *TimelineCollector) AddUserTurn(id, content string, timestamp int64) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.turns = append(tc.turns, model.DisplayTurn{
		ID:        id,
		Role:      "user",
		Content:   content,
		Timestamp: timestamp,
	})
}

// AddEvent appends an event to the current (last) assistant turn.
// If no turn exists, it creates one.
func (tc *TimelineCollector) AddEvent(evt model.DisplayEvent, agent string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Find or create the current assistant turn
	if len(tc.turns) == 0 || tc.turns[len(tc.turns)-1].Role != "assistant" {
		tc.turns = append(tc.turns, model.DisplayTurn{
			ID:        evt.ID,
			Role:      "assistant",
			Agent:     agent,
			Timestamp: evt.Timestamp,
			Events:    []model.DisplayEvent{},
		})
	}

	turn := &tc.turns[len(tc.turns)-1]

	// Stream chunk: accumulate into existing streaming message event
	if evt.Type == "stream_chunk" {
		for i := len(turn.Events) - 1; i >= 0; i-- {
			e := &turn.Events[i]
			if e.Type == "message" && e.IsStreaming && e.Agent == evt.Agent {
				e.Content += evt.Content
				return
			}
		}
		// Create new streaming message event
		turn.Events = append(turn.Events, model.DisplayEvent{
			ID:         evt.ID,
			Type:       "message",
			Agent:      evt.Agent,
			Content:    evt.Content,
			Timestamp:  evt.Timestamp,
			IsStreaming: true,
		})
		return
	}

	// Stream end: finalize streaming message
	if evt.Type == "stream_end" {
		for i := len(turn.Events) - 1; i >= 0; i-- {
			e := &turn.Events[i]
			if e.Type == "message" && e.IsStreaming && e.Agent == evt.Agent {
				e.IsStreaming = false
				// Also set the turn's content to the finalized message
				turn.Content = e.Content
				break
			}
		}
		return
	}

	// Tool result: attach to matching tool_call
	if evt.Type == "tool_result" && evt.ToolID != "" {
		for i := len(turn.Events) - 1; i >= 0; i-- {
			e := &turn.Events[i]
			if e.Type == "tool_call" && e.ToolID == evt.ToolID {
				e.ToolResult = evt.Content
				return
			}
		}
	}

	turn.Events = append(turn.Events, evt)
}

// SetTurnContent sets the final text content of the current assistant turn.
func (tc *TimelineCollector) SetTurnContent(content string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if len(tc.turns) == 0 {
		return
	}
	tc.turns[len(tc.turns)-1].Content = content
}

// Export returns a snapshot of all collected turns.
func (tc *TimelineCollector) Export() []model.DisplayTurn {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	out := make([]model.DisplayTurn, len(tc.turns))
	copy(out, tc.turns)
	return out
}

// Import replaces all turns with the given data (used on session restore).
func (tc *TimelineCollector) Import(turns []model.DisplayTurn) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.turns = make([]model.DisplayTurn, len(turns))
	copy(tc.turns, turns)
}

// Clear resets the collector.
func (tc *TimelineCollector) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.turns = nil
}

// Len returns the number of turns.
func (tc *TimelineCollector) Len() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return len(tc.turns)
}
