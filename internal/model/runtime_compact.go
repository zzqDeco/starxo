package model

const RuntimeContextCompactVersion = 1

// RuntimeContextCompact is the persisted, runtime-aware summary used when
// older conversation turns are compacted out of the model prompt.
type RuntimeContextCompact struct {
	Version              int                      `json:"version"`
	CreatedAt            int64                    `json:"createdAt"`
	UpdatedAt            int64                    `json:"updatedAt"`
	OriginalMessageCount int                      `json:"originalMessageCount"`
	OmittedMessageCount  int                      `json:"omittedMessageCount"`
	TokenEstimate        int                      `json:"tokenEstimate"`
	Summary              string                   `json:"summary"`
	ToolSearch           RuntimeToolSearchCompact `json:"toolSearch,omitempty"`
	PermissionGrants     []RuntimePermissionGrant `json:"permissionGrants,omitempty"`
	Tasks                []RuntimeTaskCompact     `json:"tasks,omitempty"`
	FileReadState        []RuntimeFileReadState   `json:"fileReadState,omitempty"`
	DiffSummaries        []RuntimeDiffSummary     `json:"diffSummaries,omitempty"`
	Todos                []RuntimeTodoItem        `json:"todos,omitempty"`
	PlanDocument         *PlanDocument            `json:"planDocument,omitempty"`
	Workspace            *RuntimeWorkspaceCompact `json:"workspace,omitempty"`
	ActiveObjective      *RunObjective            `json:"activeObjective,omitempty"`
}

// RunObjective describes the current user turn the runtime is allowed to act on.
// Older messages are historical context unless Scope is "continuation".
type RunObjective struct {
	ID                string `json:"id"`
	RunID             string `json:"runId"`
	UserMessageID     string `json:"userMessageId"`
	Scope             string `json:"scope"`
	Objective         string `json:"objective"`
	Acceptance        string `json:"acceptance,omitempty"`
	CreatedAt         int64  `json:"createdAt"`
	HistoryStartIndex int    `json:"historyStartIndex,omitempty"`
}

// RuntimeToolSearchCompact preserves deferred tool discovery and announcement
// state across compact/reload.
type RuntimeToolSearchCompact struct {
	DiscoveredTools           []DiscoveredToolRecord     `json:"discoveredTools,omitempty"`
	DeferredAnnouncementState *DeferredAnnouncementState `json:"deferredAnnouncementState,omitempty"`
	MCPInstructionsDeltaState *MCPInstructionsDeltaState `json:"mcpInstructionsDeltaState,omitempty"`
}

// RuntimeTaskCompact is a serializable view of a runtime task snapshot.
type RuntimeTaskCompact struct {
	ID          string `json:"id"`
	SessionID   string `json:"sessionId,omitempty"`
	Type        string `json:"type,omitempty"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description,omitempty"`
	Command     string `json:"command,omitempty"`
	OutputPath  string `json:"outputPath,omitempty"`
	OutputSize  int64  `json:"outputSize,omitempty"`
	StartedAt   int64  `json:"startedAt,omitempty"`
	FinishedAt  int64  `json:"finishedAt,omitempty"`
	DurationMs  int64  `json:"durationMs,omitempty"`
	ExitCode    int    `json:"exitCode,omitempty"`
	Error       string `json:"error,omitempty"`
}

// RuntimeFileReadState records the latest known read range and content hash for
// a file the agent inspected. It gives compacted prompts enough context to avoid
// losing concurrency/read-state cues after older tool results are omitted.
type RuntimeFileReadState struct {
	FilePath    string `json:"filePath"`
	StartLine   int    `json:"startLine,omitempty"`
	NumLines    int    `json:"numLines,omitempty"`
	TotalLines  int    `json:"totalLines,omitempty"`
	ContentHash string `json:"contentHash,omitempty"`
	LastReadAt  int64  `json:"lastReadAt,omitempty"`
}

// RuntimeDiffSummary records recent write/edit results in compact form.
type RuntimeDiffSummary struct {
	FilePath     string `json:"filePath"`
	ToolName     string `json:"toolName,omitempty"`
	LinesAdded   int    `json:"linesAdded,omitempty"`
	LinesRemoved int    `json:"linesRemoved,omitempty"`
	Replacements int    `json:"replacements,omitempty"`
	Patch        string `json:"patch,omitempty"`
	Summary      string `json:"summary,omitempty"`
	UpdatedAt    int64  `json:"updatedAt,omitempty"`
}

// RuntimeTodoItem mirrors tools.TodoItem without making model depend on tools.
type RuntimeTodoItem struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	DependsOn []string `json:"depends_on,omitempty"`
}

// RuntimeWorkspaceCompact preserves active worktree routing context.
type RuntimeWorkspaceCompact struct {
	Active            bool   `json:"active"`
	WorkspacePath     string `json:"workspacePath,omitempty"`
	OriginalWorkspace string `json:"originalWorkspace,omitempty"`
	WorktreePath      string `json:"worktreePath,omitempty"`
	WorktreeBranch    string `json:"worktreeBranch,omitempty"`
}

func CloneRuntimeContextCompact(in *RuntimeContextCompact) *RuntimeContextCompact {
	if in == nil {
		return nil
	}
	out := *in
	out.ToolSearch = RuntimeToolSearchCompact{
		DiscoveredTools:           cloneDiscoveredToolRecords(in.ToolSearch.DiscoveredTools),
		DeferredAnnouncementState: cloneDeferredAnnouncementState(in.ToolSearch.DeferredAnnouncementState),
		MCPInstructionsDeltaState: cloneMCPInstructionsDeltaState(in.ToolSearch.MCPInstructionsDeltaState),
	}
	out.PermissionGrants = cloneRuntimePermissionGrants(in.PermissionGrants)
	out.Tasks = cloneRuntimeTaskCompacts(in.Tasks)
	out.FileReadState = cloneRuntimeFileReadState(in.FileReadState)
	out.DiffSummaries = cloneRuntimeDiffSummaries(in.DiffSummaries)
	out.Todos = cloneRuntimeTodoItems(in.Todos)
	out.PlanDocument = ClonePlanDocument(in.PlanDocument)
	if in.Workspace != nil {
		cp := *in.Workspace
		out.Workspace = &cp
	}
	if in.ActiveObjective != nil {
		cp := *in.ActiveObjective
		out.ActiveObjective = &cp
	}
	return &out
}

func cloneRuntimeTaskCompacts(in []RuntimeTaskCompact) []RuntimeTaskCompact {
	if in == nil {
		return nil
	}
	out := make([]RuntimeTaskCompact, len(in))
	copy(out, in)
	return out
}

func cloneRuntimeFileReadState(in []RuntimeFileReadState) []RuntimeFileReadState {
	if in == nil {
		return nil
	}
	out := make([]RuntimeFileReadState, len(in))
	copy(out, in)
	return out
}

func cloneRuntimeDiffSummaries(in []RuntimeDiffSummary) []RuntimeDiffSummary {
	if in == nil {
		return nil
	}
	out := make([]RuntimeDiffSummary, len(in))
	copy(out, in)
	return out
}

func cloneRuntimeTodoItems(in []RuntimeTodoItem) []RuntimeTodoItem {
	if in == nil {
		return nil
	}
	out := make([]RuntimeTodoItem, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].DependsOn = cloneStrings(in[i].DependsOn)
	}
	return out
}
