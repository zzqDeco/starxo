package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"starxo/internal/agent"
	"starxo/internal/config"
	agentctx "starxo/internal/context"
	"starxo/internal/llm"
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

// PendingInterrupt holds the state needed to resume after an interrupt.
type PendingInterrupt struct {
	CheckpointID     string
	InterruptID      string
	BundleGeneration uint64
	RunnerKind       RunnerKind
	Info             any
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
	Generation                    uint64
	ConfigDigest                  string
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
	deferredAnnouncementState *model.DeferredAnnouncementState
	mcpInstructionsDeltaState *model.MCPInstructionsDeltaState

	// Run lifecycle
	running                      bool
	starting                     bool
	cancelFn                     context.CancelFunc
	startDone                    chan struct{}
	runDone                      chan struct{}
	pendingInterrupt             *PendingInterrupt
	streamingState               *model.StreamingState
	mode                         string // "default" or "plan"
	currentAgent                 string
	pendingStartBundleGeneration uint64
	activeBundleGeneration       uint64
	activeRunnerKind             RunnerKind
}

type SessionSnapshot struct {
	SessionData   *model.SessionData
	MessageCount  int
	HasSessionRun bool
}

func (r *SessionRun) addUserMessage(content string) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.ctxEngine.AddUserMessage(content)
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

func (r *SessionRun) addUserTurn(id, content string, timestamp int64) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.timeline.AddUserTurn(id, content, timestamp)
}

func (r *SessionRun) prepareMessages() []*schema.Message {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()
	return r.ctxEngine.PrepareMessages()
}

func (r *SessionRun) clearSessionState() {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.ctxEngine.ClearHistory()
	r.timeline.Clear()
	r.streamingState = nil
	r.discoveredTools = make(map[string]model.DiscoveredToolRecord)
	r.deferredAnnouncementState = nil
	r.mcpInstructionsDeltaState = nil
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

func (r *SessionRun) importSessionData(data *model.SessionData) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if data != nil && data.Messages != nil {
		r.ctxEngine.ImportMessages(data.Messages)
	} else {
		r.ctxEngine.ClearHistory()
	}
	if data != nil && data.Display != nil {
		r.timeline.Import(data.Display)
	} else {
		r.timeline.Clear()
	}
	r.streamingState = data.Streaming
	r.discoveredTools = make(map[string]model.DiscoveredToolRecord)
	r.deferredAnnouncementState = nil
	r.mcpInstructionsDeltaState = nil
	if data == nil {
		return
	}
	r.deferredAnnouncementState = cloneDeferredAnnouncementState(data.DeferredAnnouncementState)
	r.mcpInstructionsDeltaState = cloneMCPInstructionsDeltaState(data.MCPInstructionsDeltaState)
	for _, record := range data.DiscoveredTools {
		if record.CanonicalName == "" {
			continue
		}
		r.discoveredTools[record.CanonicalName] = record
	}
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

	return &SessionSnapshot{
		HasSessionRun: true,
		MessageCount:  r.ctxEngine.MessageCount(),
		SessionData: &model.SessionData{
			Version:                   3,
			Messages:                  r.ctxEngine.ExportMessages(),
			Display:                   r.timeline.Export(),
			Streaming:                 cloneStreamingState(r.streamingState),
			DiscoveredTools:           discovered,
			DeferredAnnouncementState: cloneDeferredAnnouncementState(r.deferredAnnouncementState),
			MCPInstructionsDeltaState: cloneMCPInstructionsDeltaState(r.mcpInstructionsDeltaState),
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

type deferredMCPProvider struct {
	chat   *ChatService
	bundle *RunnerBundle
}

func (p *deferredMCPProvider) MCPHandleSnapshot() []*tools.MCPServerHandle {
	if p.bundle == nil || len(p.bundle.MCPHandles) == 0 {
		return nil
	}
	out := make([]*tools.MCPServerHandle, len(p.bundle.MCPHandles))
	copy(out, p.bundle.MCPHandles)
	return out
}

func (p *deferredMCPProvider) LookupCatalogEntry(name string) (tools.CatalogEntry, bool) {
	if p.bundle == nil || p.bundle.MCPCatalog == nil {
		return tools.CatalogEntry{}, false
	}
	return p.bundle.MCPCatalog.LookupExact(name)
}

func (p *deferredMCPProvider) ToolPermissionContext(ctx context.Context) (tools.ToolPermissionContext, error) {
	sessionID, mode, _, err := p.sessionState(ctx)
	if err != nil {
		return tools.ToolPermissionContext{}, err
	}
	return p.permissionContext(sessionID, mode), nil
}

func (p *deferredMCPProvider) DeferredMCPState(ctx context.Context) (tools.DeferredMCPState, error) {
	sessionID, mode, discovered, err := p.sessionState(ctx)
	if err != nil {
		return tools.DeferredMCPState{}, err
	}
	if p.bundle == nil || p.bundle.MCPCatalog == nil {
		return tools.DeferredMCPState{}, nil
	}
	return tools.ComputeDeferredMCPState(p.bundle.MCPCatalog, discovered, p.permissionContext(sessionID, mode)), nil
}

func (p *deferredMCPProvider) PrepareDeferredSyntheticMessages(ctx context.Context) (*tools.DeferredSyntheticMessages, error) {
	state, err := p.DeferredMCPState(ctx)
	if err != nil {
		return nil, err
	}
	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID missing from context")
	}

	p.chat.mu.Lock()
	run, ok := p.chat.sessions[sessionID]
	p.chat.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	current := tools.NormalizeSearchableCanonicalNames(state.SearchablePoolForMode)
	prior := run.deferredAnnouncementStateSnapshot()
	msg, next := tools.BuildDeferredAnnouncementDelta(current, prior)

	needsCommit := prior == nil
	if !needsCommit && next != nil {
		needsCommit = !slices.Equal(prior.AnnouncedSearchableCanonicalNames, next.AnnouncedSearchableCanonicalNames)
	}
	if msg == nil && !needsCommit {
		return nil, nil
	}

	prepared := &tools.DeferredSyntheticMessages{}
	if msg != nil {
		prepared.Messages = []*schema.Message{msg}
	}
	if needsCommit {
		prepared.Commit = func() {
			run.setDeferredAnnouncementState(next)
		}
	}
	return prepared, nil
}

func (p *deferredMCPProvider) ToolSearchState(ctx context.Context) (tools.ToolSearchState, error) {
	state, err := p.DeferredMCPState(ctx)
	if err != nil {
		return tools.ToolSearchState{}, err
	}
	return tools.ToolSearchState{
		SearchablePool:   state.SearchablePoolForMode,
		CurrentLoaded:    state.CurrentLoadedTools,
		PendingMCPServer: state.PendingMCPServers,
	}, nil
}

func (p *deferredMCPProvider) AddDiscoveredTools(ctx context.Context, records []model.DiscoveredToolRecord) error {
	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		return fmt.Errorf("sessionID missing from context")
	}

	changed := false
	for _, record := range records {
		if record.CanonicalName == "" {
			continue
		}
		entry, ok := p.LookupCatalogEntry(record.CanonicalName)
		if !ok || !entry.ShouldDefer || entry.AlwaysLoad {
			continue
		}
		if p.chat.AddDiscoveredTool(sessionID, record) {
			changed = true
		}
	}
	if !changed {
		return nil
	}

	p.chat.mu.Lock()
	ss := p.chat.sessionService
	p.chat.mu.Unlock()
	if ss == nil {
		return nil
	}
	return ss.SaveSessionByID(sessionID)
}

func (p *deferredMCPProvider) sessionState(ctx context.Context) (string, string, map[string]model.DiscoveredToolRecord, error) {
	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		return "", "", nil, fmt.Errorf("sessionID missing from context")
	}

	p.chat.mu.Lock()
	run, ok := p.chat.sessions[sessionID]
	if !ok {
		p.chat.mu.Unlock()
		return "", "", nil, fmt.Errorf("session %s not found", sessionID)
	}
	mode := run.mode
	p.chat.mu.Unlock()

	return sessionID, mode, run.discoveredToolsSnapshot(), nil
}

func (p *deferredMCPProvider) permissionContext(sessionID, mode string) tools.ToolPermissionContext {
	servers := make(map[string]tools.MCPServerPermissionState)
	if p.bundle != nil {
		for serverName, cache := range p.bundle.CachedSurfaceMetadataByServer {
			servers[serverName] = tools.MCPServerPermissionState{
				State:                 tools.MCPServerStateFailed,
				HasCachedToolMetadata: cache.HasToolMetadata,
				SupportsResources:     cache.SupportsResources,
			}
		}
		for _, handle := range p.bundle.MCPHandles {
			if handle == nil || handle.Name == "" {
				continue
			}
			cache := p.bundle.CachedSurfaceMetadataByServer[handle.Name]
			servers[handle.Name] = tools.MCPServerPermissionState{
				State:                 handle.State,
				HasCachedToolMetadata: handle.ToolMetadataReady || len(handle.Tools) > 0 || cache.HasToolMetadata,
				SupportsResources:     handle.SupportsResources() || cache.SupportsResources,
			}
		}
	}
	return tools.ToolPermissionContext{
		SessionID: sessionID,
		Mode:      mode,
		Servers:   servers,
	}
}

func newDeferredUnknownToolHandler(provider *deferredMCPProvider) func(ctx context.Context, name, input string) (string, error) {
	return func(ctx context.Context, name, input string) (string, error) {
		state, err := provider.DeferredMCPState(ctx)
		if err != nil {
			return "", err
		}

		if name == "tool_search" {
			if len(state.SearchablePoolForMode) > 0 || len(state.PendingMCPServers) > 0 {
				return "", nil
			}
			return "tool_search is unavailable because no deferred MCP tools are currently searchable", nil
		}

		if entry, ok := provider.LookupCatalogEntry(name); ok {
			if state.IsCurrentlyLoaded(entry.CanonicalName) {
				return fmt.Sprintf("tool %s is already loaded; call it by its canonical name %s", name, entry.CanonicalName), nil
			}
			if state.IsCurrentlySearchable(entry.CanonicalName) {
				return fmt.Sprintf("tool %s is available but not currently loaded; use tool_search first", entry.CanonicalName), nil
			}
			if decision, ok := state.SearchDecisions[entry.CanonicalName]; ok && decision.Reason != "" {
				return fmt.Sprintf("tool %s is unavailable in the current mode or runtime: %s", entry.CanonicalName, decision.Reason), nil
			}
			return fmt.Sprintf("tool %s is unavailable in the current mode or runtime", entry.CanonicalName), nil
		}

		return fmt.Sprintf("unknown tool %s", name), nil
	}
}

// ChatService manages chat interactions between the frontend and the AI agent.
type ChatService struct {
	ctx context.Context

	sandbox         *sandbox.SandboxManager
	store           *config.Store
	checkpointStore compose.CheckPointStore
	now             func() time.Time
	freshnessTTL    time.Duration

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
	onAgentDone    func(sessionID string)

	mu sync.Mutex
}

// NewChatService creates a new ChatService.
func NewChatService(store *config.Store) *ChatService {
	return &ChatService{
		store:           store,
		checkpointStore: checkpoint.NewInMemoryStore(),
		sessions:        make(map[string]*SessionRun),
		now:             time.Now,
		freshnessTTL:    defaultBundleFreshnessTTL,
	}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sandbox = sbx
	s.invalidateRunners()
}

// InvalidateRunner forces runners to be rebuilt on the next message.
func (s *ChatService) InvalidateRunner() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalidateRunners()
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
		sessionID:       sessionID,
		ctxEngine:       agentctx.NewEngine(defaultSystemPrompt, defaultMaxTokens),
		timeline:        agentctx.NewTimelineCollector(),
		discoveredTools: make(map[string]model.DiscoveredToolRecord),
		mode:            "default",
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
	if mode == "plan" {
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
	if run, ok := s.sessions[sessionID]; ok {
		if run.cancelFn != nil {
			run.cancelFn()
		}
		if run.startDone != nil {
			close(run.startDone)
			run.startDone = nil
		}
		run.starting = false
		run.pendingStartBundleGeneration = 0
	}
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	s.cleanupRetiredBundlesLocked()
}

// ---------------------------------------------------------------------------
// Mode management
// ---------------------------------------------------------------------------

// SetMode switches the active session between "default" and "plan" mode.
func (s *ChatService) SetMode(mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if mode != "default" && mode != "plan" {
		return fmt.Errorf("invalid mode: %s (must be 'default' or 'plan')", mode)
	}

	run := s.activeRun()
	if run == nil {
		return fmt.Errorf("no active session")
	}

	run.mode = mode
	logger.Info("[CHAT] Mode changed", "mode", mode, "session", s.activeSessionID)
	wailsruntime.EventsEmit(s.ctx, "agent:mode_changed", ModeChangedEvent{
		Mode:      mode,
		SessionID: s.activeSessionID,
	})
	return nil
}

// GetMode returns the current agent mode for the active session.
func (s *ChatService) GetMode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	if run == nil {
		return "default"
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
	s.mu.Lock()

	if s.activeSessionID == "" {
		s.mu.Unlock()
		return fmt.Errorf("no active session")
	}

	run := s.activeRun()

	// Per-session concurrent run guard
	if run.running || run.starting {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}

	logger.Info("[CHAT] User message received",
		"length", len(userMessage),
		"preview", truncateResult(userMessage, 100),
		"session", s.activeSessionID,
	)

	// Add user message to session's context engine
	run.addUserMessage(userMessage)

	// Record user turn in session's timeline collector
	run.addUserTurn(
		fmt.Sprintf("usr-%d", time.Now().UnixNano()),
		userMessage,
		time.Now().UnixMilli(),
	)

	// Auto-escalate to plan mode for complex tasks when currently in default mode.
	// This keeps default mode flexible while enforcing strict orchestration once
	// plan mode is entered.
	if run.mode == "default" && shouldAutoPlanMode(userMessage) {
		run.mode = "plan"
		logger.Info("[CHAT] Auto-switched to plan mode",
			"session", s.activeSessionID,
			"reason", "complexity_trigger",
		)
		wailsruntime.EventsEmit(s.ctx, "agent:mode_changed", ModeChangedEvent{
			Mode:      "plan",
			SessionID: s.activeSessionID,
		})
	}

	sessionID := run.sessionID
	mode := run.mode
	baseCtx := s.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	startCtx, startCancel := context.WithCancel(baseCtx)
	run.starting = true
	run.cancelFn = startCancel
	run.startDone = make(chan struct{})
	s.mu.Unlock()

	bundle, err := s.ensureBundleReadyForNewRun(startCtx, sessionID)
	if err != nil {
		startCancel()
		s.mu.Lock()
		s.finalizeStartupLocked(sessionID)
		s.mu.Unlock()
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		logger.Error("[CHAT] Failed to prepare runner bundle", err)
		wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
			"sessionId": sessionID,
			"error":     fmt.Sprintf("Failed to build runner: %v", err),
		})
		return fmt.Errorf("failed to build runners: %w", err)
	}
	if err := startCtx.Err(); err != nil {
		startCancel()
		s.mu.Lock()
		s.finalizeStartupLocked(sessionID)
		s.mu.Unlock()
		return nil
	}

	runnerKind := runnerKindForMode(mode)
	runner := runnerForKind(bundle, runnerKind)
	if runner == nil {
		startCancel()
		s.mu.Lock()
		s.finalizeStartupLocked(sessionID)
		s.mu.Unlock()
		return fmt.Errorf("runner %s unavailable for bundle generation %d", runnerKind, bundle.Generation)
	}

	// Create a cancellable context with session identity
	runCtx, cancel := context.WithCancel(baseCtx)
	runCtx = contextWithSessionID(runCtx, sessionID)
	s.mu.Lock()
	if startCtx.Err() != nil {
		startCancel()
		s.finalizeStartupLocked(sessionID)
		s.mu.Unlock()
		cancel()
		return nil
	}
	done := make(chan struct{})
	run, err = s.publishStartupLocked(sessionID, bundle, runnerKind, cancel, done)
	s.mu.Unlock()
	if err != nil {
		startCancel()
		cancel()
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		return err
	}
	startCancel()

	// Prepare messages
	messages := run.prepareMessages()
	checkpointID := fmt.Sprintf("run-%d", time.Now().UnixNano())

	// Launch the agent run in a goroutine
	go func() {
		defer close(done)
		defer func() {
			s.mu.Lock()
			run.running = false
			run.cancelFn = nil
			run.activeBundleGeneration = 0
			run.activeRunnerKind = ""
			s.cleanupRetiredBundlesLocked()
			s.mu.Unlock()
		}()
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
					"sessionId": sessionID,
					"error":     fmt.Sprintf("Agent panic: %v", r),
				})
				wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
					"sessionId": sessionID,
				})
			}
		}()

		logger.Info("[CHAT] Agent run started",
			"message_count", len(messages),
			"mode", mode,
			"session", sessionID,
		)
		startTime := time.Now()

		events := runner.Run(runCtx, messages, adk.WithCheckPointID(checkpointID))

		lastContent, transferCount, interrupted := s.processEventsForRun(events, checkpointID, run)

		if interrupted {
			return // Don't emit done — waiting for user response
		}

		// Add final assistant response to session's context engine
		if lastContent != "" {
			run.addAssistantMessage(lastContent)
		}

		wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
			"sessionId": sessionID,
		})

		logger.Info("[CHAT] Agent run completed",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
			"has_response", lastContent != "",
			"session", sessionID,
		)

		// Notify listeners (e.g. session auto-save) with sessionID
		s.mu.Lock()
		doneFn := s.onAgentDone
		s.mu.Unlock()
		if doneFn != nil {
			doneFn(sessionID)
		}
	}()

	return nil
}

// ---------------------------------------------------------------------------
// Timeline emission
// ---------------------------------------------------------------------------

// emitTimelineForRun emits a timeline event to the frontend AND records it in the
// session's timeline collector for backend persistence.
func (s *ChatService) emitTimelineForRun(evt TimelineEvent, run *SessionRun) {
	evt.SessionID = run.sessionID
	wailsruntime.EventsEmit(s.ctx, "agent:timeline", evt)
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
	wailsruntime.EventsEmit(s.ctx, "agent:timeline", evt)
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if ok {
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
func (s *ChatService) processEventsForRun(events *adk.AsyncIterator[*adk.AgentEvent], checkpointID string, run *SessionRun) (string, int, bool) {
	var allContents []string
	var transferCount int
	lastContentByAgent := make(map[string]string) // dedup

	// Track pending tool_call_ids to detect orphans (tool calls without results)
	pendingToolCalls := make(map[string]bool)

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
				Content:   fmt.Sprintf("Error: %v", event.Err),
				Timestamp: time.Now().UnixMilli(),
			}, run)
			continue
		}

		// Handle agent actions (tool calls, transfers, interrupts)
		if event.Action != nil {
			// Interrupt detection
			if event.Action.Interrupted != nil && len(event.Action.Interrupted.InterruptContexts) > 0 {
				interruptCtx := event.Action.Interrupted.InterruptContexts[0]
				s.handleInterruptForRun(interruptCtx, checkpointID, run)
				return strings.Join(allContents, "\n\n"), transferCount, true
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
				run.addMessage(&schema.Message{
					Role:      schema.Assistant,
					Content:   msg.Content,
					ToolCalls: msg.ToolCalls,
				})
				// Track pending tool call IDs
				for _, tc := range msg.ToolCalls {
					pendingToolCalls[tc.ID] = true
				}
				continue // Don't fall through to allContents — tool call content is already stored
			}

			// Emit tool result events
			if msg.Role == schema.Tool && msg.ToolCallID != "" {
				s.emitTimelineForRun(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "tool_result",
					Agent:     event.AgentName,
					Content:   truncateResult(msg.Content, 1000),
					ToolID:    msg.ToolCallID,
					Timestamp: time.Now().UnixMilli(),
				}, run)

				// Store tool result in session's context history
				run.addToolResult(msg.ToolCallID, msg.Content)
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
	if len(pendingToolCalls) > 0 {
		for toolCallID := range pendingToolCalls {
			logger.Warn("[CHAT] Injecting synthetic tool result for orphaned tool_call",
				"tool_call_id", toolCallID, "session", sessionID)
			run.addToolResult(toolCallID, "Error: tool execution failed or was interrupted")
		}
	}

	return strings.Join(allContents, "\n\n"), transferCount, false
}

// ---------------------------------------------------------------------------
// Interrupt handling
// ---------------------------------------------------------------------------

// handleInterruptForRun processes an interrupt context for a specific session run.
func (s *ChatService) handleInterruptForRun(interruptCtx *adk.InterruptCtx, checkpointID string, run *SessionRun) {
	s.mu.Lock()
	run.pendingInterrupt = &PendingInterrupt{
		CheckpointID:     checkpointID,
		InterruptID:      interruptCtx.ID,
		BundleGeneration: run.activeBundleGeneration,
		RunnerKind:       run.activeRunnerKind,
		Info:             interruptCtx.Info,
	}
	run.activeBundleGeneration = 0
	run.activeRunnerKind = ""
	s.mu.Unlock()

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

	wailsruntime.EventsEmit(s.ctx, "agent:interrupt", evt)
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
	if run.running {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}
	_, runner, pending, err := s.resolvePendingRunnerLocked(run)
	if err != nil {
		if run != nil {
			run.pendingInterrupt = nil
			s.cleanupRetiredBundlesLocked()
		}
		s.mu.Unlock()
		return err
	}
	run.pendingInterrupt = nil
	run.activeBundleGeneration = pending.BundleGeneration
	run.activeRunnerKind = pending.RunnerKind
	sessionID := run.sessionID
	s.mu.Unlock()

	// Build resume data with user's answer
	resumeData := &tools.FollowUpInfo{
		UserAnswer: answer,
	}
	if info, ok := pending.Info.(*tools.FollowUpInfo); ok {
		resumeData.Questions = info.Questions
	}

	runCtx, cancel := context.WithCancel(s.ctx)
	runCtx = contextWithSessionID(runCtx, sessionID)
	s.mu.Lock()
	run.cancelFn = cancel
	run.running = true
	done := make(chan struct{})
	run.runDone = done
	s.mu.Unlock()

	go func() {
		defer close(done)
		defer func() {
			s.mu.Lock()
			run.running = false
			run.cancelFn = nil
			run.activeBundleGeneration = 0
			run.activeRunnerKind = ""
			s.cleanupRetiredBundlesLocked()
			s.mu.Unlock()
		}()
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
					"sessionId": sessionID,
					"error":     fmt.Sprintf("Resume panic: %v", r),
				})
				wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
					"sessionId": sessionID,
				})
			}
		}()

		logger.Info("[CHAT] Resuming after follow-up", "answer_length", len(answer), "session", sessionID)
		startTime := time.Now()

		events, err := runner.ResumeWithParams(runCtx, pending.CheckpointID, &adk.ResumeParams{
			Targets: map[string]any{
				pending.InterruptID: resumeData,
			},
		})
		if err != nil {
			wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
				"sessionId": sessionID,
				"error":     fmt.Sprintf("Resume failed: %v", err),
			})
			wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
				"sessionId": sessionID,
			})
			return
		}

		lastContent, transferCount, interrupted := s.processEventsForRun(events, pending.CheckpointID, run)

		if interrupted {
			return
		}

		if lastContent != "" {
			run.addAssistantMessage(lastContent)
		}

		wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
			"sessionId": sessionID,
		})
		logger.Info("[CHAT] Resume completed",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
			"session", sessionID,
		)

		s.mu.Lock()
		doneFn := s.onAgentDone
		s.mu.Unlock()
		if doneFn != nil {
			doneFn(sessionID)
		}
	}()

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
	if run.running {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}
	_, runner, pending, err := s.resolvePendingRunnerLocked(run)
	if err != nil {
		if run != nil {
			run.pendingInterrupt = nil
			s.cleanupRetiredBundlesLocked()
		}
		s.mu.Unlock()
		return err
	}
	run.pendingInterrupt = nil
	run.activeBundleGeneration = pending.BundleGeneration
	run.activeRunnerKind = pending.RunnerKind
	sessionID := run.sessionID
	s.mu.Unlock()

	// Build resume data with user's selection
	resumeData := &tools.ChoiceInfo{
		Selected: selectedIndex,
	}
	if info, ok := pending.Info.(*tools.ChoiceInfo); ok {
		resumeData.Question = info.Question
		resumeData.Options = info.Options
	}

	runCtx, cancel := context.WithCancel(s.ctx)
	runCtx = contextWithSessionID(runCtx, sessionID)
	s.mu.Lock()
	run.cancelFn = cancel
	run.running = true
	done := make(chan struct{})
	run.runDone = done
	s.mu.Unlock()

	go func() {
		defer close(done)
		defer func() {
			s.mu.Lock()
			run.running = false
			run.cancelFn = nil
			run.activeBundleGeneration = 0
			run.activeRunnerKind = ""
			s.cleanupRetiredBundlesLocked()
			s.mu.Unlock()
		}()
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
					"sessionId": sessionID,
					"error":     fmt.Sprintf("Resume panic: %v", r),
				})
				wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
					"sessionId": sessionID,
				})
			}
		}()

		logger.Info("[CHAT] Resuming after choice", "selected", selectedIndex, "session", sessionID)
		startTime := time.Now()

		events, err := runner.ResumeWithParams(runCtx, pending.CheckpointID, &adk.ResumeParams{
			Targets: map[string]any{
				pending.InterruptID: resumeData,
			},
		})
		if err != nil {
			wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
				"sessionId": sessionID,
				"error":     fmt.Sprintf("Resume failed: %v", err),
			})
			wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
				"sessionId": sessionID,
			})
			return
		}

		lastContent, transferCount, interrupted := s.processEventsForRun(events, pending.CheckpointID, run)

		if interrupted {
			return
		}

		if lastContent != "" {
			run.addAssistantMessage(lastContent)
		}

		wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
			"sessionId": sessionID,
		})
		logger.Info("[CHAT] Resume completed",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
			"session", sessionID,
		)

		s.mu.Lock()
		doneFn := s.onAgentDone
		s.mu.Unlock()
		if doneFn != nil {
			doneFn(sessionID)
		}
	}()

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
		s.mu.Unlock()
		return nil
	}
	if run.cancelFn != nil {
		run.cancelFn()
	}
	run.pendingInterrupt = nil
	s.cleanupRetiredBundlesLocked()
	done := run.runDone
	s.mu.Unlock()

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
		s.mu.Unlock()
		return
	}
	if run.cancelFn != nil {
		run.cancelFn()
	}
	run.pendingInterrupt = nil
	s.cleanupRetiredBundlesLocked()
	done := run.runDone
	s.mu.Unlock()

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
	defer s.mu.Unlock()

	run := s.activeRun()
	if run == nil {
		return nil
	}

	run.clearSessionState()
	run.pendingInterrupt = nil
	s.invalidateRunners()
	s.cleanupRetiredBundlesLocked()
	tools.ClearTodos()
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
func (s *ChatService) ExportSessionSnapshot(sessionID string) (*SessionSnapshot, error) {
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		return &SessionSnapshot{
			SessionData: &model.SessionData{
				Version: 3,
			},
		}, nil
	}
	return run.snapshot(), nil
}

func (s *ChatService) RestoreSessionData(sessionID string, data *model.SessionData) {
	s.mu.Lock()
	run := s.getOrCreateRun(sessionID)
	s.mu.Unlock()
	run.importSessionData(data)
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
		return false, "", "default", nil
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
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if surface == nil {
		return nil, fmt.Errorf("runner bundle surface is nil")
	}
	if s.sandbox == nil || !s.sandbox.IsConnected() {
		s.closeMCPHandlesLocked(surface.Handles)
		return nil, fmt.Errorf("sandbox is not connected")
	}

	op := s.sandbox.Operator()
	if op == nil {
		s.closeMCPHandlesLocked(surface.Handles)
		return nil, fmt.Errorf("sandbox operator is not available")
	}

	mdl, err := llm.NewChatModel(ctx, cfg.LLM)
	if err != nil {
		s.closeMCPHandlesLocked(surface.Handles)
		logger.Error("[RUNNER] Failed to create chat model", err)
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}
	logger.RunnerEvent("chat_model_created", "type", cfg.LLM.Type, "model", cfg.LLM.Model)

	bundle := &RunnerBundle{
		ConfigDigest:                  digest,
		MCPHandles:                    surface.Handles,
		SurfaceRelevantFingerprint:    surface.SurfaceRelevantFingerprint,
		CachedSurfaceMetadataByServer: cloneSurfaceCache(surface.CachedSurfaceMetadataByServer),
	}
	provider := &deferredMCPProvider{chat: s, bundle: bundle}

	topLevelCatalog := tools.NewToolCatalog()
	for _, entry := range surface.ActionCatalog.Entries() {
		wrapped := entry
		wrapped.Tool = tools.WrapMCPToolWithPermissionCheck(wrapped, provider)
		if err := topLevelCatalog.Register(wrapped); err != nil {
			s.closeMCPHandlesLocked(surface.Handles)
			return nil, fmt.Errorf("failed to register MCP action tool %s: %w", wrapped.CanonicalName, err)
		}
	}

	resourceEntries, err := tools.NewMCPResourceCatalogEntries(provider)
	if err != nil {
		s.closeMCPHandlesLocked(surface.Handles)
		return nil, fmt.Errorf("failed to build MCP resource tools: %w", err)
	}
	for _, entry := range resourceEntries {
		wrapped := entry
		wrapped.Tool = tools.WrapMCPToolWithPermissionCheck(wrapped, provider)
		if err := topLevelCatalog.Register(wrapped); err != nil {
			s.closeMCPHandlesLocked(surface.Handles)
			return nil, fmt.Errorf("failed to register MCP resource tool %s: %w", wrapped.CanonicalName, err)
		}
	}
	bundle.MCPCatalog = topLevelCatalog

	toolSearchTool, err := tools.NewToolSearchTool(provider)
	if err != nil {
		s.closeMCPHandlesLocked(surface.Handles)
		return nil, fmt.Errorf("failed to build tool_search: %w", err)
	}

	extraTools := make([]einotool.BaseTool, 0, len(topLevelCatalog.CanonicalNames())+1)
	extraTools = append(extraTools, toolSearchTool)
	extraTools = append(extraTools, topLevelCatalog.Tools()...)

	ac := s.buildAgentContext()
	deferredHandler := tools.NewDynamicMCPSurfaceMiddleware(provider)
	unknownToolsHandler := newDeferredUnknownToolHandler(provider)

	deepAgentDefault, err := agent.BuildDeepAgentForMode(
		ctx, mdl, op, extraTools, ac, agent.DeepAgentModeDefault,
		[]adk.ChatModelAgentMiddleware{deferredHandler},
		unknownToolsHandler,
	)
	if err != nil {
		s.closeMCPHandlesLocked(surface.Handles)
		logger.Error("[RUNNER] Failed to build default deep agent", err)
		return nil, fmt.Errorf("failed to build default deep agent: %w", err)
	}

	deepAgentPlan, err := agent.BuildDeepAgentForMode(
		ctx, mdl, op, extraTools, ac, agent.DeepAgentModePlan,
		[]adk.ChatModelAgentMiddleware{deferredHandler},
		unknownToolsHandler,
	)
	if err != nil {
		s.closeMCPHandlesLocked(surface.Handles)
		logger.Error("[RUNNER] Failed to build plan deep agent", err)
		return nil, fmt.Errorf("failed to build plan deep agent: %w", err)
	}

	bundle.DefaultRunner = agent.BuildDefaultRunner(ctx, deepAgentDefault, s.checkpointStore)
	bundle.PlanRunner, err = agent.BuildPlanRunner(ctx, mdl, deepAgentPlan, ac, s.checkpointStore)
	if err != nil {
		s.closeMCPHandlesLocked(surface.Handles)
		logger.Error("[RUNNER] Failed to build plan runner", err)
		return nil, fmt.Errorf("failed to build plan runner: %w", err)
	}
	bundle.LastFreshnessCheckAt = s.now()

	for _, handle := range bundle.MCPHandles {
		if handle == nil || handle.LastError == nil {
			continue
		}
		wailsruntime.EventsEmit(s.ctx, "agent:error",
			fmt.Sprintf("MCP server %s unavailable (%s): %v", handle.Name, handle.State, handle.LastError))
	}
	logger.RunnerEvent("runners_built", "extra_tools", len(extraTools))
	return bundle, nil
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
			if err := s.waitDetachedTask(ctx, task); err != nil {
				return nil, err
			}
			_, latestDigest, err := s.currentConfigSnapshot()
			if err != nil {
				return nil, err
			}
			if latestDigest != task.TargetConfigDigest {
				continue
			}
			if task.err != nil {
				if task.fallbackToCurrent {
					s.mu.Lock()
					if err := ctx.Err(); err != nil {
						s.mu.Unlock()
						return nil, err
					}
					if s.installedBundle == nil {
						s.mu.Unlock()
						return nil, fmt.Errorf("current installed bundle is unavailable after freshness fallback")
					}
					reserved, err := s.reserveInstalledBundleLocked(sessionID)
					s.mu.Unlock()
					if err != nil {
						return nil, err
					}
					return reserved, nil
				}
				return nil, task.err
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
	}
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

// shouldAutoPlanMode applies a deterministic heuristic to decide whether a
// user request is complex enough to auto-enter plan mode.
func shouldAutoPlanMode(userMessage string) bool {
	msg := strings.ToLower(strings.TrimSpace(userMessage))
	if msg == "" {
		return false
	}

	// Explicit intent to plan.
	explicitPlanSignals := []string{
		"plan mode", "planning mode", "plan-mode",
		"计划模式", "规划模式", "进入计划", "进入规划",
	}
	for _, k := range explicitPlanSignals {
		if strings.Contains(msg, k) {
			return true
		}
	}

	stepSignals := []string{
		"and then", "then ", "after that", "step by step",
		"先", "然后", "再", "并且", "同时", "步骤",
	}
	workSignals := []string{
		"write", "edit", "refactor", "implement", "fix", "debug",
		"run", "test", "verify", "validate", "build",
		"写", "改", "重构", "实现", "修复", "调试",
		"运行", "测试", "验证", "构建", "编译",
	}

	stepCount := 0
	for _, k := range stepSignals {
		if strings.Contains(msg, k) {
			stepCount++
		}
	}

	workCount := 0
	for _, k := range workSignals {
		if strings.Contains(msg, k) {
			workCount++
		}
	}

	// Complex multi-action intent: contains sequencing and multiple work signals.
	return stepCount > 0 && workCount >= 2
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
		s.emitTimelineForSession(TimelineEvent{
			ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Type:      eventType,
			Agent:     agentName,
			ToolName:  toolName,
			ToolArgs:  toolArgs,
			ToolID:    toolID,
			Content:   result,
			Timestamp: time.Now().UnixMilli(),
		}, sessionID)
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
