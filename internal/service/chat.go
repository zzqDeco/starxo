package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"starxo/internal/agent"
	"starxo/internal/config"
	agentctx "starxo/internal/context"
	"starxo/internal/logger"
	"starxo/internal/model"
	"starxo/internal/sandbox"
	checkpoint "starxo/internal/store"
	"starxo/internal/tools"
)

// Context key for propagating session identity through agent execution.
type contextKey string

const sessionIDCtxKey contextKey = "sessionID"

const defaultBundleFreshnessTTL = 30 * time.Second

func contextWithSessionID(ctx context.Context, sessionID string) context.Context {
	ctx = context.WithValue(ctx, sessionIDCtxKey, sessionID)
	// Also store a plain-string key so lower-level internal packages can read
	// session scope without importing service package types (avoids import cycles).
	return context.WithValue(ctx, "sessionID", sessionID)
}

// SessionIDFromContext extracts the session ID from a context.
func SessionIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDCtxKey).(string); ok {
		return v
	}
	return ""
}

// Default context engine parameters.
const (
	defaultSystemPrompt = "You are an intelligent coding agent that helps users write, debug, and execute code in a sandboxed environment. You have access to tools for file operations, shell commands, and code execution. Always explain your approach before taking action."
	defaultMaxTokens    = 8000
)

const (
	orphanRepairSupersededReason    = "Error: tool execution was superseded by a newer user request"
	orphanRepairStoppedReason       = "Error: tool execution was stopped before the user provided interrupt input"
	orphanRepairSandboxReason       = "Error: tool execution was stopped because the sandbox connection was lost"
	orphanRepairSandboxSwitchReason = "Error: tool execution was stopped because the active sandbox changed"
)

const sandboxChangedAgentText = "Active sandbox changed; the agent run was stopped."

type runtimeRunWaiter struct {
	sessionID string
	done      <-chan struct{}
}

// PendingInterrupt holds the state needed to resume after an interrupt.
type PendingInterrupt struct {
	CheckpointID     string
	InterruptID      string
	BundleGeneration uint64
	RunnerKind       RunnerKind
	Info             any
	Objective        *model.RunObjective
	ToolCallIDs      []string
}

type RunnerKind string

const (
	RunnerKindDefault RunnerKind = "default"
	RunnerKindPlan    RunnerKind = "plan"
)

type cachedMCPServerSurface struct {
	ConfigIdentityDigest string
	HasToolMetadata      bool
	SupportsResources    bool
	ActionEntries        []tools.CatalogEntry
}

type RunnerBundle struct {
	Generation   uint64
	ConfigDigest string
	DefaultAgent adk.Agent
	PlanAgent    adk.Agent
	// Deprecated: top-level session execution is driven by TurnLoop over the
	// agents above. These fields remain temporarily for older bundle tests and
	// compatibility with legacy helper paths.
	DefaultRunner                 *adk.Runner
	PlanRunner                    *adk.Runner
	MCPCatalog                    *tools.ToolCatalog
	MCPHandles                    []*tools.MCPServerHandle
	LastFreshnessCheckAt          time.Time
	SurfaceRelevantFingerprint    string
	CachedSurfaceMetadataByServer map[string]cachedMCPServerSurface
}

type detachedBundleTaskKind string

const (
	detachedBundleTaskColdStart detachedBundleTaskKind = "cold-start"
	detachedBundleTaskFreshness detachedBundleTaskKind = "freshness"
)

type detachedBundleTask struct {
	Kind                 detachedBundleTaskKind
	key                  string
	TargetConfigDigest   string
	ExpectedGeneration   uint64
	ExpectedConfigDigest string
	err                  error
	fallbackToCurrent    bool
	done                 chan struct{}
}

type runnerBundleSurface struct {
	Handles                       []*tools.MCPServerHandle
	ActionCatalog                 *tools.ToolCatalog
	CachedSurfaceMetadataByServer map[string]cachedMCPServerSurface
	SurfaceRelevantFingerprint    string
}

// SessionRun holds per-session agent execution state.
// Each session gets its own context engine, timeline, and run lifecycle.
type SessionRun struct {
	sessionID                 string
	stateMu                   sync.RWMutex
	ctxEngine                 *agentctx.Engine
	timeline                  *agentctx.TimelineCollector
	discoveredTools           map[string]model.DiscoveredToolRecord
	permissionGrants          map[string]model.RuntimePermissionGrant
	permissionAudit           []model.RuntimePermissionAudit
	deferredAnnouncementState *model.DeferredAnnouncementState
	mcpInstructionsDeltaState *model.MCPInstructionsDeltaState
	runtimeContextCompact     *model.RuntimeContextCompact
	fileReadState             map[string]model.RuntimeFileReadState
	diffSummaries             []model.RuntimeDiffSummary
	planDocument              *model.PlanDocument
	pendingPlanApproval       *model.PendingPlanApproval
	pendingPlanAttachment     *model.PendingPlanAttachment
	activeObjective           *model.RunObjective

	// Run lifecycle
	running                      bool
	starting                     bool
	cancelFn                     context.CancelFunc
	startDone                    chan struct{}
	runDone                      chan struct{}
	turnLoop                     *adk.TurnLoop[runtimeTurnItem, *schema.Message]
	turnLoopCancel               context.CancelFunc
	turnLoopDone                 chan struct{}
	turnLoopStarted              bool
	pendingInterrupt             *PendingInterrupt
	streamingState               *model.StreamingState
	mode                         string // "default" or "plan"
	currentAgent                 string
	pendingStartBundleGeneration uint64
	activeBundleGeneration       uint64
	activeRunnerKind             RunnerKind
}

type SessionSnapshot struct {
	SessionData          *model.SessionData
	MessageCount         int
	HasSessionRun        bool
	DeferredSurfaceDebug *DeferredSurfaceDebug
}

func (r *SessionRun) addUserMessage(content string) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.ctxEngine.AddUserMessage(content)
}

func (r *SessionRun) beginObjective(userMessageID, runID, userMessage string, createdAt int64) *model.RunObjective {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	scope := objectiveScope(userMessage)
	historyStart := 0
	if scope == "standalone" {
		historyStart = r.ctxEngine.MessageCount() - 1
		if historyStart < 0 {
			historyStart = 0
		}
	}
	obj := &model.RunObjective{
		ID:                fmt.Sprintf("obj-%d", createdAt),
		RunID:             runID,
		UserMessageID:     userMessageID,
		Scope:             scope,
		Objective:         strings.TrimSpace(userMessage),
		Acceptance:        "Answer the current user request and verify any file or command changes that are part of that request.",
		CreatedAt:         createdAt,
		HistoryStartIndex: historyStart,
	}
	r.activeObjective = obj
	return cloneRunObjective(obj)
}

func (r *SessionRun) currentObjective() *model.RunObjective {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()
	return cloneRunObjective(r.activeObjective)
}

func (r *SessionRun) addAssistantMessage(content string) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.ctxEngine.AddAssistantMessage(content)
}

func (r *SessionRun) addToolResult(toolCallID, content string) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.ctxEngine.AddToolResult(toolCallID, content)
}

func (r *SessionRun) addMessage(msg *schema.Message) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.ctxEngine.AddMessage(msg)
}

func (r *SessionRun) repairOrphanToolHistory(reason string) agentctx.RepairResult {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	return r.ctxEngine.RepairOrphanToolCalls(reason)
}

func (r *SessionRun) removeCurrentTurnToolCallsForIDs(toolCallIDs map[string]pendingRuntimeToolCall, toolMessages, toolResultMessages []*schema.Message) int {
	if len(toolCallIDs) == 0 {
		return 0
	}
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	history := r.ctxEngine.History()
	msgs := history.GetAll()
	removeIDs := make(map[string]struct{}, len(toolCallIDs))
	for id := range toolCallIDs {
		removeIDs[id] = struct{}{}
	}
	turnMessages := make(map[*schema.Message]struct{}, len(toolMessages))
	for _, msg := range toolMessages {
		if msg != nil {
			turnMessages[msg] = struct{}{}
		}
	}
	turnResults := make(map[*schema.Message]struct{}, len(toolResultMessages))
	for _, msg := range toolResultMessages {
		if msg != nil {
			turnResults[msg] = struct{}{}
		}
	}
	filtered := make([]*schema.Message, 0, len(msgs))
	removed := 0
	for _, msg := range msgs {
		if msg == nil {
			filtered = append(filtered, msg)
			continue
		}
		if _, isCurrentTurnToolMessage := turnMessages[msg]; isCurrentTurnToolMessage && len(msg.ToolCalls) > 0 {
			keptCalls := make([]schema.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				if _, drop := removeIDs[tc.ID]; drop {
					continue
				}
				keptCalls = append(keptCalls, tc)
			}
			if len(keptCalls) != len(msg.ToolCalls) {
				if len(keptCalls) == 0 && msg.Content == "" {
					removed++
					continue
				}
				cloned := *msg
				cloned.ToolCalls = keptCalls
				msg = &cloned
			}
		}
		if _, isCurrentTurnToolResult := turnResults[msg]; isCurrentTurnToolResult && msg.ToolCallID != "" {
			if _, drop := removeIDs[msg.ToolCallID]; drop {
				removed++
				continue
			}
		}
		filtered = append(filtered, msg)
	}
	if removed > 0 {
		history.SetAll(filtered)
	}
	return removed
}

func (r *SessionRun) addUserTurn(id, content string, timestamp int64) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.timeline.AddUserTurn(id, content, timestamp)
}

func (r *SessionRun) prepareMessages() []*schema.Message {
	return r.prepareMessagesWithCompact(nil)
}

func (r *SessionRun) prepareMessagesWithCompact(compact *model.RuntimeContextCompact) []*schema.Message {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()
	objective := cloneRunObjective(r.activeObjective)
	pinned := runtimeObjectivePinnedMessages(objective)
	historyStart := 0
	if objective != nil && objective.Scope == "standalone" {
		historyStart = objective.HistoryStartIndex
	}
	return r.ctxEngine.PrepareMessagesWithCompactFrom(pinned, scopeCompactForObjective(compact, objective), historyStart)
}

func (r *SessionRun) clearSessionState() {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.ctxEngine.ClearHistory()
	r.timeline.Clear()
	r.streamingState = nil
	r.discoveredTools = make(map[string]model.DiscoveredToolRecord)
	r.permissionGrants = make(map[string]model.RuntimePermissionGrant)
	r.permissionAudit = nil
	r.deferredAnnouncementState = nil
	r.mcpInstructionsDeltaState = nil
	r.runtimeContextCompact = nil
	r.fileReadState = make(map[string]model.RuntimeFileReadState)
	r.diffSummaries = nil
	r.planDocument = nil
	r.pendingPlanApproval = nil
	r.pendingPlanAttachment = nil
	r.activeObjective = nil
}

func (r *SessionRun) setStreamingState(state *model.StreamingState) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.streamingState = state
}

func (r *SessionRun) streamingStateSnapshot() *model.StreamingState {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()
	if r.streamingState == nil {
		return nil
	}
	ss := *r.streamingState
	return &ss
}

func (r *SessionRun) importSessionData(data *model.SessionData) agentctx.RepairResult {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	repair := agentctx.RepairResult{}
	if data != nil && data.Messages != nil {
		repair = r.ctxEngine.ImportMessages(data.Messages)
	} else {
		r.ctxEngine.ClearHistory()
	}
	if data != nil && data.Display != nil {
		r.timeline.Import(data.Display)
	} else {
		r.timeline.Clear()
	}
	r.streamingState = nil
	r.discoveredTools = make(map[string]model.DiscoveredToolRecord)
	r.permissionGrants = make(map[string]model.RuntimePermissionGrant)
	r.permissionAudit = nil
	r.deferredAnnouncementState = nil
	r.mcpInstructionsDeltaState = nil
	r.runtimeContextCompact = nil
	r.fileReadState = make(map[string]model.RuntimeFileReadState)
	r.diffSummaries = nil
	r.planDocument = nil
	r.pendingPlanApproval = nil
	r.pendingPlanAttachment = nil
	r.activeObjective = nil
	r.mode = model.ModeDefault
	if data == nil {
		return repair
	}
	r.streamingState = model.CloneStreamingState(data.Streaming)
	r.mode = data.Mode
	r.deferredAnnouncementState = cloneDeferredAnnouncementState(data.DeferredAnnouncementState)
	r.mcpInstructionsDeltaState = cloneMCPInstructionsDeltaState(data.MCPInstructionsDeltaState)
	r.runtimeContextCompact = model.CloneRuntimeContextCompact(data.RuntimeContextCompact)
	if r.runtimeContextCompact != nil {
		r.activeObjective = cloneRunObjective(r.runtimeContextCompact.ActiveObjective)
	}
	r.planDocument = model.ClonePlanDocument(data.PlanDocument)
	r.pendingPlanApproval = model.ClonePendingPlanApproval(data.PendingPlanApproval)
	r.pendingPlanAttachment = model.ClonePendingPlanAttachment(data.PendingPlanAttachment)
	if data.RuntimeContextCompact != nil {
		for _, state := range data.RuntimeContextCompact.FileReadState {
			if state.FilePath != "" {
				r.fileReadState[state.FilePath] = state
			}
		}
		r.diffSummaries = append([]model.RuntimeDiffSummary(nil), data.RuntimeContextCompact.DiffSummaries...)
	}
	for _, record := range data.DiscoveredTools {
		if record.CanonicalName == "" {
			continue
		}
		r.discoveredTools[record.CanonicalName] = record
	}
	for _, grant := range data.PermissionGrants {
		if grant.ToolName == "" || grant.Decision != tools.ToolPermissionDecisionAllowSession {
			continue
		}
		r.permissionGrants[grant.ToolName] = grant
	}
	r.permissionAudit = append([]model.RuntimePermissionAudit(nil), data.PermissionAudit...)
	if len(r.permissionAudit) == 0 && data.RuntimeContextCompact != nil {
		r.permissionAudit = append([]model.RuntimePermissionAudit(nil), data.RuntimeContextCompact.PermissionAudit...)
	}
	return repair
}

func (r *SessionRun) snapshot() *SessionSnapshot {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()

	discovered := make([]model.DiscoveredToolRecord, 0, len(r.discoveredTools))
	for _, record := range r.discoveredTools {
		discovered = append(discovered, record)
	}
	sort.Slice(discovered, func(i, j int) bool {
		return discovered[i].CanonicalName < discovered[j].CanonicalName
	})
	grants := make([]model.RuntimePermissionGrant, 0, len(r.permissionGrants))
	for _, grant := range r.permissionGrants {
		grants = append(grants, grant)
	}
	sort.Slice(grants, func(i, j int) bool {
		return grants[i].ToolName < grants[j].ToolName
	})
	audit := append([]model.RuntimePermissionAudit(nil), r.permissionAudit...)

	return &SessionSnapshot{
		HasSessionRun: true,
		MessageCount:  r.ctxEngine.MessageCount(),
		SessionData: &model.SessionData{
			Version:                   model.SessionDataVersion,
			Messages:                  r.ctxEngine.ExportMessages(),
			Display:                   r.timeline.Export(),
			Streaming:                 cloneStreamingState(r.streamingState),
			DiscoveredTools:           discovered,
			PermissionGrants:          grants,
			PermissionAudit:           audit,
			DeferredAnnouncementState: cloneDeferredAnnouncementState(r.deferredAnnouncementState),
			MCPInstructionsDeltaState: cloneMCPInstructionsDeltaState(r.mcpInstructionsDeltaState),
			RuntimeContextCompact:     model.CloneRuntimeContextCompact(r.runtimeContextCompact),
			Mode:                      r.mode,
			PlanDocument:              model.ClonePlanDocument(r.planDocument),
			PendingPlanApproval:       model.ClonePendingPlanApproval(r.pendingPlanApproval),
			PendingPlanAttachment:     model.ClonePendingPlanAttachment(r.pendingPlanAttachment),
		},
	}
}

func (r *SessionRun) upsertDiscoveredTool(record model.DiscoveredToolRecord) bool {
	if record.CanonicalName == "" {
		return false
	}
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.discoveredTools == nil {
		r.discoveredTools = make(map[string]model.DiscoveredToolRecord)
	}
	if _, exists := r.discoveredTools[record.CanonicalName]; exists {
		return false
	}
	r.discoveredTools[record.CanonicalName] = record
	return true
}

func (r *SessionRun) discoveredToolsSnapshot() map[string]model.DiscoveredToolRecord {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()
	out := make(map[string]model.DiscoveredToolRecord, len(r.discoveredTools))
	for k, v := range r.discoveredTools {
		out[k] = v
	}
	return out
}

func (r *SessionRun) replaceDiscoveredTools(records []model.DiscoveredToolRecord) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.discoveredTools = make(map[string]model.DiscoveredToolRecord, len(records))
	for _, record := range records {
		if record.CanonicalName == "" {
			continue
		}
		r.discoveredTools[record.CanonicalName] = record
	}
}

func (r *SessionRun) deferredAnnouncementStateSnapshot() *model.DeferredAnnouncementState {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()
	return cloneDeferredAnnouncementState(r.deferredAnnouncementState)
}

func (r *SessionRun) setDeferredAnnouncementState(state *model.DeferredAnnouncementState) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.deferredAnnouncementState = cloneDeferredAnnouncementState(state)
}

func (r *SessionRun) mcpInstructionsDeltaStateSnapshot() *model.MCPInstructionsDeltaState {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()
	return cloneMCPInstructionsDeltaState(r.mcpInstructionsDeltaState)
}

func (r *SessionRun) applySyntheticDeltaStates(
	announcement *model.DeferredAnnouncementState,
	updateAnnouncement bool,
	instructions *model.MCPInstructionsDeltaState,
	updateInstructions bool,
) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if updateAnnouncement {
		r.deferredAnnouncementState = cloneDeferredAnnouncementState(announcement)
	}
	if updateInstructions {
		r.mcpInstructionsDeltaState = cloneMCPInstructionsDeltaState(instructions)
	}
}

func cloneStreamingState(in *model.StreamingState) *model.StreamingState {
	if in == nil {
		return nil
	}
	cp := *in
	return &cp
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneDeferredAnnouncementState(in *model.DeferredAnnouncementState) *model.DeferredAnnouncementState {
	if in == nil {
		return nil
	}
	return &model.DeferredAnnouncementState{
		AnnouncedSearchableCanonicalNames: cloneStrings(in.AnnouncedSearchableCanonicalNames),
	}
}

func cloneMCPInstructionsDeltaState(in *model.MCPInstructionsDeltaState) *model.MCPInstructionsDeltaState {
	if in == nil {
		return nil
	}
	return &model.MCPInstructionsDeltaState{
		LastAnnouncedSearchableServers:  cloneStrings(in.LastAnnouncedSearchableServers),
		LastAnnouncedPendingServers:     cloneStrings(in.LastAnnouncedPendingServers),
		LastAnnouncedUnavailableServers: cloneStrings(in.LastAnnouncedUnavailableServers),
		LastInstructionsFingerprint:     in.LastInstructionsFingerprint,
	}
}

// ChatService manages chat interactions between the frontend and the AI agent.
type ChatService struct {
	ctx context.Context

	sandbox           *sandbox.SandboxManager
	store             *config.Store
	checkpointStore   compose.CheckPointStore
	now               func() time.Time
	freshnessTTL      time.Duration
	runtimeOptions    ChatRuntimeOptions
	runtimeTasks      *runtimeTaskManager
	runtimeWorkspaces *runtimeWorkspaceManager
	runtimeLSP        *runtimeLSPManager

	installedBundle            *RunnerBundle
	retiredBundles             []*RunnerBundle
	coldStartTask              *detachedBundleTask
	freshnessTask              *detachedBundleTask
	nextGeneration             uint64
	probeBundleSurfaceFn       func(context.Context, *config.AppConfig, map[string]cachedMCPServerSurface) (*runnerBundleSurface, error)
	prepareRunnerBundleFn      func(context.Context, *config.AppConfig, string, map[string]cachedMCPServerSurface) (*RunnerBundle, error)
	prepareBundleFromSurfaceFn func(context.Context, *config.AppConfig, string, *runnerBundleSurface) (*RunnerBundle, error)
	closeRunnerBundleFn        func(*RunnerBundle)

	// Per-session execution state
	sessions        map[string]*SessionRun
	activeSessionID string

	// Service deps
	sessionService *SessionService
	sandboxService *SandboxService
	onAgentDone    func(sessionID string)

	permissionMu       sync.Mutex
	permissionRequests map[string]*runtimePermissionRequest

	mu sync.Mutex
}

// NewChatService creates a new ChatService.
func NewChatService(store *config.Store, opts ...ChatRuntimeOptions) *ChatService {
	runtimeOptions := ChatRuntimeOptions{}
	if len(opts) > 0 {
		runtimeOptions = opts[0]
	}
	s := &ChatService{
		store:              store,
		checkpointStore:    checkpoint.NewInMemoryStore(),
		sessions:           make(map[string]*SessionRun),
		now:                time.Now,
		freshnessTTL:       defaultBundleFreshnessTTL,
		runtimeOptions:     runtimeOptions,
		permissionRequests: make(map[string]*runtimePermissionRequest),
	}
	s.runtimeTasks = newRuntimeTaskManager(s.now, func(event string, data any) {
		wailsEmit(s.ctx, event, data)
	})
	s.runtimeTasks.onTaskGraphChanged = func(sessionID string) {
		s.saveSessionRuntimeState(sessionID)
	}
	s.runtimeWorkspaces = newRuntimeWorkspaceManager(s.now)
	s.runtimeLSP = newRuntimeLSPManager(s.now)
	return s
}

// SetContext stores the Wails application context. Called from app.go startup.
func (s *ChatService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetDependencies injects the sandbox manager.
// The ctxEngine parameter is accepted for backward compatibility but ignored;
// per-session context engines are managed inside SessionRun.
func (s *ChatService) SetDependencies(sbx *sandbox.SandboxManager, _ *agentctx.Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sandbox = sbx
}

// UpdateSandbox updates the sandbox manager reference.
func (s *ChatService) UpdateSandbox(sbx *sandbox.SandboxManager) {
	var stoppedSessions []string
	var clearedInterruptSessions []string
	var repairedSessions map[string]agentctx.RepairResult
	s.mu.Lock()
	s.sandbox = sbx
	if sbx == nil {
		stoppedSessions, clearedInterruptSessions, repairedSessions = s.cancelRunsForSandboxLossLocked()
	}
	s.invalidateRunners()
	lsp := s.runtimeLSP
	ctx := s.ctx
	s.mu.Unlock()
	if lsp != nil {
		lsp.CloseAll()
	}
	for _, sessionID := range stoppedSessions {
		wailsEmit(ctx, "agent:error", map[string]interface{}{
			"sessionId": sessionID,
			"error":     sandboxConnectionLostAgentText,
		})
		s.emitRunState(sessionID)
	}
	emitted := stringSet(stoppedSessions)
	for sessionID, repair := range repairedSessions {
		s.saveSessionAfterHistoryRepair(sessionID, repair)
		if _, ok := emitted[sessionID]; !ok {
			s.emitRunState(sessionID)
			emitted[sessionID] = struct{}{}
		}
	}
	for _, sessionID := range clearedInterruptSessions {
		if _, ok := emitted[sessionID]; !ok {
			s.emitRunState(sessionID)
			emitted[sessionID] = struct{}{}
		}
	}
}

// StopRunsNotBoundToSandbox stops any in-flight agent run whose session binding
// does not match targetContainerID. The desktop app has one mutable sandbox
// manager, so switching that manager to another sandbox while old runs continue
// would let their later tool calls execute in the wrong workspace.
func (s *ChatService) StopRunsNotBoundToSandbox(targetContainerID string) ([]string, error) {
	sessionBindings := s.sessionSandboxBindings()

	s.mu.Lock()
	stoppedSessions, clearedInterruptSessions, repairedSessions, waiters := s.stopRunsNotBoundToSandboxLocked(targetContainerID, sessionBindings)
	ctx := s.ctx
	s.mu.Unlock()

	timeoutSession := ""
	for _, waiter := range waiters {
		if waiter.done == nil {
			continue
		}
		select {
		case <-waiter.done:
		case <-time.After(5 * time.Second):
			logger.Warn("[CHAT] Timed out waiting for run to stop before sandbox switch",
				"session", waiter.sessionID,
				"target_container", targetContainerID,
			)
			timeoutSession = waiter.sessionID
		}
		if timeoutSession != "" {
			break
		}
	}

	for _, sessionID := range stoppedSessions {
		wailsEmit(ctx, "agent:error", map[string]interface{}{
			"sessionId": sessionID,
			"error":     sandboxChangedAgentText,
		})
		s.emitRunState(sessionID)
	}
	emitted := stringSet(stoppedSessions)
	for sessionID, repair := range repairedSessions {
		s.saveSessionAfterHistoryRepair(sessionID, repair)
		if _, ok := emitted[sessionID]; !ok {
			s.emitRunState(sessionID)
			emitted[sessionID] = struct{}{}
		}
	}
	for _, sessionID := range clearedInterruptSessions {
		if _, ok := emitted[sessionID]; !ok {
			s.emitRunState(sessionID)
			emitted[sessionID] = struct{}{}
		}
	}
	if timeoutSession != "" {
		return stoppedSessions, fmt.Errorf("timed out waiting for session %s to stop before switching sandbox", timeoutSession)
	}
	return stoppedSessions, nil
}

func (s *ChatService) sessionSandboxBindings() map[string]string {
	s.mu.Lock()
	sessionSvc := s.sessionService
	sessionIDs := make([]string, 0, len(s.sessions))
	for sessionID := range s.sessions {
		sessionIDs = append(sessionIDs, sessionID)
	}
	s.mu.Unlock()

	bindings := make(map[string]string, len(sessionIDs))
	if sessionSvc == nil {
		return bindings
	}
	for _, sessionID := range sessionIDs {
		bindings[sessionID] = sessionSvc.GetSessionBoundContainerID(sessionID)
	}
	return bindings
}

func (s *ChatService) stopRunsNotBoundToSandboxLocked(targetContainerID string, sessionBindings map[string]string) ([]string, []string, map[string]agentctx.RepairResult, []runtimeRunWaiter) {
	stopped := make([]string, 0)
	clearedInterrupts := make([]string, 0)
	repaired := make(map[string]agentctx.RepairResult)
	waiters := make([]runtimeRunWaiter, 0)

	for sessionID, run := range s.sessions {
		if run == nil {
			continue
		}
		if boundContainerID := sessionBindings[sessionID]; targetContainerID != "" && boundContainerID == targetContainerID {
			continue
		}
		if !run.running && !run.starting {
			if run.pendingInterrupt != nil {
				_, repair := s.clearPendingInterruptLocked(run, orphanRepairSandboxSwitchReason)
				clearedInterrupts = append(clearedInterrupts, sessionID)
				s.logHistoryRepair(sessionID, "sandbox_switch", repair)
				if repair.Count > 0 {
					repaired[sessionID] = repair
				}
			}
			continue
		}
		if run.turnLoop != nil {
			run.turnLoop.Stop(runtimeTurnLoopStopOptions("sandbox_changed")...)
		} else if run.cancelFn != nil {
			run.cancelFn()
		}
		run.cancelFn = nil
		if run.pendingInterrupt != nil {
			clearedInterrupts = append(clearedInterrupts, sessionID)
		}
		_, repair := s.clearPendingInterruptLocked(run, orphanRepairSandboxSwitchReason)
		s.logHistoryRepair(sessionID, "sandbox_switch", repair)
		if repair.Count > 0 {
			repaired[sessionID] = repair
		}
		if run.starting && run.startDone != nil {
			waiters = append(waiters, runtimeRunWaiter{sessionID: sessionID, done: run.startDone})
		} else if run.running && run.runDone != nil {
			waiters = append(waiters, runtimeRunWaiter{sessionID: sessionID, done: run.runDone})
		}
		stopped = append(stopped, sessionID)
	}
	if len(stopped) > 0 {
		s.cleanupRetiredBundlesLocked()
	}
	sort.Strings(stopped)
	sort.Strings(clearedInterrupts)
	return stopped, compactStringSlice(clearedInterrupts), repaired, waiters
}

func (s *ChatService) cancelRunsForSandboxLossLocked() ([]string, []string, map[string]agentctx.RepairResult) {
	stopped := make([]string, 0)
	clearedInterrupts := make([]string, 0)
	repaired := make(map[string]agentctx.RepairResult)
	for sessionID, run := range s.sessions {
		if run == nil || (!run.running && !run.starting) {
			if run != nil && run.pendingInterrupt != nil {
				_, repair := s.clearPendingInterruptLocked(run, orphanRepairSandboxReason)
				clearedInterrupts = append(clearedInterrupts, sessionID)
				s.logHistoryRepair(sessionID, "sandbox_lost", repair)
				if repair.Count > 0 {
					repaired[sessionID] = repair
				}
			}
			continue
		}
		if run.turnLoop == nil && run.cancelFn == nil {
			continue
		}
		if run.turnLoop != nil {
			run.turnLoop.Stop(runtimeTurnLoopStopOptions("sandbox_lost")...)
		} else {
			run.cancelFn()
		}
		run.cancelFn = nil
		hadInterrupt := run.pendingInterrupt != nil
		_, repair := s.clearPendingInterruptLocked(run, orphanRepairSandboxReason)
		if hadInterrupt {
			clearedInterrupts = append(clearedInterrupts, sessionID)
		}
		s.logHistoryRepair(sessionID, "sandbox_lost", repair)
		if repair.Count > 0 {
			repaired[sessionID] = repair
		}
		stopped = append(stopped, sessionID)
	}
	if len(stopped) > 0 {
		s.cleanupRetiredBundlesLocked()
	}
	sort.Strings(stopped)
	sort.Strings(clearedInterrupts)
	return stopped, compactStringSlice(clearedInterrupts), repaired
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func compactStringSlice(values []string) []string {
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

func (s *ChatService) clearPendingInterruptLocked(run *SessionRun, reason string) (string, agentctx.RepairResult) {
	if run == nil || run.pendingInterrupt == nil {
		if run != nil {
			return run.sessionID, agentctx.RepairResult{}
		}
		return "", agentctx.RepairResult{}
	}
	pending := run.pendingInterrupt
	run.pendingInterrupt = nil
	repair := run.repairOrphanToolHistory(reason)
	if repair.Count == 0 && len(pending.ToolCallIDs) > 0 {
		logger.Info("[CHAT] Cleared pending interrupt without orphan repair",
			"session", run.sessionID,
			"interrupt_id", pending.InterruptID,
			"tool_call_ids", pending.ToolCallIDs,
		)
	}
	return run.sessionID, repair
}

func (s *ChatService) logHistoryRepair(sessionID, source string, repair agentctx.RepairResult) {
	if repair.Count == 0 {
		return
	}
	logger.Warn("[CHAT] Repaired orphan tool-call history",
		"session", sessionID,
		"source", source,
		"count", repair.Count,
		"tool_call_ids", repair.ToolCallIDs,
	)
}

func (s *ChatService) saveSessionAfterHistoryRepair(sessionID string, repair agentctx.RepairResult) {
	if sessionID == "" || repair.Count == 0 {
		return
	}
	s.mu.Lock()
	sessionSvc := s.sessionService
	s.mu.Unlock()
	if sessionSvc == nil {
		return
	}
	if err := sessionSvc.SaveSessionByID(sessionID); err != nil {
		logger.Warn("[CHAT] Failed to schedule repaired history save", "session", sessionID, "error", err)
	}
}

// InvalidateRunner forces runners to be rebuilt on the next message.
func (s *ChatService) InvalidateRunner() {
	s.mu.Lock()
	s.invalidateRunners()
	lsp := s.runtimeLSP
	s.mu.Unlock()
	if lsp != nil {
		lsp.CloseAll()
	}
}

func (s *ChatService) invalidateRunners() {
	s.retireBundleLocked(s.installedBundle)
	s.installedBundle = nil
}

// SetOnAgentDone registers a callback that fires after the agent finishes processing.
// The callback receives the sessionID of the completed run.
func (s *ChatService) SetOnAgentDone(fn func(sessionID string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onAgentDone = fn
}

// SetSessionService injects the session service.
func (s *ChatService) SetSessionService(ss *SessionService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionService = ss
}

// SetSandboxService injects the sandbox service for active session/runtime guards.
func (s *ChatService) SetSandboxService(ss *SandboxService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sandboxService = ss
}

func (s *ChatService) validateActiveSessionSandbox(sessionID string) error {
	s.mu.Lock()
	sessionService := s.sessionService
	sandboxService := s.sandboxService
	mgr := s.sandbox
	s.mu.Unlock()

	if sessionService == nil {
		// Unit-level callers can exercise the runtime loop without the desktop
		// session/sandbox services. The Wails app always injects SessionService,
		// so product paths still enforce session-scoped sandbox binding below.
		return nil
	}

	boundContainerID := sessionService.GetSessionBoundContainerID(sessionID)
	if boundContainerID == "" {
		return fmt.Errorf("please activate a sandbox for this session before running the agent")
	}
	if sandboxService != nil {
		activeContainerID := sandboxService.ActiveContainerRegID()
		if activeContainerID == "" {
			return fmt.Errorf("please activate sandbox %s before running the agent", boundContainerID)
		}
		if activeContainerID != boundContainerID {
			return fmt.Errorf("active sandbox %s does not match session sandbox %s", activeContainerID, boundContainerID)
		}
	}
	if mgr == nil || !mgr.SSHConnected() {
		return fmt.Errorf("SSH not connected")
	}
	if !mgr.HasActiveContainer() {
		return fmt.Errorf("please activate sandbox %s before running the agent", boundContainerID)
	}
	return nil
}

func (s *ChatService) activeWorkspaceContainerID(sessionID string) string {
	s.mu.Lock()
	sessionService := s.sessionService
	sandboxService := s.sandboxService
	s.mu.Unlock()
	if sessionService != nil {
		if containerID := sessionService.GetSessionBoundContainerID(sessionID); containerID != "" {
			return containerID
		}
	}
	if sandboxService == nil {
		return ""
	}
	return sandboxService.ActiveContainerRegID()
}

// ---------------------------------------------------------------------------
// Per-session state management
// ---------------------------------------------------------------------------

// getOrCreateRun returns the SessionRun for the given session.
// Creates a new one if it doesn't exist. Caller must hold s.mu.
func (s *ChatService) getOrCreateRun(sessionID string) *SessionRun {
	if run, ok := s.sessions[sessionID]; ok {
		return run
	}
	run := &SessionRun{
		sessionID:        sessionID,
		ctxEngine:        agentctx.NewEngine(defaultSystemPrompt, defaultMaxTokens),
		timeline:         agentctx.NewTimelineCollector(),
		discoveredTools:  make(map[string]model.DiscoveredToolRecord),
		permissionGrants: make(map[string]model.RuntimePermissionGrant),
		permissionAudit:  nil,
		fileReadState:    make(map[string]model.RuntimeFileReadState),
		mode:             model.ModeDefault,
	}
	s.sessions[sessionID] = run
	return run
}

func (s *ChatService) anySessionRunningLocked() bool {
	for _, run := range s.sessions {
		if run.running {
			return true
		}
	}
	return false
}

func (s *ChatService) nextBundleGenerationLocked() uint64 {
	s.nextGeneration++
	return s.nextGeneration
}

func (s *ChatService) retireBundleLocked(bundle *RunnerBundle) {
	if bundle == nil {
		return
	}
	if s.bundleGenerationReferencedLocked(bundle.Generation) {
		s.retiredBundles = append(s.retiredBundles, bundle)
		return
	}
	s.closeRunnerBundleLocked(bundle)
}

func (s *ChatService) closeMCPHandlesLocked(handles []*tools.MCPServerHandle) {
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if err := handle.Close(); err != nil {
			logger.Warn("[CHAT] Failed to close MCP handle", "server", handle.Name, "error", err)
		}
	}
}

func (s *ChatService) closeRunnerBundleLocked(bundle *RunnerBundle) {
	if bundle == nil {
		return
	}
	if s.closeRunnerBundleFn != nil {
		s.closeRunnerBundleFn(bundle)
		return
	}
	s.closeMCPHandlesLocked(bundle.MCPHandles)
}

func (s *ChatService) closeRunnerBundle(bundle *RunnerBundle) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeRunnerBundleLocked(bundle)
}

func (s *ChatService) bundleGenerationReferencedLocked(generation uint64) bool {
	if generation == 0 {
		return false
	}
	for _, run := range s.sessions {
		if run.pendingStartBundleGeneration == generation {
			return true
		}
		if run.activeBundleGeneration == generation {
			return true
		}
		if run.pendingInterrupt != nil && run.pendingInterrupt.BundleGeneration == generation {
			return true
		}
	}
	return false
}

func (s *ChatService) cleanupRetiredBundlesLocked() {
	if len(s.retiredBundles) == 0 {
		return
	}
	keep := s.retiredBundles[:0]
	for _, bundle := range s.retiredBundles {
		if bundle == nil {
			continue
		}
		if s.bundleGenerationReferencedLocked(bundle.Generation) {
			keep = append(keep, bundle)
			continue
		}
		s.closeRunnerBundleLocked(bundle)
	}
	s.retiredBundles = keep
}

func (s *ChatService) installedBundleLocked() *RunnerBundle {
	return s.installedBundle
}

func (s *ChatService) findBundleByGenerationLocked(generation uint64) *RunnerBundle {
	if generation == 0 {
		return nil
	}
	if s.installedBundle != nil && s.installedBundle.Generation == generation {
		return s.installedBundle
	}
	for _, bundle := range s.retiredBundles {
		if bundle != nil && bundle.Generation == generation {
			return bundle
		}
	}
	return nil
}

func runnerKindForMode(mode string) RunnerKind {
	if mode == model.ModePlan {
		return RunnerKindPlan
	}
	return RunnerKindDefault
}

func runnerForKind(bundle *RunnerBundle, kind RunnerKind) *adk.Runner {
	if bundle == nil {
		return nil
	}
	if kind == RunnerKindPlan {
		return bundle.PlanRunner
	}
	return bundle.DefaultRunner
}

func agentForKind(bundle *RunnerBundle, kind RunnerKind) adk.Agent {
	if bundle == nil {
		return nil
	}
	if kind == RunnerKindPlan {
		return bundle.PlanAgent
	}
	return bundle.DefaultAgent
}

func bundleKey(generation uint64, digest string) string {
	return fmt.Sprintf("%d:%s", generation, digest)
}

func coldStartTaskKey(targetConfigDigest string) string {
	return fmt.Sprintf("cold-start:%s", targetConfigDigest)
}

func cloneCatalogEntries(entries []tools.CatalogEntry) []tools.CatalogEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]tools.CatalogEntry, len(entries))
	copy(out, entries)
	for i := range out {
		if len(out[i].Aliases) > 0 {
			out[i].Aliases = append([]string(nil), out[i].Aliases...)
		}
	}
	return out
}

func cloneSurfaceCache(in map[string]cachedMCPServerSurface) map[string]cachedMCPServerSurface {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]cachedMCPServerSurface, len(in))
	for server, cache := range in {
		out[server] = cachedMCPServerSurface{
			ConfigIdentityDigest: cache.ConfigIdentityDigest,
			HasToolMetadata:      cache.HasToolMetadata,
			SupportsResources:    cache.SupportsResources,
			ActionEntries:        cloneCatalogEntries(cache.ActionEntries),
		}
	}
	return out
}

type mcpServerConfigIdentity struct {
	Name      string              `json:"name"`
	Transport string              `json:"transport"`
	Command   string              `json:"command"`
	Args      []string            `json:"args"`
	URL       string              `json:"url"`
	Env       []mcpServerEnvValue `json:"env"`
	Enabled   bool                `json:"enabled"`
}

type mcpServerEnvValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func configDigest(cfg *config.AppConfig) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func mcpServerConfigIdentityDigest(cfg config.MCPServerConfig) (string, error) {
	env := make([]mcpServerEnvValue, 0, len(cfg.Env))
	for k, v := range cfg.Env {
		env = append(env, mcpServerEnvValue{Key: k, Value: v})
	}
	sort.Slice(env, func(i, j int) bool {
		return env[i].Key < env[j].Key
	})

	identity := mcpServerConfigIdentity{
		Name:      cfg.Name,
		Transport: cfg.Transport,
		Command:   cfg.Command,
		Args:      append([]string{}, cfg.Args...),
		URL:       cfg.URL,
		Env:       env,
		Enabled:   cfg.Enabled,
	}

	if identity.Args == nil {
		identity.Args = []string{}
	}
	if identity.Env == nil {
		identity.Env = []mcpServerEnvValue{}
	}

	data, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func configuredMCPServerIdentityDigests(cfg *config.AppConfig) (map[string]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	out := make(map[string]string, len(cfg.MCP.Servers))
	for _, server := range cfg.MCP.Servers {
		digest, err := mcpServerConfigIdentityDigest(server)
		if err != nil {
			return nil, fmt.Errorf("compute MCP server config identity for %s: %w", server.Name, err)
		}
		out[server.Name] = digest
	}
	return out, nil
}

func matchingSurfaceCacheEntry(cache map[string]cachedMCPServerSurface, serverName, currentIdentityDigest string) (cachedMCPServerSurface, bool) {
	if len(cache) == 0 || serverName == "" || currentIdentityDigest == "" {
		return cachedMCPServerSurface{}, false
	}
	entry, ok := cache[serverName]
	if !ok || entry.ConfigIdentityDigest == "" || entry.ConfigIdentityDigest != currentIdentityDigest {
		return cachedMCPServerSurface{}, false
	}
	return entry, true
}

func bundleSurfaceFreshForPruning(bundle *RunnerBundle, now time.Time, freshnessTTL time.Duration) bool {
	if bundle == nil {
		return false
	}
	if freshnessTTL <= 0 {
		return true
	}
	return now.Sub(bundle.LastFreshnessCheckAt) < freshnessTTL
}

func (s *ChatService) detachedTaskContext() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *ChatService) activeDetachedTaskLocked(kind detachedBundleTaskKind) *detachedBundleTask {
	if kind == detachedBundleTaskColdStart {
		return s.coldStartTask
	}
	return s.freshnessTask
}

func (s *ChatService) setDetachedTaskLocked(task *detachedBundleTask) {
	if task == nil {
		return
	}
	if task.Kind == detachedBundleTaskColdStart {
		s.coldStartTask = task
		return
	}
	s.freshnessTask = task
}

func (s *ChatService) clearDetachedTaskLocked(task *detachedBundleTask) {
	if task == nil {
		return
	}
	if task.Kind == detachedBundleTaskColdStart {
		if s.coldStartTask == task {
			s.coldStartTask = nil
		}
		return
	}
	if s.freshnessTask == task {
		s.freshnessTask = nil
	}
}

func (s *ChatService) finalizeStartupLocked(sessionID string) {
	run, ok := s.sessions[sessionID]
	if !ok {
		s.cleanupRetiredBundlesLocked()
		return
	}
	run.starting = false
	run.cancelFn = nil
	run.pendingStartBundleGeneration = 0
	if run.startDone != nil {
		close(run.startDone)
		run.startDone = nil
	}
	s.cleanupRetiredBundlesLocked()
}

func (s *ChatService) publishStartupLocked(
	sessionID string,
	bundle *RunnerBundle,
	runnerKind RunnerKind,
	cancel context.CancelFunc,
	done chan struct{},
) (*SessionRun, error) {
	run, ok := s.sessions[sessionID]
	if !ok {
		s.cleanupRetiredBundlesLocked()
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	if bundle == nil {
		s.finalizeStartupLocked(sessionID)
		return nil, fmt.Errorf("runner bundle is required")
	}
	if run.running {
		s.finalizeStartupLocked(sessionID)
		return nil, fmt.Errorf("agent is already running in this session")
	}
	if !run.starting {
		s.finalizeStartupLocked(sessionID)
		return nil, fmt.Errorf("session %s startup is no longer active", sessionID)
	}
	if run.pendingStartBundleGeneration != bundle.Generation {
		s.finalizeStartupLocked(sessionID)
		return nil, fmt.Errorf("session %s startup is no longer active", sessionID)
	}
	run.cancelFn = cancel
	run.running = true
	run.starting = false
	run.pendingStartBundleGeneration = 0
	run.activeBundleGeneration = bundle.Generation
	run.activeRunnerKind = runnerKind
	run.runDone = done
	if run.startDone != nil {
		close(run.startDone)
		run.startDone = nil
	}
	s.cleanupRetiredBundlesLocked()
	return run, nil
}

func (s *ChatService) reservePendingStartLocked(sessionID string, generation uint64) (*SessionRun, error) {
	if generation == 0 {
		return nil, fmt.Errorf("runner bundle generation is required")
	}
	run, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	if run.running {
		return nil, fmt.Errorf("agent is already running in this session")
	}
	if !run.starting {
		return nil, fmt.Errorf("session %s startup is no longer active", sessionID)
	}
	run.pendingStartBundleGeneration = generation
	return run, nil
}

func (s *ChatService) clearPendingStartLocked(sessionID string) {
	s.finalizeStartupLocked(sessionID)
}

func (s *ChatService) reserveInstalledBundleLocked(sessionID string) (*RunnerBundle, error) {
	bundle := s.installedBundle
	if bundle == nil {
		return nil, fmt.Errorf("runner bundle is not installed")
	}
	if _, err := s.reservePendingStartLocked(sessionID, bundle.Generation); err != nil {
		return nil, err
	}
	return bundle, nil
}

// activeRun returns the SessionRun for the currently active session.
// Returns nil if no active session is set. Caller must hold s.mu.
func (s *ChatService) activeRun() *SessionRun {
	if s.activeSessionID == "" {
		return nil
	}
	return s.getOrCreateRun(s.activeSessionID)
}

// SetActiveSessionID sets the currently active session.
func (s *ChatService) SetActiveSessionID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeSessionID = id
}

// GetActiveSessionID returns the currently active session ID.
func (s *ChatService) GetActiveSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeSessionID
}

// GetOrCreateRun returns the SessionRun for the given session (for SessionService).
func (s *ChatService) GetOrCreateRun(sessionID string) *SessionRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getOrCreateRun(sessionID)
}

// RemoveSession removes a session's run state from memory.
func (s *ChatService) RemoveSession(sessionID string) {
	s.mu.Lock()
	hadSession := false
	if run, ok := s.sessions[sessionID]; ok {
		hadSession = true
		if run.turnLoop != nil {
			s.resetRuntimeTurnLoopLocked(run)
		} else if run.cancelFn != nil {
			run.cancelFn()
		}
		if run.running || run.starting {
			s.finishRuntimeTurnLocked(run)
		}
		if run.startDone != nil {
			close(run.startDone)
			run.startDone = nil
		}
		run.starting = false
		run.pendingStartBundleGeneration = 0
	}
	delete(s.sessions, sessionID)
	s.cleanupRetiredBundlesLocked()
	s.mu.Unlock()
	if hadSession {
		s.deleteRuntimeTurnCheckpoint(sessionID)
	}
	tools.ClearTodosForSession(sessionID)
}

// ---------------------------------------------------------------------------
// Mode management
// ---------------------------------------------------------------------------

// SetMode switches the active session between "default" and "plan" mode.
func (s *ChatService) SetMode(mode string) error {
	s.mu.Lock()

	if mode != model.ModeDefault && mode != model.ModePlan {
		s.mu.Unlock()
		return fmt.Errorf("invalid mode: %s (must be 'default' or 'plan')", mode)
	}

	run := s.activeRun()
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("no active session")
	}

	if run.mode == mode {
		s.mu.Unlock()
		return nil
	}

	run.mode = mode
	sessionID := s.activeSessionID
	sessionSvc := s.sessionService
	ctx := s.ctx
	s.mu.Unlock()

	logger.Info("[CHAT] Mode changed", "mode", mode, "session", sessionID)
	if ctx != nil {
		wailsruntime.EventsEmit(ctx, "agent:mode_changed", ModeChangedEvent{
			Mode:      mode,
			SessionID: sessionID,
		})
	}
	s.emitRunState(sessionID)
	if sessionSvc != nil && sessionID != "" {
		if err := sessionSvc.SaveSessionByID(sessionID); err != nil {
			logger.Warn("[CHAT] Failed to schedule mode save", "session", sessionID, "error", err)
		}
	}
	return nil
}

// GetMode returns the current agent mode for the active session.
func (s *ChatService) GetMode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	if run == nil {
		return model.ModeDefault
	}
	return run.mode
}

// ---------------------------------------------------------------------------
// Run status
// ---------------------------------------------------------------------------

// IsRunning returns whether the active session has a running agent.
func (s *ChatService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	return run != nil && run.running
}

// IsSessionRunning returns whether a specific session has a running agent.
func (s *ChatService) IsSessionRunning(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	return ok && run.running
}

func (s *ChatService) runStateEventLocked(sessionID string) (RunStateEvent, bool) {
	run, ok := s.sessions[sessionID]
	if !ok {
		return RunStateEvent{}, false
	}
	mode := run.mode
	if mode == "" {
		mode = model.ModeDefault
	}
	return RunStateEvent{
		SessionID:    sessionID,
		Running:      run.running || run.starting,
		CurrentAgent: run.currentAgent,
		Mode:         mode,
		HasInterrupt: run.pendingInterrupt != nil,
	}, true
}

func (s *ChatService) emitRunState(sessionID string) {
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	evt, ok := s.runStateEventLocked(sessionID)
	ctx := s.ctx
	s.mu.Unlock()
	if !ok || ctx == nil || ctx.Value("events") == nil {
		return
	}
	wailsruntime.EventsEmit(ctx, "agent:run_state", evt)
}

// WaitForSessionDone waits for a specific session's agent run to complete.
func (s *ChatService) WaitForSessionDone(sessionID string, timeout time.Duration) error {
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	if !ok || !run.running {
		s.mu.Unlock()
		return nil
	}
	done := run.runDone
	s.mu.Unlock()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for session %s", sessionID)
	}
}

// ---------------------------------------------------------------------------
// SendMessage
// ---------------------------------------------------------------------------

// SendMessage processes a user message through the agent and streams results to the frontend.
func (s *ChatService) SendMessage(userMessage string) error {
	for {
		s.mu.Lock()
		sessionID := s.activeSessionID
		s.mu.Unlock()
		if sessionID == "" {
			return fmt.Errorf("no active session")
		}
		if err := s.validateActiveSessionSandbox(sessionID); err != nil {
			return err
		}

		s.mu.Lock()
		if s.activeSessionID == "" {
			s.mu.Unlock()
			return fmt.Errorf("no active session")
		}
		if s.activeSessionID != sessionID {
			s.mu.Unlock()
			continue
		}
		run := s.activeRun()
		if run == nil {
			s.mu.Unlock()
			return fmt.Errorf("no active session")
		}
		sessionID = run.sessionID
		if run.starting && !run.running {
			if run.cancelFn != nil {
				run.cancelFn()
			} else if run.turnLoop != nil {
				run.turnLoop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
			}
			done := run.startDone
			logger.Info("[CHAT] Canceling startup before replacing objective", "session", sessionID)
			s.mu.Unlock()
			if done != nil {
				select {
				case <-done:
				case <-time.After(runtimeTurnStartupPreemptTimeout):
					return fmt.Errorf("previous agent startup is still canceling for session %s", sessionID)
				}
			}
			s.emitRunState(sessionID)
			continue
		}
		preempt := run.running
		clearPendingInterrupt := run.pendingInterrupt != nil && !preempt
		var repairSessionID string
		var repair agentctx.RepairResult
		if clearPendingInterrupt {
			repairSessionID, repair = s.clearPendingInterruptLocked(run, orphanRepairSupersededReason)
			s.logHistoryRepair(repairSessionID, "new_user_turn", repair)
			s.resetRuntimeTurnLoopLocked(run)
		}
		item := s.newRuntimeUserTurnItem(sessionID, userMessage)
		loop := s.ensureRuntimeTurnLoopLocked(sessionID, run)
		logger.Info("[CHAT] User message received",
			"length", len(userMessage),
			"preview", truncateResult(userMessage, 100),
			"session", sessionID,
		)
		s.mu.Unlock()
		s.saveSessionAfterHistoryRepair(repairSessionID, repair)

		if !preempt {
			if err := s.deleteRuntimeTurnCheckpoint(sessionID); err != nil {
				return err
			}
		}
		ok := false
		if preempt {
			var ack <-chan struct{}
			ok, ack = loop.Push(item, adk.WithPreemptTimeout[runtimeTurnItem, *schema.Message](adk.AfterToolCalls, runtimeTurnPreemptTimeout))
			if ack != nil {
				go func() {
					<-ack
					logger.Info("[CHAT] Runtime turn preempt acknowledged", "session", sessionID)
				}()
			}
		} else {
			ok, _ = loop.Push(item)
		}
		if !ok {
			s.mu.Lock()
			run = s.sessions[sessionID]
			if run == nil {
				s.mu.Unlock()
				return fmt.Errorf("session %s not found", sessionID)
			}
			if run.turnLoop == loop {
				s.resetRuntimeTurnLoopLocked(run)
			}
			loop = s.ensureRuntimeTurnLoopLocked(sessionID, run)
			s.mu.Unlock()
			ok, _ = loop.Push(item)
		}
		if !ok {
			return fmt.Errorf("runtime turn loop is stopped for session %s", sessionID)
		}
		s.mu.Lock()
		if run = s.sessions[sessionID]; run != nil && run.turnLoop == loop {
			s.startRuntimeTurnLoopLocked(sessionID, run, loop)
		}
		s.mu.Unlock()
		s.emitRunState(sessionID)
		return nil
	}
}

// ---------------------------------------------------------------------------
// Timeline emission
// ---------------------------------------------------------------------------

// emitTimelineForRun emits a timeline event to the frontend AND records it in the
// session's timeline collector for backend persistence.
func (s *ChatService) emitTimelineForRun(evt TimelineEvent, run *SessionRun) {
	evt.SessionID = run.sessionID
	wailsEmit(s.ctx, "agent:timeline", evt)
	if evt.Agent != "" {
		agentChanged := false
		s.mu.Lock()
		if current, ok := s.sessions[run.sessionID]; ok && current.currentAgent != evt.Agent {
			current.currentAgent = evt.Agent
			agentChanged = true
		}
		s.mu.Unlock()
		if agentChanged {
			s.emitRunState(run.sessionID)
		}
	}
	run.stateMu.Lock()
	defer run.stateMu.Unlock()
	run.timeline.AddEvent(model.DisplayEvent{
		ID:        evt.ID,
		Type:      evt.Type,
		Agent:     evt.Agent,
		Content:   evt.Content,
		ToolName:  evt.ToolName,
		ToolArgs:  evt.ToolArgs,
		ToolID:    evt.ToolID,
		Timestamp: evt.Timestamp,
	}, evt.Agent)
}

// emitTimelineForSession emits a timeline event using a session ID lookup
// (for OnToolEvent callbacks where we only have context, not a run reference).
func (s *ChatService) emitTimelineForSession(evt TimelineEvent, sessionID string) {
	evt.SessionID = sessionID
	wailsEmit(s.ctx, "agent:timeline", evt)
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if ok {
		agentChanged := false
		if evt.Agent != "" {
			s.mu.Lock()
			if current, exists := s.sessions[sessionID]; exists && current.currentAgent != evt.Agent {
				current.currentAgent = evt.Agent
				agentChanged = true
			}
			s.mu.Unlock()
		}
		if agentChanged {
			s.emitRunState(sessionID)
		}
		run.stateMu.Lock()
		defer run.stateMu.Unlock()
		run.timeline.AddEvent(model.DisplayEvent{
			ID:        evt.ID,
			Type:      evt.Type,
			Agent:     evt.Agent,
			Content:   evt.Content,
			ToolName:  evt.ToolName,
			ToolArgs:  evt.ToolArgs,
			ToolID:    evt.ToolID,
			Timestamp: evt.Timestamp,
		}, evt.Agent)
	}
}

// ---------------------------------------------------------------------------
// Event processing
// ---------------------------------------------------------------------------

// processEventsForRun consumes the event stream for a specific session run,
// emits frontend events, and detects interrupts.
// Returns the last message content, transfer count, and whether an interrupt occurred.
func (s *ChatService) processEventsForRun(events *adk.AsyncIterator[*adk.AgentEvent], checkpointID string, run *SessionRun, preempted <-chan struct{}) (string, int, bool, bool) {
	var allContents []string
	var transferCount int
	lastContentByAgent := make(map[string]string) // dedup

	// Track pending tool_call_ids to detect orphans (tool calls without results)
	pendingToolCalls := make(map[string]pendingRuntimeToolCall)
	var currentTurnToolMessages []*schema.Message
	var currentTurnToolResultMessages []*schema.Message

	sessionID := run.sessionID

	// Debounced intermediate save: persist at most once per 10 seconds during agent execution
	var lastSaveTime time.Time
	maybeSave := func() {
		if time.Since(lastSaveTime) > 10*time.Second {
			s.mu.Lock()
			ss := s.sessionService
			s.mu.Unlock()
			if ss != nil {
				go func() { _ = ss.SaveSessionByID(sessionID) }()
			}
			lastSaveTime = time.Now()
		}
	}

	for {
		event, ok := events.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			logger.Error("[CHAT] Agent event error", event.Err, "agent", event.AgentName, "session", sessionID)
			// Emit error as timeline info event so the user sees it, but do NOT break
			// the event loop — subsequent events (including sub-agent work) may follow.
			s.emitTimelineForRun(TimelineEvent{
				ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
				Type:      "info",
				Agent:     event.AgentName,
				Content:   runtimeAgentErrorContent(event.Err, sessionID),
				Timestamp: time.Now().UnixMilli(),
			}, run)
			continue
		}

		// Handle agent actions (tool calls, transfers, interrupts)
		if event.Action != nil {
			// Interrupt detection
			if event.Action.Interrupted != nil && len(event.Action.Interrupted.InterruptContexts) > 0 {
				interruptCtx := event.Action.Interrupted.InterruptContexts[0]
				s.handleInterruptForRun(interruptCtx, checkpointID, run, pendingToolCallIDs(pendingToolCalls))
				return strings.Join(allContents, "\n\n"), transferCount, true, runtimeTurnSignalClosed(preempted)
			}

			if event.Action.TransferToAgent != nil {
				transferCount++
				destName := event.Action.TransferToAgent.DestAgentName
				logger.Transfer(event.AgentName, destName,
					"transfer_count", transferCount,
				)

				// Agent descriptions for enriched transfer events
				agentDescs := map[string]string{
					"code_writer":   "代码读写与编辑",
					"code_executor": "命令与脚本执行",
					"file_manager":  "文件批量操作",
				}

				s.emitTimelineForRun(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "transfer",
					Agent:     event.AgentName,
					Content:   destName,
					ToolArgs:  agentDescs[destName],
					Timestamp: time.Now().UnixMilli(),
				}, run)

				// Emit thinking indicator so the user sees the sub-agent is active
				s.emitTimelineForRun(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "thinking",
					Agent:     destName,
					Timestamp: time.Now().UnixMilli(),
				}, run)
			}
		}

		// Handle message output
		if event.Output != nil && event.Output.MessageOutput != nil {
			mv := event.Output.MessageOutput
			var msg *schema.Message
			var err error
			var wasStreamed bool

			if mv.IsStreaming && mv.MessageStream != nil {
				msg, err = s.drainStreamForRun(mv.MessageStream, event.AgentName, run)
				wasStreamed = true
			} else {
				msg, err = mv.GetMessage()
			}

			if err != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
					"sessionId": sessionID,
					"error":     fmt.Sprintf("failed to get message: %v", err),
				})
				continue
			}
			if msg == nil {
				continue
			}

			// Emit tool call timeline events
			if len(msg.ToolCalls) > 0 {
				// Surface the LLM's reasoning text before tool calls.
				if msg.Content != "" {
					logger.Info("[CHAT] Reasoning text found with tool calls",
						"agent", event.AgentName,
						"content_len", len(msg.Content),
						"preview", truncateResult(msg.Content, 100),
					)
					s.emitTimelineForRun(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "reasoning",
						Agent:     event.AgentName,
						Content:   msg.Content,
						Timestamp: time.Now().UnixMilli(),
					}, run)
				} else {
					logger.Info("[CHAT] No reasoning text with tool calls (Content empty)",
						"agent", event.AgentName,
						"tool_count", len(msg.ToolCalls),
					)
				}

				for _, tc := range msg.ToolCalls {
					s.emitTimelineForRun(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "tool_call",
						Agent:     event.AgentName,
						ToolName:  tc.Function.Name,
						ToolArgs:  tc.Function.Arguments,
						ToolID:    tc.ID,
						Timestamp: time.Now().UnixMilli(),
					}, run)
				}

				// Store tool call message in session's context history
				toolMsg := &schema.Message{
					Role:      schema.Assistant,
					Content:   msg.Content,
					ToolCalls: msg.ToolCalls,
				}
				run.addMessage(toolMsg)
				currentTurnToolMessages = append(currentTurnToolMessages, toolMsg)
				// Track pending tool call IDs
				for _, tc := range msg.ToolCalls {
					pendingToolCalls[tc.ID] = pendingRuntimeToolCall{
						name: tc.Function.Name,
						args: tc.Function.Arguments,
					}
				}
				continue // Don't fall through to allContents — tool call content is already stored
			}

			// Emit tool result events
			if msg.Role == schema.Tool && msg.ToolCallID != "" {
				resultContent := truncateResult(msg.Content, 1000)
				if call, ok := pendingToolCalls[msg.ToolCallID]; ok {
					resultContent = timelineToolResultContent(call.name, msg.Content)
				}
				s.emitTimelineForRun(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "tool_result",
					Agent:     event.AgentName,
					Content:   resultContent,
					ToolID:    msg.ToolCallID,
					Timestamp: time.Now().UnixMilli(),
				}, run)

				// Store tool result in session's context history.
				toolResultMsg := &schema.Message{
					Role:       schema.Tool,
					Content:    msg.Content,
					ToolCallID: msg.ToolCallID,
				}
				run.addMessage(toolResultMsg)
				currentTurnToolResultMessages = append(currentTurnToolResultMessages, toolResultMsg)
				if call, ok := pendingToolCalls[msg.ToolCallID]; ok {
					if call.name == tools.ToolSearchName {
						s.recordToolSearchOutputForRun(run, msg)
					}
					run.recordRuntimeToolResult(call.name, call.args, msg.Content, s.now().UnixMilli())
					s.emitRuntimeWorktreeToolEvent(sessionID, call.name, call.args, msg.Content)
				}
				// Mark this tool call as resolved
				delete(pendingToolCalls, msg.ToolCallID)

				// Debounced intermediate save
				maybeSave()

				// Emit thinking indicator after sub-agent tool result
				if event.AgentName != "coding_agent" {
					s.emitTimelineForRun(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "thinking",
						Agent:     event.AgentName,
						Timestamp: time.Now().UnixMilli(),
					}, run)
				}

				continue
			}

			// Emit message event (assistant messages only)
			if msg.Content != "" && msg.Role == schema.Assistant {
				// Dedup: skip if same agent sent identical content
				if prev, ok := lastContentByAgent[event.AgentName]; ok && prev == msg.Content {
					continue
				}
				lastContentByAgent[event.AgentName] = msg.Content
				allContents = append(allContents, msg.Content)

				// Only emit timeline event if content was NOT already streamed
				if !wasStreamed {
					s.emitTimelineForRun(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "message",
						Agent:     event.AgentName,
						Content:   msg.Content,
						Timestamp: time.Now().UnixMilli(),
					}, run)
				}
			}
		}
	}

	// Fix orphaned tool calls: inject synthetic error responses for any tool_call_ids
	// that were stored but never received a matching tool result.
	wasPreempted := runtimeTurnSignalClosed(preempted)
	if len(pendingToolCalls) > 0 {
		if wasPreempted {
			removed := run.removeCurrentTurnToolCallsForIDs(pendingToolCalls, currentTurnToolMessages, currentTurnToolResultMessages)
			logger.Info("[CHAT] Dropped unresolved tool-call history for preempted turn",
				"pending_tool_calls", len(pendingToolCalls), "removed_messages", removed, "session", sessionID)
		} else {
			for toolCallID := range pendingToolCalls {
				logger.Warn("[CHAT] Injecting synthetic tool result for orphaned tool_call",
					"tool_call_id", toolCallID, "session", sessionID)
				run.addToolResult(toolCallID, "Error: tool execution failed or was interrupted")
			}
		}
	}

	return strings.Join(allContents, "\n\n"), transferCount, false, wasPreempted
}

func pendingToolCallIDs(calls map[string]pendingRuntimeToolCall) []string {
	if len(calls) == 0 {
		return nil
	}
	ids := make([]string, 0, len(calls))
	for id := range calls {
		if id != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func runtimeAgentErrorContent(err error, sessionID string) string {
	if err == nil {
		return ""
	}
	content := fmt.Sprintf("Error: %v", err)
	if strings.Contains(err.Error(), "No tool output found") {
		content += fmt.Sprintf("\nHistory repair failed; please report session id %s.", sessionID)
	}
	return content
}

func (s *ChatService) recordToolSearchOutputForRun(run *SessionRun, msg *schema.Message) {
	if run == nil {
		return
	}
	matches := toolSearchMatchesFromMessage(msg)
	if len(matches) == 0 {
		return
	}

	s.mu.Lock()
	generation := run.activeBundleGeneration
	bundle := s.findBundleByGenerationLocked(generation)
	s.mu.Unlock()
	if bundle == nil || bundle.MCPCatalog == nil {
		return
	}

	sessionID := run.sessionID
	now := s.now().UnixMilli()
	for _, name := range matches {
		entry, ok := bundle.MCPCatalog.LookupExact(name)
		if !ok || !entry.ShouldDefer || entry.AlwaysLoad {
			continue
		}
		s.AddDiscoveredTool(sessionID, model.DiscoveredToolRecord{
			CanonicalName: entry.CanonicalName,
			Server:        entry.Server,
			Kind:          entry.Kind,
			DiscoveredAt:  now,
		})
	}
}

func toolSearchMatchesFromMessage(msg *schema.Message) []string {
	if msg == nil {
		return nil
	}
	seen := make(map[string]struct{})
	matches := make([]string, 0)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		matches = append(matches, name)
	}

	if strings.TrimSpace(msg.Content) != "" {
		var output struct {
			Matches []string `json:"matches"`
		}
		if err := json.Unmarshal([]byte(msg.Content), &output); err == nil {
			for _, name := range output.Matches {
				add(name)
			}
		}
	}
	for _, part := range msg.UserInputMultiContent {
		if part.Type != schema.ChatMessagePartTypeToolSearchResult || part.ToolSearchResult == nil {
			continue
		}
		for _, info := range part.ToolSearchResult.Tools {
			if info == nil {
				continue
			}
			add(info.Name)
		}
	}
	return matches
}

func timelineToolResultLimit(toolName string) int {
	switch toolName {
	case tools.RuntimeToolWorktreeDiff:
		return 24000
	case tools.RuntimeToolWrite, "write_file", tools.RuntimeToolEdit, "str_replace_editor":
		return 8000
	case tools.RuntimeToolWorktreeMerge, tools.RuntimeToolEnterWorktree, tools.RuntimeToolExitWorktree:
		return 6000
	default:
		return 1000
	}
}

func timelineToolResultContent(toolName string, result string) string {
	limit := timelineToolResultLimit(toolName)
	switch toolName {
	case tools.RuntimeToolWorktreeDiff:
		return truncateWorktreeDiffTimelineJSON(result, limit)
	case tools.RuntimeToolWorktreeMerge:
		return truncateWorktreeMergeTimelineJSON(result, limit)
	case tools.RuntimeToolWrite, "write_file", tools.RuntimeToolEdit, "str_replace_editor":
		return truncateFileDiffTimelineJSON(toolName, result, limit)
	}
	return truncateResult(result, limit)
}

func truncateFileDiffTimelineJSON(toolName string, result string, limit int) string {
	if limit <= 0 || len(result) <= limit {
		return result
	}
	switch toolName {
	case tools.RuntimeToolWrite, "write_file":
		var out tools.WriteOutput
		if err := json.Unmarshal([]byte(result), &out); err != nil {
			return truncateResult(result, limit)
		}
		next, ok := encodeTimelinePatchResult(limit, out.Patch, func(patch string, truncated bool) (string, error) {
			out.Patch = patch
			out.Truncated = out.Truncated || truncated
			encoded, err := json.Marshal(out)
			return string(encoded), err
		})
		if ok {
			return next
		}
	case tools.RuntimeToolEdit, "str_replace_editor":
		var out tools.EditOutput
		if err := json.Unmarshal([]byte(result), &out); err != nil {
			return truncateResult(result, limit)
		}
		next, ok := encodeTimelinePatchResult(limit, out.Patch, func(patch string, truncated bool) (string, error) {
			out.Patch = patch
			out.Truncated = out.Truncated || truncated
			encoded, err := json.Marshal(out)
			return string(encoded), err
		})
		if ok {
			return next
		}
	}
	return truncateResult(result, limit)
}

func encodeTimelinePatchResult(limit int, patch string, encode func(string, bool) (string, error)) (string, bool) {
	encoded, err := encode(patch, false)
	if err != nil {
		return "", false
	}
	if len(encoded) <= limit {
		return encoded, true
	}
	for i := 0; i < 16 && len(encoded) > limit; i++ {
		over := len(encoded) - limit
		nextPatch, truncated := shrinkTimelineText(patch, over+256)
		if !truncated {
			return "", false
		}
		patch = nextPatch
		encoded, err = encode(patch, true)
		if err != nil {
			return "", false
		}
	}
	if len(encoded) > limit {
		return "", false
	}
	return encoded, true
}

func truncateWorktreeDiffTimelineJSON(result string, limit int) string {
	if limit <= 0 || len(result) <= limit {
		return result
	}
	var out tools.WorktreeDiffOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		return truncateResult(result, limit)
	}
	if out.Diff != "" && out.UntrackedDiff != "" {
		// DiffWorktree serializes untracked patches into Diff as the combined
		// patch payload. Timeline JSON can omit the duplicate field and still
		// render the full review patch in the frontend.
		out.UntrackedDiff = ""
	}

	encoded, err := json.Marshal(out)
	if err != nil {
		return truncateResult(result, limit)
	}
	if len(encoded) <= limit {
		return string(encoded)
	}

	for i := 0; i < 24 && len(encoded) > limit; i++ {
		over := len(encoded) - limit
		reduced := false
		switch {
		case out.Diff != "":
			next, truncated := shrinkTimelineText(out.Diff, over+512)
			out.Diff = next
			out.Truncated = out.Truncated || truncated
			reduced = truncated
		case out.UntrackedDiff != "":
			next, truncated := shrinkTimelineText(out.UntrackedDiff, over+512)
			out.UntrackedDiff = next
			out.UntrackedTruncated = out.UntrackedTruncated || truncated
			reduced = truncated
		case out.DiffStat != "":
			next, truncated := shrinkTimelineText(out.DiffStat, over+256)
			out.DiffStat = next
			reduced = truncated
		case out.Status != "":
			next, truncated := shrinkTimelineText(out.Status, over+256)
			out.Status = next
			reduced = truncated
		case out.Message != "":
			next, truncated := shrinkTimelineText(out.Message, over+128)
			out.Message = next
			reduced = truncated
		}
		if !reduced {
			return truncateResult(result, limit)
		}
		encoded, err = json.Marshal(out)
		if err != nil {
			return truncateResult(result, limit)
		}
	}
	if len(encoded) > limit {
		return truncateResult(result, limit)
	}
	return string(encoded)
}

func truncateWorktreeMergeTimelineJSON(result string, limit int) string {
	if limit <= 0 || len(result) <= limit {
		return result
	}
	var out tools.WorktreeMergeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		return truncateResult(result, limit)
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return truncateResult(result, limit)
	}
	for i := 0; i < 16 && len(encoded) > limit; i++ {
		over := len(encoded) - limit
		reduced := false
		switch {
		case out.MergeOutput != "":
			next, truncated := shrinkTimelineText(out.MergeOutput, over+512)
			out.MergeOutput = next
			reduced = truncated
		case len(out.ConflictFiles) > 0:
			next, truncated := shrinkTimelineConflictFiles(out.ConflictFiles, over+512)
			out.ConflictFiles = next
			reduced = truncated
		case out.RecoveryHint != "":
			next, truncated := shrinkTimelineText(out.RecoveryHint, over+256)
			out.RecoveryHint = next
			reduced = truncated
		case out.Message != "":
			next, truncated := shrinkTimelineText(out.Message, over+128)
			out.Message = next
			reduced = truncated
		}
		if !reduced {
			return truncateResult(result, limit)
		}
		encoded, err = json.Marshal(out)
		if err != nil {
			return truncateResult(result, limit)
		}
	}
	if len(encoded) > limit {
		return truncateResult(result, limit)
	}
	return string(encoded)
}

func shrinkTimelineConflictFiles(files []string, removeBytes int) ([]string, bool) {
	if len(files) == 0 {
		return files, false
	}
	next := append([]string(nil), files...)
	changed := false
	if len(next) > 20 {
		omitted := len(next) - 20
		next = append(next[:20], fmt.Sprintf("... (%d more files)", omitted))
		changed = true
	}
	for i, file := range next {
		if len(file) <= 160 {
			continue
		}
		next[i] = file[:120] + "..." + file[len(file)-32:]
		changed = true
	}
	if changed {
		return next, true
	}
	if len(next) > 1 {
		omitted := len(next) - 1
		return []string{next[0], fmt.Sprintf("... (%d more files)", omitted)}, true
	}
	shortened, truncated := shrinkTimelineText(next[0], removeBytes+128)
	if truncated {
		next[0] = shortened
		return next, true
	}
	return files, false
}

func shrinkTimelineText(value string, removeBytes int) (string, bool) {
	if value == "" {
		return value, false
	}
	marker := "\n... (truncated for timeline)"
	target := len(value) - removeBytes
	if target > len(value)-1 {
		target = len(value) - 1
	}
	if target <= len(marker) {
		return "", true
	}
	if target >= len(value) {
		return value, false
	}
	return value[:target-len(marker)] + marker, true
}

func (s *ChatService) emitRuntimeWorktreeToolEvent(sessionID, toolName, argsJSON, result string) {
	if strings.HasPrefix(strings.TrimSpace(result), "Error:") {
		return
	}
	activeContainerID := s.activeWorkspaceContainerID(sessionID)
	now := time.Now().UnixMilli()
	switch toolName {
	case tools.RuntimeToolEnterWorktree, tools.RuntimeToolExitWorktree, tools.RuntimeToolWorktreeMerge:
		wailsEmit(s.ctx, "runtime:worktree_changed", map[string]string{"sessionId": sessionID, "action": toolName})
		wailsEmit(s.ctx, "workspace:changed", WorkspaceChangedEvent{
			SessionID:   sessionID,
			ContainerID: activeContainerID,
			Source:      "agent",
			Action:      toolName,
			CreatedAt:   now,
		})
	case tools.RuntimeToolWorktreeDiff:
		wailsEmit(s.ctx, "runtime:worktree_reviewed", map[string]string{"sessionId": sessionID})
	}
	if path, ok := runtimeToolWorkspaceChangePath(toolName, argsJSON, result); ok {
		wailsEmit(s.ctx, "workspace:changed", WorkspaceChangedEvent{
			SessionID:   sessionID,
			ContainerID: activeContainerID,
			Path:        path,
			Source:      "agent",
			Action:      toolName,
			CreatedAt:   now,
		})
	}
}

func runtimeToolWorkspaceChangePath(toolName, argsJSON, result string) (string, bool) {
	switch toolName {
	case tools.RuntimeToolWrite:
		var out tools.WriteOutput
		if err := json.Unmarshal([]byte(result), &out); err == nil && out.FilePath != "" {
			return out.FilePath, true
		}
		return "", false
	case "write_file":
		var legacyOut struct {
			Success bool `json:"success"`
		}
		if err := json.Unmarshal([]byte(result), &legacyOut); err == nil && legacyOut.Success {
			if path := runtimeToolArgPath(argsJSON); path != "" {
				return path, true
			}
		}
		return "", false
	case tools.RuntimeToolEdit:
		var out tools.EditOutput
		if err := json.Unmarshal([]byte(result), &out); err == nil && out.FilePath != "" {
			return out.FilePath, true
		}
		return "", false
	case "str_replace_editor":
		var out tools.EditOutput
		if err := json.Unmarshal([]byte(result), &out); err == nil && out.FilePath != "" {
			return out.FilePath, true
		}
		args := runtimeToolArgs(argsJSON)
		if strings.EqualFold(args.Command, "view") {
			return "", false
		}
		if path := args.FilePathOrPath(); path != "" {
			return path, true
		}
		return "", false
	case tools.RuntimeToolLSPEdit:
		var out tools.LSPEditOutput
		if err := json.Unmarshal([]byte(result), &out); err == nil {
			if len(out.ChangedFiles) > 1 {
				return "", out.EditCount > 0
			}
			if len(out.ChangedFiles) > 0 {
				return out.ChangedFiles[0], out.EditCount > 0
			}
			return out.FilePath, out.EditCount > 0
		}
		return "", false
	case tools.RuntimeToolNotebookEdit:
		var out tools.NotebookEditOutput
		if err := json.Unmarshal([]byte(result), &out); err == nil {
			return out.FilePath, out.Command != "view"
		}
		return "", false
	case tools.RuntimeToolBash, "shell_execute":
		if !bashCommandLooksMutating(runtimeToolArgs(argsJSON).Command) {
			return "", false
		}
		var out tools.BashOutput
		if err := json.Unmarshal([]byte(result), &out); err == nil {
			return "", out.BackgroundTaskID == "" && out.ExitCode == 0
		}
		var shellOut tools.ShellOutput
		if err := json.Unmarshal([]byte(result), &shellOut); err == nil {
			return "", shellOut.ExitCode == 0
		}
		return "", false
	default:
		return "", false
	}
}

type runtimeToolArgsSnapshot struct {
	Path     string `json:"path"`
	FilePath string `json:"file_path"`
	Command  string `json:"command"`
}

func (a runtimeToolArgsSnapshot) FilePathOrPath() string {
	if a.FilePath != "" {
		return a.FilePath
	}
	return a.Path
}

func runtimeToolArgs(argsJSON string) runtimeToolArgsSnapshot {
	var args runtimeToolArgsSnapshot
	_ = json.Unmarshal([]byte(argsJSON), &args)
	return args
}

func runtimeToolArgPath(argsJSON string) string {
	return runtimeToolArgs(argsJSON).FilePathOrPath()
}

func bashCommandLooksMutating(command string) bool {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return false
	}
	lowered := strings.ToLower(cmd)
	if strings.ContainsAny(lowered, "><") {
		return true
	}
	mutatingNeedles := []string{
		"touch ", "mkdir ", "rm ", "mv ", "cp ", "chmod ", "chown ", "ln ",
		"tee ", "sed -i", "perl -i", "patch ", "git apply", "git checkout",
		"git switch", "git restore", "git reset", "git clean", "go mod tidy",
		"gofmt -w", "prettier --write", "npm install", "npm i ", "pnpm install",
		"yarn install", "bun install", "cargo add", "cargo fmt", "ruff --fix",
		"python -m pip install", "pip install",
	}
	for _, needle := range mutatingNeedles {
		needle = strings.TrimSpace(needle)
		if strings.Contains(lowered, needle+" ") || strings.HasPrefix(lowered, needle+" ") || lowered == needle {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Interrupt handling
// ---------------------------------------------------------------------------

// handleInterruptForRun processes an interrupt context for a specific session run.
func (s *ChatService) handleInterruptForRun(interruptCtx *adk.InterruptCtx, checkpointID string, run *SessionRun, toolCallIDs []string) {
	s.mu.Lock()
	run.pendingInterrupt = &PendingInterrupt{
		CheckpointID:     checkpointID,
		InterruptID:      interruptCtx.ID,
		BundleGeneration: run.activeBundleGeneration,
		RunnerKind:       run.activeRunnerKind,
		Info:             interruptCtx.Info,
		Objective:        run.currentObjective(),
		ToolCallIDs:      append([]string(nil), toolCallIDs...),
	}
	run.activeBundleGeneration = 0
	run.activeRunnerKind = ""
	s.mu.Unlock()
	s.emitRunState(run.sessionID)

	// Determine interrupt type and emit event
	var evt InterruptEvent
	evt.InterruptID = interruptCtx.ID
	evt.CheckpointID = checkpointID
	evt.SessionID = run.sessionID

	switch info := interruptCtx.Info.(type) {
	case *tools.FollowUpInfo:
		evt.Type = "followup"
		evt.Questions = info.Questions
		logger.Info("[CHAT] Interrupt: follow-up questions", "count", len(info.Questions), "session", run.sessionID)
	case *tools.ChoiceInfo:
		evt.Type = "choice"
		evt.Question = info.Question
		for _, opt := range info.Options {
			evt.Options = append(evt.Options, InterruptOption{
				Label:       opt.Label,
				Description: opt.Description,
			})
		}
		logger.Info("[CHAT] Interrupt: choice", "question", info.Question, "options", len(info.Options), "session", run.sessionID)
	default:
		logger.Warn("[CHAT] Unknown interrupt type", "type", fmt.Sprintf("%T", interruptCtx.Info))
		evt.Type = "followup"
		evt.Questions = []string{fmt.Sprintf("%v", interruptCtx.Info)}
	}

	wailsEmit(s.ctx, "agent:interrupt", evt)
	s.emitTimelineForRun(TimelineEvent{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      "interrupt",
		Agent:     "system",
		Content:   fmt.Sprintf("Waiting for user input: %s", evt.Type),
		Timestamp: time.Now().UnixMilli(),
	}, run)
}

func (s *ChatService) resolvePendingRunnerLocked(run *SessionRun) (*RunnerBundle, *adk.Runner, *PendingInterrupt, error) {
	if run == nil {
		return nil, nil, nil, fmt.Errorf("no active session")
	}
	pending := run.pendingInterrupt
	if pending == nil {
		return nil, nil, nil, fmt.Errorf("no pending interrupt to resume")
	}
	bundle := s.findBundleByGenerationLocked(pending.BundleGeneration)
	if bundle == nil {
		return nil, nil, nil, fmt.Errorf("runner bundle generation %d is no longer available for resume", pending.BundleGeneration)
	}
	runner := runnerForKind(bundle, pending.RunnerKind)
	if runner == nil {
		return nil, nil, nil, fmt.Errorf("runner %s is unavailable for bundle generation %d", pending.RunnerKind, pending.BundleGeneration)
	}
	return bundle, runner, pending, nil
}

// ---------------------------------------------------------------------------
// Resume
// ---------------------------------------------------------------------------

// ResumeWithAnswer resumes execution after the user answers follow-up questions.
func (s *ChatService) ResumeWithAnswer(answer string) error {
	s.mu.Lock()
	run := s.activeRun()
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("no active session")
	}
	if run.running || run.starting {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}
	pending := run.pendingInterrupt
	if pending == nil {
		s.mu.Unlock()
		return fmt.Errorf("no pending interrupt to resume")
	}
	sessionID := run.sessionID
	s.mu.Unlock()
	if err := s.validateActiveSessionSandbox(sessionID); err != nil {
		return err
	}
	s.mu.Lock()
	run = s.sessions[sessionID]
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	if run.running || run.starting {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}
	if run.pendingInterrupt != pending {
		s.mu.Unlock()
		return fmt.Errorf("pending interrupt changed for session %s", sessionID)
	}
	item := s.newRuntimeResumeAnswerItem(sessionID, pending, answer)
	run.pendingInterrupt = nil
	s.resetRuntimeTurnLoopLocked(run)
	loop := s.ensureRuntimeTurnLoopLocked(sessionID, run)
	s.mu.Unlock()

	logger.Info("[CHAT] Resuming after follow-up", "answer_length", len(answer), "session", sessionID)
	if ok, _ := loop.Push(item); !ok {
		s.mu.Lock()
		if run = s.sessions[sessionID]; run != nil && run.pendingInterrupt == nil {
			run.pendingInterrupt = pending
		}
		s.mu.Unlock()
		return fmt.Errorf("runtime turn loop is stopped for session %s", sessionID)
	}
	s.mu.Lock()
	if run = s.sessions[sessionID]; run == nil {
		s.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	if run.turnLoop != loop {
		if run.pendingInterrupt == nil {
			run.pendingInterrupt = pending
		}
		s.mu.Unlock()
		return fmt.Errorf("runtime turn loop changed for session %s", sessionID)
	}
	s.startRuntimeTurnLoopLocked(sessionID, run, loop)
	s.mu.Unlock()
	s.emitRunState(sessionID)
	return nil
}

// ResumeWithChoice resumes execution after the user selects a choice.
func (s *ChatService) ResumeWithChoice(selectedIndex int) error {
	s.mu.Lock()
	run := s.activeRun()
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("no active session")
	}
	if run.running || run.starting {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}
	pending := run.pendingInterrupt
	if pending == nil {
		s.mu.Unlock()
		return fmt.Errorf("no pending interrupt to resume")
	}
	sessionID := run.sessionID
	s.mu.Unlock()
	if err := s.validateActiveSessionSandbox(sessionID); err != nil {
		return err
	}
	s.mu.Lock()
	run = s.sessions[sessionID]
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	if run.running || run.starting {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}
	if run.pendingInterrupt != pending {
		s.mu.Unlock()
		return fmt.Errorf("pending interrupt changed for session %s", sessionID)
	}
	item := s.newRuntimeResumeChoiceItem(sessionID, pending, selectedIndex)
	run.pendingInterrupt = nil
	s.resetRuntimeTurnLoopLocked(run)
	loop := s.ensureRuntimeTurnLoopLocked(sessionID, run)
	s.mu.Unlock()

	logger.Info("[CHAT] Resuming after choice", "selected", selectedIndex, "session", sessionID)
	if ok, _ := loop.Push(item); !ok {
		s.mu.Lock()
		if run = s.sessions[sessionID]; run != nil && run.pendingInterrupt == nil {
			run.pendingInterrupt = pending
		}
		s.mu.Unlock()
		return fmt.Errorf("runtime turn loop is stopped for session %s", sessionID)
	}
	s.mu.Lock()
	if run = s.sessions[sessionID]; run == nil {
		s.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	if run.turnLoop != loop {
		if run.pendingInterrupt == nil {
			run.pendingInterrupt = pending
		}
		s.mu.Unlock()
		return fmt.Errorf("runtime turn loop changed for session %s", sessionID)
	}
	s.startRuntimeTurnLoopLocked(sessionID, run, loop)
	s.mu.Unlock()
	s.emitRunState(sessionID)
	return nil
}

// ---------------------------------------------------------------------------
// Stop
// ---------------------------------------------------------------------------

// StopGeneration cancels the currently running agent in the active session.
func (s *ChatService) StopGeneration() error {
	s.mu.Lock()
	run := s.activeRun()
	if run == nil {
		s.mu.Unlock()
		return nil
	}
	if run.starting {
		if run.cancelFn != nil {
			run.cancelFn()
		} else if run.turnLoop != nil {
			run.turnLoop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
		}
		done := run.startDone
		s.mu.Unlock()
		if done != nil {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				logger.Warn("[CHAT] StopGeneration timed out during startup", "session", s.activeSessionID)
			}
		}
		return nil
	}
	if !run.running {
		sessionID := run.sessionID
		hadInterrupt := run.pendingInterrupt != nil
		repairSessionID, repair := s.clearPendingInterruptLocked(run, orphanRepairStoppedReason)
		s.logHistoryRepair(repairSessionID, "stop_idle", repair)
		if hadInterrupt {
			s.resetRuntimeTurnLoopLocked(run)
		}
		s.cleanupRetiredBundlesLocked()
		s.mu.Unlock()
		s.saveSessionAfterHistoryRepair(repairSessionID, repair)
		if hadInterrupt {
			s.deleteRuntimeTurnCheckpoint(sessionID)
			s.emitRunState(sessionID)
		}
		return nil
	}
	if run.turnLoop != nil {
		run.turnLoop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
	} else if run.cancelFn != nil {
		run.cancelFn()
	}
	repairSessionID, repair := s.clearPendingInterruptLocked(run, orphanRepairStoppedReason)
	s.logHistoryRepair(repairSessionID, "stop_running", repair)
	s.cleanupRetiredBundlesLocked()
	done := run.runDone
	s.mu.Unlock()
	s.saveSessionAfterHistoryRepair(repairSessionID, repair)

	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			logger.Warn("[CHAT] StopGeneration timed out", "session", s.activeSessionID)
		}
	}
	return nil
}

// StopSessionGeneration cancels a running agent in a specific session.
func (s *ChatService) StopSessionGeneration(sessionID string) {
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return
	}
	if run.starting {
		if run.cancelFn != nil {
			run.cancelFn()
		} else if run.turnLoop != nil {
			run.turnLoop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
		}
		done := run.startDone
		s.mu.Unlock()
		if done != nil {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				logger.Warn("[CHAT] StopSessionGeneration timed out during startup", "session", sessionID)
			}
		}
		return
	}
	if !run.running {
		hadInterrupt := run.pendingInterrupt != nil
		repairSessionID, repair := s.clearPendingInterruptLocked(run, orphanRepairStoppedReason)
		s.logHistoryRepair(repairSessionID, "stop_session_idle", repair)
		if hadInterrupt {
			s.resetRuntimeTurnLoopLocked(run)
		}
		s.mu.Unlock()
		s.saveSessionAfterHistoryRepair(repairSessionID, repair)
		if hadInterrupt {
			s.deleteRuntimeTurnCheckpoint(sessionID)
			s.emitRunState(sessionID)
		}
		return
	}
	if run.turnLoop != nil {
		run.turnLoop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
	} else if run.cancelFn != nil {
		run.cancelFn()
	}
	repairSessionID, repair := s.clearPendingInterruptLocked(run, orphanRepairStoppedReason)
	s.logHistoryRepair(repairSessionID, "stop_session_running", repair)
	s.cleanupRetiredBundlesLocked()
	done := run.runDone
	s.mu.Unlock()
	s.saveSessionAfterHistoryRepair(repairSessionID, repair)

	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			logger.Warn("[CHAT] StopSessionGeneration timed out", "session", sessionID)
		}
	}
}

// ---------------------------------------------------------------------------
// History & state access
// ---------------------------------------------------------------------------

// ClearHistory resets the conversation history for the active session.
func (s *ChatService) ClearHistory() error {
	s.mu.Lock()

	run := s.activeRun()
	if run == nil {
		s.mu.Unlock()
		return nil
	}

	run.clearSessionState()
	run.pendingInterrupt = nil
	s.resetRuntimeTurnLoopLocked(run)
	s.invalidateRunners()
	s.cleanupRetiredBundlesLocked()
	sessionID := s.activeSessionID
	sessionSvc := s.sessionService
	runtimeTasks := s.runtimeTasks
	s.mu.Unlock()

	s.deleteRuntimeTurnCheckpoint(sessionID)
	tools.ClearTodosForSession(sessionID)
	if runtimeTasks != nil {
		runtimeTasks.ClearTaskItemsForSession(sessionID)
	}
	if sessionSvc != nil && sessionID != "" {
		if err := sessionSvc.SaveSessionByID(sessionID); err != nil {
			logger.Warn("[CHAT] Failed to schedule clear-history save", "session", sessionID, "error", err)
		}
	}
	return nil
}

// Timeline returns the timeline collector for the active session.
func (s *ChatService) Timeline() *agentctx.TimelineCollector {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	if run == nil {
		return agentctx.NewTimelineCollector() // return empty if no session
	}
	return run.timeline
}

// StreamingState returns the current streaming state for the active session (nil if not streaming).
func (s *ChatService) StreamingState() *model.StreamingState {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	if run == nil {
		return nil
	}
	return run.streamingStateSnapshot()
}

// CtxEngine returns the context engine for the active session.
func (s *ChatService) CtxEngine() *agentctx.Engine {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	if run == nil {
		return nil
	}
	return run.ctxEngine
}

// SessionCtxEngine returns the context engine for a specific session.
func (s *ChatService) SessionCtxEngine(sessionID string) *agentctx.Engine {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	return run.ctxEngine
}

// SessionTimeline returns the timeline collector for a specific session.
func (s *ChatService) SessionTimeline(sessionID string) *agentctx.TimelineCollector {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	return run.timeline
}

// SessionStreamingState returns the streaming state for a specific session.
func (s *ChatService) SessionStreamingState(sessionID string) *model.StreamingState {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	return run.streamingStateSnapshot()
}

// ExportSessionSnapshot exports a single consistent snapshot for the given session.
// The snapshot is copied under the session's state lock; disk IO must happen elsewhere.
func (s *ChatService) exportDeferredSurfaceDebugForSnapshot(sessionID string) (*DeferredSurfaceDebug, error) {
	if !s.runtimeOptionsSnapshot().DeferredSurfaceDebugAPIEnabled {
		return nil, nil
	}
	return s.buildDeferredSurfaceDebug(sessionID, true)
}

func (s *ChatService) ExportSessionSnapshot(sessionID string) (*SessionSnapshot, error) {
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		debug, err := s.exportDeferredSurfaceDebugForSnapshot(sessionID)
		if err != nil {
			return nil, err
		}
		return &SessionSnapshot{
			SessionData:          model.DefaultSessionData(),
			DeferredSurfaceDebug: debug,
		}, nil
	}

	s.refreshRuntimeContextCompact(sessionID, run)
	snapshot := run.snapshot()
	debug, err := s.exportDeferredSurfaceDebugForSnapshot(sessionID)
	if err != nil {
		return nil, err
	}
	snapshot.DeferredSurfaceDebug = debug
	return snapshot, nil
}

func logSessionDataNormalizeWarnings(sessionID, source string, warnings []string) {
	for _, warning := range warnings {
		logger.Warn("[CHAT] Normalized session data",
			"session", sessionID,
			"source", source,
			"warning", warning,
		)
	}
}

func (s *ChatService) restoreNormalizedSessionData(sessionID string, data *model.SessionData) {
	s.mu.Lock()
	run := s.getOrCreateRun(sessionID)
	tasks := s.runtimeTasks
	workspaces := s.runtimeWorkspaces
	s.mu.Unlock()
	repair := run.importSessionData(data)
	s.logHistoryRepair(sessionID, "restore_session", repair)
	if data != nil && data.RuntimeContextCompact != nil {
		if tasks != nil {
			tasks.RestoreCompactTasks(sessionID, data.RuntimeContextCompact.Tasks)
			tasks.RestoreCompactTaskItems(sessionID, data.RuntimeContextCompact.TaskItems)
		}
		if workspaces != nil {
			workspaces.RestoreCompactSnapshot(sessionID, data.RuntimeContextCompact.Workspace)
		}
		tools.RestoreTodosForSession(sessionID, data.RuntimeContextCompact.Todos)
	} else {
		if tasks != nil {
			tasks.RestoreCompactTaskItems(sessionID, nil)
		}
		tools.ClearTodosForSession(sessionID)
	}
	if repair.Count > 0 {
		go s.saveSessionAfterHistoryRepair(sessionID, repair)
	}
}

func (s *ChatService) RestoreSessionData(sessionID string, data *model.SessionData) {
	normalized, warnings := model.NormalizeSessionData(data)
	logSessionDataNormalizeWarnings(sessionID, "direct_restore", warnings)
	s.restoreNormalizedSessionData(sessionID, normalized)
}

func (s *ChatService) AddDiscoveredTool(sessionID string, record model.DiscoveredToolRecord) bool {
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		return false
	}
	return run.upsertDiscoveredTool(record)
}

func (s *ChatService) ReplaceDiscoveredTools(sessionID string, records []model.DiscoveredToolRecord) {
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		return
	}
	run.replaceDiscoveredTools(records)
}

func (s *ChatService) PruneDiscoveredToolsForSave(_ string, records []model.DiscoveredToolRecord) []model.DiscoveredToolRecord {
	if len(records) == 0 {
		return nil
	}
	records = s.pruneExperimentalDeferredDiscoveries(records)
	if len(records) == 0 {
		return nil
	}

	cfg, currentDigest, err := s.currentConfigSnapshot()
	if err != nil || cfg == nil {
		return cloneAndSortDiscoveredTools(records)
	}
	configuredServers, err := configuredMCPServerIdentityDigests(cfg)
	if err != nil {
		return cloneAndSortDiscoveredTools(records)
	}

	s.mu.Lock()
	var (
		catalog *tools.ToolCatalog
		cache   map[string]cachedMCPServerSurface
	)
	if s.installedBundle != nil &&
		s.installedBundle.ConfigDigest == currentDigest &&
		bundleSurfaceFreshForPruning(s.installedBundle, s.now(), s.freshnessTTL) {
		catalog = s.installedBundle.MCPCatalog
		cache = cloneSurfaceCache(s.installedBundle.CachedSurfaceMetadataByServer)
	}
	s.mu.Unlock()

	knownDeferredCanonicalNames := make(map[string]struct{})
	serverHasTrustedToolMetadata := make(map[string]bool)
	trustedServerCanonicalNames := make(map[string]map[string]struct{})
	if catalog != nil {
		for _, entry := range catalog.Entries() {
			if !entry.IsMcp || !entry.ShouldDefer {
				continue
			}
			knownDeferredCanonicalNames[entry.CanonicalName] = struct{}{}
			if entry.Server == "" || entry.IsResourceTool {
				continue
			}
			serverHasTrustedToolMetadata[entry.Server] = true
			if _, ok := trustedServerCanonicalNames[entry.Server]; !ok {
				trustedServerCanonicalNames[entry.Server] = make(map[string]struct{})
			}
			trustedServerCanonicalNames[entry.Server][entry.CanonicalName] = struct{}{}
		}
	}
	for server := range cache {
		expectedDigest, ok := configuredServers[server]
		if !ok {
			continue
		}
		trustedCacheEntry, trusted := matchingSurfaceCacheEntry(cache, server, expectedDigest)
		if !trusted {
			continue
		}
		if !trustedCacheEntry.HasToolMetadata {
			continue
		}
		serverHasTrustedToolMetadata[server] = true
		if _, exists := trustedServerCanonicalNames[server]; !exists {
			trustedServerCanonicalNames[server] = make(map[string]struct{})
		}
		for _, entry := range trustedCacheEntry.ActionEntries {
			if !entry.IsMcp || !entry.ShouldDefer {
				continue
			}
			trustedServerCanonicalNames[server][entry.CanonicalName] = struct{}{}
		}
	}

	pruned := make([]model.DiscoveredToolRecord, 0, len(records))
	for _, record := range records {
		if record.CanonicalName == "" {
			continue
		}
		if record.Server != "" {
			if _, ok := configuredServers[record.Server]; !ok {
				continue
			}
		}
		if _, ok := knownDeferredCanonicalNames[record.CanonicalName]; ok {
			pruned = append(pruned, record)
			continue
		}
		if record.Server != "" && serverHasTrustedToolMetadata[record.Server] {
			if _, ok := trustedServerCanonicalNames[record.Server][record.CanonicalName]; !ok {
				continue
			}
		}
		pruned = append(pruned, record)
	}
	return cloneAndSortDiscoveredTools(pruned)
}

func (s *ChatService) pruneExperimentalDeferredDiscoveries(records []model.DiscoveredToolRecord) []model.DiscoveredToolRecord {
	if len(records) == 0 {
		return nil
	}
	if s.runtimeOptionsSnapshot().DevDeferredBuiltinSampleEnabled {
		return records
	}
	pruned := make([]model.DiscoveredToolRecord, 0, len(records))
	for _, record := range records {
		if record.CanonicalName == tools.DevDeferredBuiltinSampleCanonicalName {
			continue
		}
		pruned = append(pruned, record)
	}
	return pruned
}

func cloneAndSortDiscoveredTools(records []model.DiscoveredToolRecord) []model.DiscoveredToolRecord {
	if len(records) == 0 {
		return nil
	}
	out := make([]model.DiscoveredToolRecord, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		if record.CanonicalName == "" {
			continue
		}
		if _, exists := seen[record.CanonicalName]; exists {
			continue
		}
		seen[record.CanonicalName] = struct{}{}
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CanonicalName < out[j].CanonicalName
	})
	return out
}

// GetSessionRunSnapshot returns a snapshot of a session's run state for the
// SessionSwitchedEvent. Safe to call from SessionService.
func (s *ChatService) GetSessionRunSnapshot(sessionID string) (running bool, currentAgent, mode string, interrupt *InterruptEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	if !ok {
		return false, "", model.ModeDefault, nil
	}
	running = run.running
	currentAgent = run.currentAgent
	mode = run.mode
	if run.pendingInterrupt != nil {
		interrupt = s.buildInterruptEvent(run.pendingInterrupt, sessionID)
	}
	return
}

// buildInterruptEvent converts a PendingInterrupt into an InterruptEvent.
// Caller must hold s.mu.
func (s *ChatService) buildInterruptEvent(pi *PendingInterrupt, sessionID string) *InterruptEvent {
	evt := &InterruptEvent{
		InterruptID:  pi.InterruptID,
		CheckpointID: pi.CheckpointID,
		SessionID:    sessionID,
	}
	switch info := pi.Info.(type) {
	case *tools.FollowUpInfo:
		evt.Type = "followup"
		evt.Questions = info.Questions
	case *tools.ChoiceInfo:
		evt.Type = "choice"
		evt.Question = info.Question
		for _, opt := range info.Options {
			evt.Options = append(evt.Options, InterruptOption{
				Label:       opt.Label,
				Description: opt.Description,
			})
		}
	default:
		evt.Type = "followup"
		evt.Questions = []string{fmt.Sprintf("%v", pi.Info)}
	}
	return evt
}

// ---------------------------------------------------------------------------
// Runner building
// ---------------------------------------------------------------------------

// BuildRunners builds and installs the shared runner bundle using the current config.
func (s *ChatService) BuildRunners() error {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	cfg, digest, err := s.currentConfigSnapshot()
	if err != nil {
		return err
	}

	s.mu.Lock()
	prevCache := map[string]cachedMCPServerSurface(nil)
	if s.installedBundle != nil {
		prevCache = cloneSurfaceCache(s.installedBundle.CachedSurfaceMetadataByServer)
	}
	s.mu.Unlock()

	bundle, err := s.prepareRunnerBundle(ctx, cfg, digest, prevCache)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.installRunnerBundleLocked(bundle)
	return nil
}

func (s *ChatService) currentConfigSnapshot() (*config.AppConfig, string, error) {
	if s.store == nil {
		return nil, "", fmt.Errorf("config store is not available")
	}
	cfg := s.store.Get()
	digest, err := configDigest(cfg)
	if err != nil {
		return nil, "", fmt.Errorf("compute config digest: %w", err)
	}
	return cfg, digest, nil
}

func (s *ChatService) probeRunnerBundleSurface(ctx context.Context, cfg *config.AppConfig, previousCache map[string]cachedMCPServerSurface) (*runnerBundleSurface, error) {
	if s.probeBundleSurfaceFn != nil {
		return s.probeBundleSurfaceFn(ctx, cfg, previousCache)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	surface := &runnerBundleSurface{
		ActionCatalog:                 tools.NewToolCatalog(),
		CachedSurfaceMetadataByServer: make(map[string]cachedMCPServerSurface),
	}
	configIdentityDigests, err := configuredMCPServerIdentityDigests(cfg)
	if err != nil {
		return nil, err
	}

	for _, serverCfg := range cfg.MCP.Servers {
		if !serverCfg.Enabled {
			continue
		}

		handle, err := tools.ConnectMCPServerHandle(ctx, serverCfg)
		if err != nil {
			s.closeMCPHandlesLocked(surface.Handles)
			return nil, err
		}
		surface.Handles = append(surface.Handles, handle)

		identityDigest := configIdentityDigests[serverCfg.Name]
		cache, _ := matchingSurfaceCacheEntry(previousCache, serverCfg.Name, identityDigest)
		cache.ConfigIdentityDigest = identityDigest
		if handle.SupportsResources() {
			cache.SupportsResources = true
		}

		if handle.ToolMetadataReady {
			liveEntries := make([]tools.CatalogEntry, 0, len(handle.Tools))
			for _, raw := range handle.Tools {
				_, entry, err := tools.NewMCPActionAdapter(handle, raw)
				if err != nil {
					s.closeMCPHandlesLocked(surface.Handles)
					return nil, err
				}
				if err := surface.ActionCatalog.Register(entry); err != nil {
					s.closeMCPHandlesLocked(surface.Handles)
					return nil, err
				}
				liveEntries = append(liveEntries, entry)
			}
			cache.ActionEntries = cloneCatalogEntries(liveEntries)
			cache.HasToolMetadata = true
		} else if handle.State == tools.MCPServerStatePending && cache.HasToolMetadata {
			for _, entry := range cache.ActionEntries {
				if err := surface.ActionCatalog.Register(entry); err != nil {
					s.closeMCPHandlesLocked(surface.Handles)
					return nil, err
				}
			}
		}

		surface.CachedSurfaceMetadataByServer[serverCfg.Name] = cache
	}

	fingerprint, err := surfaceRelevantFingerprint(surface.ActionCatalog, surface.Handles, surface.CachedSurfaceMetadataByServer)
	if err != nil {
		s.closeMCPHandlesLocked(surface.Handles)
		return nil, err
	}
	surface.SurfaceRelevantFingerprint = fingerprint
	return surface, nil
}

func (s *ChatService) prepareRunnerBundle(ctx context.Context, cfg *config.AppConfig, digest string, previousCache map[string]cachedMCPServerSurface) (*RunnerBundle, error) {
	if s.prepareRunnerBundleFn != nil {
		return s.prepareRunnerBundleFn(ctx, cfg, digest, previousCache)
	}
	surface, err := s.probeRunnerBundleSurface(ctx, cfg, previousCache)
	if err != nil {
		return nil, err
	}
	return s.prepareRunnerBundleFromSurface(ctx, cfg, digest, surface)
}

func (s *ChatService) prepareRunnerBundleFromSurface(ctx context.Context, cfg *config.AppConfig, digest string, surface *runnerBundleSurface) (*RunnerBundle, error) {
	if s.prepareBundleFromSurfaceFn != nil {
		return s.prepareBundleFromSurfaceFn(ctx, cfg, digest, surface)
	}
	return newRuntimeBundleBuilder(s, cfg, digest, surface).Build(ctx)
}

func (s *ChatService) installRunnerBundleLocked(bundle *RunnerBundle) {
	if bundle == nil {
		return
	}
	bundle.Generation = s.nextBundleGenerationLocked()
	bundle.LastFreshnessCheckAt = s.now()
	oldBundle := s.installedBundle
	s.installedBundle = bundle
	s.retireBundleLocked(oldBundle)
}

func (s *ChatService) updateBundleFreshnessLocked(expectedGeneration uint64, expectedDigest string, checkedAt time.Time) bool {
	if s.installedBundle == nil {
		return false
	}
	if s.installedBundle.Generation != expectedGeneration || s.installedBundle.ConfigDigest != expectedDigest {
		return false
	}
	s.installedBundle.LastFreshnessCheckAt = checkedAt
	return true
}

func (s *ChatService) finishDetachedTaskLocked(task *detachedBundleTask) {
	s.clearDetachedTaskLocked(task)
	close(task.done)
}

func (s *ChatService) waitDetachedTask(ctx context.Context, task *detachedBundleTask) error {
	if task == nil {
		return nil
	}
	select {
	case <-task.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *ChatService) runColdStartTask(task *detachedBundleTask, cfg *config.AppConfig) {
	var discard *RunnerBundle
	defer func() {
		s.mu.Lock()
		s.finishDetachedTaskLocked(task)
		s.mu.Unlock()
		if discard != nil {
			s.closeRunnerBundle(discard)
		}
	}()

	bundle, err := s.prepareRunnerBundle(s.detachedTaskContext(), cfg, task.TargetConfigDigest, nil)
	if err != nil {
		task.err = err
		return
	}

	s.mu.Lock()
	_, currentDigest, digestErr := s.currentConfigSnapshot()
	if digestErr != nil {
		task.err = digestErr
		discard = bundle
		s.mu.Unlock()
		return
	}
	if s.installedBundle == nil && currentDigest == task.TargetConfigDigest {
		s.installRunnerBundleLocked(bundle)
		s.mu.Unlock()
		return
	}
	discard = bundle
	s.mu.Unlock()
}

func toolInfoSignature(entry tools.CatalogEntry) string {
	if entry.Tool == nil {
		return ""
	}
	info, err := entry.Tool.Info(context.Background())
	if err != nil || info == nil {
		return fmt.Sprintf("info-error:%v", err)
	}
	payload := struct {
		Name   string `json:"name"`
		Desc   string `json:"desc"`
		Params string `json:"params"`
	}{
		Name:   info.Name,
		Desc:   info.Desc,
		Params: fmt.Sprintf("%#v", info.ParamsOneOf),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return payload.Name + "|" + payload.Desc + "|" + payload.Params
	}
	return string(data)
}

func surfaceRelevantFingerprint(catalog *tools.ToolCatalog, handles []*tools.MCPServerHandle, cache map[string]cachedMCPServerSurface) (string, error) {
	type entrySummary struct {
		CanonicalName string   `json:"canonicalName"`
		RemoteName    string   `json:"remoteName"`
		Aliases       []string `json:"aliases,omitempty"`
		Kind          string   `json:"kind"`
		Title         string   `json:"title,omitempty"`
		Description   string   `json:"description,omitempty"`
		SearchHint    string   `json:"searchHint,omitempty"`
		ReadOnlyHint  bool     `json:"readOnlyHint"`
		ReadOnlyTrust bool     `json:"readOnlyTrust"`
		ToolInfo      string   `json:"toolInfo"`
	}
	type serverSummary struct {
		Name                 string         `json:"name"`
		State                string         `json:"state"`
		HasCachedToolMeta    bool           `json:"hasCachedToolMeta"`
		SupportsResources    bool           `json:"supportsResources"`
		DeferredActionEntrys []entrySummary `json:"deferredActionEntries,omitempty"`
	}

	if catalog == nil {
		return "", nil
	}

	entriesByServer := make(map[string][]tools.CatalogEntry)
	for _, entry := range catalog.Entries() {
		if !entry.IsMcp || entry.IsResourceTool {
			continue
		}
		entriesByServer[entry.Server] = append(entriesByServer[entry.Server], entry)
	}

	summaries := make([]serverSummary, 0, len(handles))
	for _, handle := range handles {
		if handle == nil || handle.Name == "" {
			continue
		}
		cacheEntry := cache[handle.Name]
		summary := serverSummary{
			Name:              handle.Name,
			State:             string(handle.State),
			HasCachedToolMeta: handle.ToolMetadataReady || len(handle.Tools) > 0 || cacheEntry.HasToolMetadata,
			SupportsResources: handle.SupportsResources() || cacheEntry.SupportsResources,
		}
		for _, entry := range entriesByServer[handle.Name] {
			summary.DeferredActionEntrys = append(summary.DeferredActionEntrys, entrySummary{
				CanonicalName: entry.CanonicalName,
				RemoteName:    entry.RemoteName,
				Aliases:       append([]string(nil), entry.Aliases...),
				Kind:          entry.Kind,
				Title:         entry.Title,
				Description:   entry.Description,
				SearchHint:    entry.SearchHint,
				ReadOnlyHint:  entry.ReadOnlyHint,
				ReadOnlyTrust: entry.ReadOnlyTrusted,
				ToolInfo:      toolInfoSignature(entry),
			})
		}
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})
	for i := range summaries {
		sort.Slice(summaries[i].DeferredActionEntrys, func(a, b int) bool {
			return summaries[i].DeferredActionEntrys[a].CanonicalName < summaries[i].DeferredActionEntrys[b].CanonicalName
		})
	}

	data, err := json.Marshal(summaries)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (s *ChatService) ensureBundleReadyForNewRun(ctx context.Context, sessionID string) (*RunnerBundle, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		cfg, currentDigest, err := s.currentConfigSnapshot()
		if err != nil {
			return nil, err
		}

		s.mu.Lock()
		bundle := s.installedBundle
		bundleFresh := bundle != nil && currentDigest == bundle.ConfigDigest &&
			(s.freshnessTTL <= 0 || s.now().Sub(bundle.LastFreshnessCheckAt) < s.freshnessTTL)
		if bundle != nil && currentDigest == bundle.ConfigDigest && bundleFresh {
			if err := ctx.Err(); err != nil {
				s.mu.Unlock()
				return nil, err
			}
			reserved, err := s.reserveInstalledBundleLocked(sessionID)
			s.mu.Unlock()
			if err != nil {
				return nil, err
			}
			return reserved, nil
		}

		if bundle == nil {
			task := s.coldStartTask
			if task == nil {
				task = &detachedBundleTask{
					Kind:               detachedBundleTaskColdStart,
					key:                coldStartTaskKey(currentDigest),
					TargetConfigDigest: currentDigest,
					done:               make(chan struct{}),
				}
				s.setDetachedTaskLocked(task)
				s.mu.Unlock()
				go s.runColdStartTask(task, cfg)
			} else {
				s.mu.Unlock()
			}
			reuse := currentDigest == task.TargetConfigDigest
			if err := s.waitDetachedTask(ctx, task); err != nil {
				return nil, err
			}
			if !reuse {
				continue
			}
			if task.err != nil {
				return nil, task.err
			}
			continue
		}

		task := s.freshnessTask
		if task != nil {
			s.mu.Unlock()
			reserved, done, err := s.finishFreshnessTaskForRun(ctx, task, sessionID)
			if err != nil || done {
				return reserved, err
			}
			continue
		}

		task = &detachedBundleTask{
			Kind:                 detachedBundleTaskFreshness,
			key:                  bundleKey(bundle.Generation, bundle.ConfigDigest),
			TargetConfigDigest:   currentDigest,
			ExpectedGeneration:   bundle.Generation,
			ExpectedConfigDigest: bundle.ConfigDigest,
			done:                 make(chan struct{}),
		}
		expectedFingerprint := bundle.SurfaceRelevantFingerprint
		prevCache := cloneSurfaceCache(bundle.CachedSurfaceMetadataByServer)
		s.setDetachedTaskLocked(task)
		s.mu.Unlock()

		go s.runFreshnessTask(task, cfg, expectedFingerprint, prevCache)
		reserved, done, err := s.finishFreshnessTaskForRun(ctx, task, sessionID)
		if err != nil || done {
			return reserved, err
		}
	}
}

func (s *ChatService) finishFreshnessTaskForRun(ctx context.Context, task *detachedBundleTask, sessionID string) (*RunnerBundle, bool, error) {
	if err := s.waitDetachedTask(ctx, task); err != nil {
		return nil, true, err
	}
	_, latestDigest, err := s.currentConfigSnapshot()
	if err != nil {
		return nil, true, err
	}
	if latestDigest != task.TargetConfigDigest {
		return nil, false, nil
	}
	if task.err == nil {
		return nil, false, nil
	}
	if !task.fallbackToCurrent {
		return nil, true, task.err
	}

	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, true, err
	}
	if s.installedBundle == nil {
		s.mu.Unlock()
		return nil, true, fmt.Errorf("current installed bundle is unavailable after freshness fallback")
	}
	reserved, err := s.reserveInstalledBundleLocked(sessionID)
	s.mu.Unlock()
	if err != nil {
		return nil, true, err
	}
	return reserved, true, nil
}

func (s *ChatService) runFreshnessTask(task *detachedBundleTask, cfg *config.AppConfig, expectedFingerprint string, prevCache map[string]cachedMCPServerSurface) {
	var discard *RunnerBundle
	defer func() {
		s.mu.Lock()
		s.finishDetachedTaskLocked(task)
		s.mu.Unlock()
		if discard != nil {
			s.closeRunnerBundle(discard)
		}
	}()

	surface, err := s.probeRunnerBundleSurface(s.detachedTaskContext(), cfg, prevCache)
	if err != nil {
		logger.Warn("[RUNNER] Freshness probe failed", "error", err)
		task.err = err
		task.fallbackToCurrent = true
		return
	}

	if task.TargetConfigDigest == task.ExpectedConfigDigest && surface.SurfaceRelevantFingerprint == expectedFingerprint {
		s.closeMCPHandlesLocked(surface.Handles)
		s.mu.Lock()
		_, currentDigest, digestErr := s.currentConfigSnapshot()
		if digestErr != nil {
			task.err = digestErr
			s.mu.Unlock()
			return
		}
		if s.installedBundle != nil &&
			s.installedBundle.Generation == task.ExpectedGeneration &&
			s.installedBundle.ConfigDigest == task.ExpectedConfigDigest &&
			currentDigest == task.TargetConfigDigest {
			s.updateBundleFreshnessLocked(task.ExpectedGeneration, task.ExpectedConfigDigest, s.now())
		}
		s.mu.Unlock()
		return
	}

	newBundle, err := s.prepareRunnerBundleFromSurface(s.detachedTaskContext(), cfg, task.TargetConfigDigest, surface)
	if err != nil {
		logger.Warn("[RUNNER] Freshness rebuild preparation failed", "error", err)
		task.err = err
		return
	}

	s.mu.Lock()
	_, currentDigest, digestErr := s.currentConfigSnapshot()
	if digestErr != nil {
		task.err = digestErr
		discard = newBundle
		s.mu.Unlock()
		return
	}
	if s.installedBundle != nil &&
		s.installedBundle.Generation == task.ExpectedGeneration &&
		s.installedBundle.ConfigDigest == task.ExpectedConfigDigest &&
		currentDigest == task.TargetConfigDigest {
		s.installRunnerBundleLocked(newBundle)
		s.mu.Unlock()
		return
	}
	discard = newBundle
	s.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// formatToolCall formats a tool call for display purposes.
func formatToolCall(tc schema.ToolCall) string {
	detail := fmt.Sprintf("%s(%s)", tc.Function.Name, tc.Function.Arguments)
	if len(detail) > 200 {
		detail = detail[:200] + "..."
	}
	return detail
}

// truncateResult truncates a tool result to maxLen characters.
func truncateResult(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

// buildAgentContext constructs an AgentContext from the current session and sandbox state.
func (s *ChatService) buildAgentContext() agent.AgentContext {
	ac := agent.DefaultAgentContext()

	if s.sessionService != nil {
		ac.WorkspacePath = s.sessionService.GetWorkspacePath()
	}

	// Inject SSH info from config
	cfg := s.store.Get()
	if cfg != nil {
		ac.SSHHost = cfg.SSH.Host
		ac.SSHPort = cfg.SSH.Port
		ac.SSHUser = cfg.SSH.User
	}

	if s.sandbox != nil {
		docker := s.sandbox.Docker()
		if docker != nil {
			ac.ContainerName = docker.ContainerName()
			cid := docker.ContainerID()
			if len(cid) > 12 {
				cid = cid[:12]
			}
			ac.ContainerID = cid
		}
	}

	// Set up OnToolEvent to emit sub-agent tool calls as timeline events.
	// The context carries session identity for proper event routing.
	ac.OnToolEvent = func(ctx context.Context, agentName, eventType, toolName, toolArgs, toolID, result string) {
		sessionID := SessionIDFromContext(ctx)
		content := result
		if eventType == "tool_result" {
			content = timelineToolResultContent(toolName, result)
		}
		s.emitTimelineForSession(TimelineEvent{
			ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Type:      eventType,
			Agent:     agentName,
			ToolName:  toolName,
			ToolArgs:  toolArgs,
			ToolID:    toolID,
			Content:   content,
			Timestamp: time.Now().UnixMilli(),
		}, sessionID)
		if eventType == "tool_result" {
			s.emitRuntimeWorktreeToolEvent(sessionID, toolName, toolArgs, result)
		}
	}

	return ac
}

// getWorkspacePath returns the workspace path from the session service or a default.
func (s *ChatService) getWorkspacePath() string {
	if s.sessionService != nil {
		return s.sessionService.GetWorkspacePath()
	}
	return "/workspace"
}

// drainStreamForRun iterates a MessageStream for a specific session run,
// batching stream_chunk timeline events with a 50ms window to reduce IPC frequency,
// and returns the final concatenated message.
func (s *ChatService) drainStreamForRun(stream adk.MessageStream, agentName string, run *SessionRun) (*schema.Message, error) {
	defer stream.Close()

	var chunks []*schema.Message
	var streamingContent strings.Builder

	const batchInterval = 50 * time.Millisecond
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	var pendingText strings.Builder

	flushChunks := func() {
		if pendingText.Len() == 0 {
			return
		}
		merged := pendingText.String()
		pendingText.Reset()
		s.emitTimelineForRun(TimelineEvent{
			ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Type:      "stream_chunk",
			Agent:     agentName,
			Content:   merged,
			Timestamp: time.Now().UnixMilli(),
		}, run)
	}

	done := false
	for !done {
		select {
		case <-ticker.C:
			flushChunks()
			// Update session's streaming state for mid-stream saves
			run.setStreamingState(&model.StreamingState{
				PartialContent: streamingContent.String(),
				AgentName:      agentName,
			})
		default:
			chunk, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					done = true
					break
				}
				flushChunks()
				return nil, err
			}
			chunks = append(chunks, chunk)

			if chunk.Content != "" {
				pendingText.WriteString(chunk.Content)
				streamingContent.WriteString(chunk.Content)
			}
		}
	}

	// Flush remaining
	flushChunks()

	// Clear session's streaming state
	run.setStreamingState(nil)

	// Signal stream end
	s.emitTimelineForRun(TimelineEvent{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      "stream_end",
		Agent:     agentName,
		Timestamp: time.Now().UnixMilli(),
	}, run)

	if len(chunks) == 0 {
		return nil, nil
	}

	return schema.ConcatMessages(chunks)
}
