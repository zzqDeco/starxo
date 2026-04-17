package model

const (
	SessionDataVersion = 4

	ModeDefault = "default"
	ModePlan    = "plan"

	PendingPlanAttachmentKindApproved = "approved"
	PendingPlanAttachmentKindRejected = "rejected"
)

// SessionData is the unified on-disk format that combines LLM conversation
// history (messages) and frontend display data (timeline turns) into a single
// atomically-written file. This eliminates the dual-source split between
// messages.json (backend) and display.json (frontend).
type SessionData struct {
	Version                   int                        `json:"version"`                             // format version for future migrations
	Messages                  []PersistedMessage         `json:"messages"`                            // LLM conversation history
	Display                   []DisplayTurn              `json:"display"`                             // frontend-renderable timeline turns
	Streaming                 *StreamingState            `json:"streaming,omitempty"`                 // non-nil when interrupted mid-stream
	DiscoveredTools           []DiscoveredToolRecord     `json:"discoveredTools,omitempty"`           // MCP deferred discovery state
	DeferredAnnouncementState *DeferredAnnouncementState `json:"deferredAnnouncementState,omitempty"` // persisted deferred tools delta state
	MCPInstructionsDeltaState *MCPInstructionsDeltaState `json:"mcpInstructionsDeltaState,omitempty"` // persisted MCP instructions summary state
	Mode                      string                     `json:"mode,omitempty"`                      // persisted session mode
	PlanDocument              *PlanDocument              `json:"planDocument,omitempty"`              // persisted plan document
	PendingPlanApproval       *PendingPlanApproval       `json:"pendingPlanApproval,omitempty"`       // persisted approval gate state
	PendingPlanAttachment     *PendingPlanAttachment     `json:"pendingPlanAttachment,omitempty"`     // persisted continuation attachment
}

// PlanDocument is the persisted plan artifact for plan mode v2.
type PlanDocument struct {
	Markdown  string `json:"markdown"`
	UpdatedAt int64  `json:"updatedAt"`
}

// PendingPlanApproval marks that a plan is waiting for explicit approval.
type PendingPlanApproval struct {
	RequestedAt int64 `json:"requestedAt"`
}

// PendingPlanAttachment is a one-shot continuation attachment persisted across reloads.
type PendingPlanAttachment struct {
	Kind      string `json:"kind"`
	Markdown  string `json:"markdown"`
	Feedback  string `json:"feedback,omitempty"`
	CreatedAt int64  `json:"createdAt"`
}

// DeferredAnnouncementState tracks which deferred tool canonical names have
// already been announced to the model for this session.
type DeferredAnnouncementState struct {
	AnnouncedSearchableCanonicalNames []string `json:"announcedSearchableCanonicalNames"`
}

// MCPInstructionsDeltaState tracks the last MCP server-summary snapshot that
// was announced to the model for this session.
type MCPInstructionsDeltaState struct {
	LastAnnouncedSearchableServers  []string `json:"lastAnnouncedSearchableServers"`
	LastAnnouncedPendingServers     []string `json:"lastAnnouncedPendingServers"`
	LastAnnouncedUnavailableServers []string `json:"lastAnnouncedUnavailableServers"`
	LastInstructionsFingerprint     string   `json:"lastInstructionsFingerprint"`
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

const (
	SessionDataWarningInvalidMode                  = "invalid session mode; downgraded to default"
	SessionDataWarningInvalidPendingAttachmentKind = "invalid pending plan attachment kind; dropped attachment"
)

// DefaultSessionData returns the canonical zero-value session data for v4.
func DefaultSessionData() *SessionData {
	return &SessionData{
		Version: SessionDataVersion,
		Mode:    ModeDefault,
	}
}

// NormalizeSessionData applies v4 defaults and downgrade rules to session data.
// It always returns a copy so callers cannot accidentally alias plan-mode state.
func NormalizeSessionData(data *SessionData) (*SessionData, []string) {
	if data == nil {
		return nil, nil
	}

	normalized := &SessionData{
		Version:                   SessionDataVersion,
		Messages:                  clonePersistedMessages(data.Messages),
		Display:                   cloneDisplayTurns(data.Display),
		Streaming:                 CloneStreamingState(data.Streaming),
		DiscoveredTools:           cloneDiscoveredToolRecords(data.DiscoveredTools),
		DeferredAnnouncementState: cloneDeferredAnnouncementState(data.DeferredAnnouncementState),
		MCPInstructionsDeltaState: cloneMCPInstructionsDeltaState(data.MCPInstructionsDeltaState),
		Mode:                      data.Mode,
		PlanDocument:              ClonePlanDocument(data.PlanDocument),
		PendingPlanApproval:       ClonePendingPlanApproval(data.PendingPlanApproval),
		PendingPlanAttachment:     ClonePendingPlanAttachment(data.PendingPlanAttachment),
	}

	warnings := make([]string, 0, 2)
	switch normalized.Mode {
	case "", ModeDefault:
		normalized.Mode = ModeDefault
	case ModePlan:
	default:
		normalized.Mode = ModeDefault
		warnings = append(warnings, SessionDataWarningInvalidMode)
	}

	if normalized.PendingPlanAttachment != nil {
		switch normalized.PendingPlanAttachment.Kind {
		case PendingPlanAttachmentKindApproved, PendingPlanAttachmentKindRejected:
		default:
			normalized.PendingPlanAttachment = nil
			warnings = append(warnings, SessionDataWarningInvalidPendingAttachmentKind)
		}
	}

	return normalized, warnings
}

func ClonePlanDocument(in *PlanDocument) *PlanDocument {
	if in == nil {
		return nil
	}
	cp := *in
	return &cp
}

func ClonePendingPlanApproval(in *PendingPlanApproval) *PendingPlanApproval {
	if in == nil {
		return nil
	}
	cp := *in
	return &cp
}

func ClonePendingPlanAttachment(in *PendingPlanAttachment) *PendingPlanAttachment {
	if in == nil {
		return nil
	}
	cp := *in
	return &cp
}

func CloneStreamingState(in *StreamingState) *StreamingState {
	if in == nil {
		return nil
	}
	cp := *in
	return &cp
}

func clonePersistedMessages(in []PersistedMessage) []PersistedMessage {
	if in == nil {
		return nil
	}
	out := make([]PersistedMessage, len(in))
	for i := range in {
		out[i] = in[i]
		if in[i].ToolCalls != nil {
			out[i].ToolCalls = make([]PersistedToolCall, len(in[i].ToolCalls))
			copy(out[i].ToolCalls, in[i].ToolCalls)
		}
	}
	return out
}

func cloneDisplayTurns(in []DisplayTurn) []DisplayTurn {
	if in == nil {
		return nil
	}
	out := make([]DisplayTurn, len(in))
	for i := range in {
		out[i] = in[i]
		if in[i].Events != nil {
			out[i].Events = make([]DisplayEvent, len(in[i].Events))
			copy(out[i].Events, in[i].Events)
		}
	}
	return out
}

func cloneDiscoveredToolRecords(in []DiscoveredToolRecord) []DiscoveredToolRecord {
	if in == nil {
		return nil
	}
	out := make([]DiscoveredToolRecord, len(in))
	copy(out, in)
	return out
}

func cloneDeferredAnnouncementState(in *DeferredAnnouncementState) *DeferredAnnouncementState {
	if in == nil {
		return nil
	}
	return &DeferredAnnouncementState{
		AnnouncedSearchableCanonicalNames: cloneStrings(in.AnnouncedSearchableCanonicalNames),
	}
}

func cloneMCPInstructionsDeltaState(in *MCPInstructionsDeltaState) *MCPInstructionsDeltaState {
	if in == nil {
		return nil
	}
	return &MCPInstructionsDeltaState{
		LastAnnouncedSearchableServers:  cloneStrings(in.LastAnnouncedSearchableServers),
		LastAnnouncedPendingServers:     cloneStrings(in.LastAnnouncedPendingServers),
		LastAnnouncedUnavailableServers: cloneStrings(in.LastAnnouncedUnavailableServers),
		LastInstructionsFingerprint:     in.LastInstructionsFingerprint,
	}
}

func cloneStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
