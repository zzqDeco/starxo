package service

import (
	"context"
	"fmt"
	"strings"

	"starxo/internal/model"
	"starxo/internal/tools"
)

type runtimeModeOverrideCtxKey struct{}

func contextWithRuntimeModeOverride(ctx context.Context, mode string) context.Context {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return ctx
	}
	return context.WithValue(ctx, runtimeModeOverrideCtxKey{}, mode)
}

func runtimeModeOverrideFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	mode, ok := ctx.Value(runtimeModeOverrideCtxKey{}).(string)
	mode = strings.TrimSpace(mode)
	return mode, ok && mode != ""
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
	sessionID, mode, discovered, err := p.sessionState(ctx)
	if err != nil {
		return nil, err
	}
	permCtx := p.permissionContext(sessionID, mode)
	state := tools.DeferredMCPState{}
	if p.bundle != nil && p.bundle.MCPCatalog != nil {
		state = tools.ComputeDeferredMCPState(p.bundle.MCPCatalog, discovered, permCtx)
	}

	p.chat.mu.Lock()
	run, ok := p.chat.sessions[sessionID]
	p.chat.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	currentDigest := ""
	configSnapshotError := ""
	if _, digest, err := p.chat.currentConfigSnapshot(); err != nil {
		configSnapshotError = err.Error()
	} else {
		currentDigest = digest
	}

	bundleConfigDigest := ""
	var bundleGeneration uint64
	if p.bundle != nil {
		bundleConfigDigest = p.bundle.ConfigDigest
		bundleGeneration = p.bundle.Generation
	}

	computation := buildDeferredSurfaceComputation(deferredSurfaceDebugInput{
		CurrentConfigDigest: currentDigest,
		BundleConfigDigest:  bundleConfigDigest,
		BundleGeneration:    bundleGeneration,
		State:               state,
		PermissionContext:   permCtx,
		AnnouncementState:   run.deferredAnnouncementStateSnapshot(),
		InstructionsState:   run.mcpInstructionsDeltaStateSnapshot(),
		ConfigSnapshotError: configSnapshotError,
	})
	logDeferredSurfaceComputed(sessionID, mode, computation.Debug)

	if computation.AnnouncementMessage == nil &&
		computation.InstructionsMessage == nil &&
		!computation.UpdateAnnouncement &&
		!computation.UpdateInstructions {
		return nil, nil
	}

	prepared := &tools.DeferredSyntheticMessages{}
	if computation.AnnouncementMessage != nil {
		prepared.Messages = append(prepared.Messages, computation.AnnouncementMessage)
	}
	if computation.InstructionsMessage != nil {
		prepared.Messages = append(prepared.Messages, computation.InstructionsMessage)
	}
	if computation.UpdateAnnouncement || computation.UpdateInstructions {
		prepared.Commit = func() {
			run.applySyntheticDeltaStates(
				computation.AnnouncementNext,
				computation.UpdateAnnouncement,
				computation.InstructionsNext,
				computation.UpdateInstructions,
			)
			logDeferredSurfaceCommitted(
				sessionID,
				mode,
				computation.AnnouncementNext,
				computation.UpdateAnnouncement,
				computation.InstructionsNext,
				computation.UpdateInstructions,
			)
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
	if override, ok := runtimeModeOverrideFromContext(ctx); ok {
		mode = override
	}

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

		if name == tools.ToolSearchName {
			return "", nil
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
