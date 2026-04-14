package service

import (
	"errors"
	"sort"

	"github.com/cloudwego/eino/schema"

	"starxo/internal/logger"
	"starxo/internal/model"
	"starxo/internal/tools"
)

const (
	DeferredSurfaceDebugAPIDisabledMessage = "deferred surface debug API is disabled"
	deferredSurfaceSessionNotFoundMessage  = "session not found"
)

type ChatRuntimeOptions struct {
	DeferredSurfaceDebugAPIEnabled  bool
	DevDeferredBuiltinSampleEnabled bool
}

type DeferredAnnouncementPreview struct {
	Mode     string
	Added    []string
	Removed  []string
	WillEmit bool
}

type DeferredInstructionsSummary struct {
	SearchableServers  []string
	PendingServers     []string
	UnavailableServers []string
	Fingerprint        string
	WillEmit           bool
}

type DeferredSurfaceDebug struct {
	CurrentConfigDigest                   string
	BundleConfigDigest                    string
	BundleGeneration                      uint64
	SearchablePoolCanonicalNames          []string
	LoadablePoolCanonicalNames            []string
	EffectiveDiscoveredCanonicalNames     []string
	CurrentLoadedCanonicalNames           []string
	ToolSearchCurrentLoadedCanonicalNames []string
	PendingMCPServers                     []string
	ToolSearchVisible                     bool
	AnnouncementState                     model.DeferredAnnouncementState
	AnnouncementPreview                   DeferredAnnouncementPreview
	InstructionsState                     model.MCPInstructionsDeltaState
	InstructionsSummary                   DeferredInstructionsSummary
	ConfigSnapshotError                   string
	BuildWarnings                         []string
}

type deferredSurfaceDebugInput struct {
	CurrentConfigDigest string
	BundleConfigDigest  string
	BundleGeneration    uint64
	State               tools.DeferredMCPState
	PermissionContext   tools.ToolPermissionContext
	AnnouncementState   *model.DeferredAnnouncementState
	InstructionsState   *model.MCPInstructionsDeltaState
	ConfigSnapshotError string
	BuildWarnings       []string
}

type deferredSurfaceComputation struct {
	Debug               DeferredSurfaceDebug
	AnnouncementMessage *schema.Message
	AnnouncementNext    *model.DeferredAnnouncementState
	UpdateAnnouncement  bool
	InstructionsMessage *schema.Message
	InstructionsNext    *model.MCPInstructionsDeltaState
	UpdateInstructions  bool
}

type deferredSurfaceBundleSnapshot struct {
	ConfigDigest string
	Generation   uint64
	Catalog      *tools.ToolCatalog
	Handles      []*tools.MCPServerHandle
	Cache        map[string]cachedMCPServerSurface
}

type deferredSurfaceRunSnapshot struct {
	Mode              string
	Discovered        map[string]model.DiscoveredToolRecord
	AnnouncementState *model.DeferredAnnouncementState
	InstructionsState *model.MCPInstructionsDeltaState
}

func RuntimeOptionsFromEnv(lookup func(string) string) ChatRuntimeOptions {
	if lookup == nil {
		return ChatRuntimeOptions{}
	}
	return ChatRuntimeOptions{
		DeferredSurfaceDebugAPIEnabled:  lookup("STARXO_ENABLE_DEFERRED_SURFACE_DEBUG_API") == "1",
		DevDeferredBuiltinSampleEnabled: lookup("STARXO_ENABLE_DEV_DEFERRED_BUILTIN_SAMPLE") == "1",
	}
}

func (s *ChatService) runtimeOptionsSnapshot() ChatRuntimeOptions {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runtimeOptions
}

func (r *SessionRun) deferredSurfaceDebugSnapshot() deferredSurfaceRunSnapshot {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()

	discovered := make(map[string]model.DiscoveredToolRecord, len(r.discoveredTools))
	for k, v := range r.discoveredTools {
		discovered[k] = v
	}

	return deferredSurfaceRunSnapshot{
		Mode:              r.mode,
		Discovered:        discovered,
		AnnouncementState: cloneDeferredAnnouncementState(r.deferredAnnouncementState),
		InstructionsState: cloneMCPInstructionsDeltaState(r.mcpInstructionsDeltaState),
	}
}

func (s *ChatService) captureBundleSnapshot() deferredSurfaceBundleSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.installedBundle == nil {
		return deferredSurfaceBundleSnapshot{}
	}
	out := deferredSurfaceBundleSnapshot{
		ConfigDigest: s.installedBundle.ConfigDigest,
		Generation:   s.installedBundle.Generation,
		Catalog:      s.installedBundle.MCPCatalog,
		Cache:        cloneSurfaceCache(s.installedBundle.CachedSurfaceMetadataByServer),
	}
	if len(s.installedBundle.MCPHandles) > 0 {
		out.Handles = append([]*tools.MCPServerHandle(nil), s.installedBundle.MCPHandles...)
	}
	return out
}

func (s *ChatService) buildDeferredSurfaceDebug(sessionID string, sessionMissingAsWarning bool) (*DeferredSurfaceDebug, error) {
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		if !sessionMissingAsWarning {
			return nil, errors.New(deferredSurfaceSessionNotFoundMessage)
		}
		computation := buildDeferredSurfaceComputation(deferredSurfaceDebugInput{
			BuildWarnings: []string{deferredSurfaceSessionNotFoundMessage},
		})
		debug := computation.Debug
		return &debug, nil
	}

	runSnapshot := run.deferredSurfaceDebugSnapshot()
	bundleSnapshot := s.captureBundleSnapshot()

	currentDigest := ""
	configSnapshotError := ""
	if _, digest, err := s.currentConfigSnapshot(); err != nil {
		configSnapshotError = err.Error()
	} else {
		currentDigest = digest
	}

	permCtx := tools.ToolPermissionContext{
		SessionID: sessionID,
		Mode:      runSnapshot.Mode,
		Servers:   deferredSurfacePermissionServers(bundleSnapshot),
	}
	state := tools.DeferredMCPState{}
	if bundleSnapshot.Catalog != nil {
		state = tools.ComputeDeferredMCPState(bundleSnapshot.Catalog, runSnapshot.Discovered, permCtx)
	}

	computation := buildDeferredSurfaceComputation(deferredSurfaceDebugInput{
		CurrentConfigDigest: currentDigest,
		BundleConfigDigest:  bundleSnapshot.ConfigDigest,
		BundleGeneration:    bundleSnapshot.Generation,
		State:               state,
		PermissionContext:   permCtx,
		AnnouncementState:   runSnapshot.AnnouncementState,
		InstructionsState:   runSnapshot.InstructionsState,
		ConfigSnapshotError: configSnapshotError,
	})
	debug := computation.Debug
	return &debug, nil
}

func (s *ChatService) GetDeferredSurfaceDebug(sessionID string) (*DeferredSurfaceDebug, error) {
	if !s.runtimeOptionsSnapshot().DeferredSurfaceDebugAPIEnabled {
		return nil, errors.New(DeferredSurfaceDebugAPIDisabledMessage)
	}
	return s.buildDeferredSurfaceDebug(sessionID, false)
}

func deferredSurfacePermissionServers(bundle deferredSurfaceBundleSnapshot) map[string]tools.MCPServerPermissionState {
	servers := make(map[string]tools.MCPServerPermissionState)
	for serverName, cache := range bundle.Cache {
		servers[serverName] = tools.MCPServerPermissionState{
			State:                 tools.MCPServerStateFailed,
			HasCachedToolMetadata: cache.HasToolMetadata,
			SupportsResources:     cache.SupportsResources,
		}
	}
	for _, handle := range bundle.Handles {
		if handle == nil || handle.Name == "" {
			continue
		}
		cache := bundle.Cache[handle.Name]
		servers[handle.Name] = tools.MCPServerPermissionState{
			State:                 handle.State,
			HasCachedToolMetadata: handle.ToolMetadataReady || len(handle.Tools) > 0 || cache.HasToolMetadata,
			SupportsResources:     handle.SupportsResources() || cache.SupportsResources,
		}
	}
	return servers
}

func buildDeferredSurfaceComputation(input deferredSurfaceDebugInput) deferredSurfaceComputation {
	searchablePool := normalizeEntryCanonicalNames(input.State.SearchablePoolForMode)
	loadablePool := normalizeEntryCanonicalNames(input.State.LoadablePoolForMode)
	effectiveDiscovered := normalizeEntryCanonicalNames(input.State.EffectiveDiscovered)
	currentLoaded := normalizeEntryCanonicalNames(input.State.CurrentLoadedTools)
	pendingServers := normalizeStrings(input.State.PendingMCPServers)
	announcementState := normalizedAnnouncementStateValue(input.AnnouncementState)
	instructionsState := normalizedInstructionsStateValue(input.InstructionsState)
	buildWarnings := normalizeWarnings(input.BuildWarnings)

	previousAnnouncement := cloneStrings(announcementState.AnnouncedSearchableCanonicalNames)
	added, removed := diffSortedDebugStrings(previousAnnouncement, searchablePool)
	announcementMode := ""
	switch {
	case input.AnnouncementState == nil:
		announcementMode = "bootstrap"
	case len(added) > 0 || len(removed) > 0:
		announcementMode = "delta"
	}
	announcementMessage, announcementNext := tools.BuildDeferredAnnouncementDelta(searchablePool, input.AnnouncementState)
	updateAnnouncement := input.AnnouncementState == nil || !equalStringSlices(previousAnnouncement, announcementNext.AnnouncedSearchableCanonicalNames)

	summary := tools.NormalizeMCPInstructionsSummary(input.State, input.PermissionContext)
	instructionsMessage, instructionsNext := tools.BuildMCPInstructionsDeltaMessage(summary, input.InstructionsState)
	updateInstructions := input.InstructionsState == nil || !mcpInstructionsStateMatches(input.InstructionsState, instructionsNext)

	return deferredSurfaceComputation{
		Debug: DeferredSurfaceDebug{
			CurrentConfigDigest:                   input.CurrentConfigDigest,
			BundleConfigDigest:                    input.BundleConfigDigest,
			BundleGeneration:                      input.BundleGeneration,
			SearchablePoolCanonicalNames:          searchablePool,
			LoadablePoolCanonicalNames:            loadablePool,
			EffectiveDiscoveredCanonicalNames:     effectiveDiscovered,
			CurrentLoadedCanonicalNames:           currentLoaded,
			ToolSearchCurrentLoadedCanonicalNames: effectiveDiscovered,
			PendingMCPServers:                     pendingServers,
			ToolSearchVisible:                     tools.ToolSearchVisible(input.State),
			AnnouncementState:                     announcementState,
			AnnouncementPreview: DeferredAnnouncementPreview{
				Mode:     announcementMode,
				Added:    normalizeStrings(added),
				Removed:  normalizeStrings(removed),
				WillEmit: announcementMessage != nil,
			},
			InstructionsState: instructionsState,
			InstructionsSummary: DeferredInstructionsSummary{
				SearchableServers:  normalizeStrings(summary.SearchableServers),
				PendingServers:     normalizeStrings(summary.PendingServers),
				UnavailableServers: normalizeStrings(summary.UnavailableServers),
				Fingerprint:        summary.Fingerprint,
				WillEmit:           instructionsMessage != nil,
			},
			ConfigSnapshotError: input.ConfigSnapshotError,
			BuildWarnings:       buildWarnings,
		},
		AnnouncementMessage: announcementMessage,
		AnnouncementNext:    announcementNext,
		UpdateAnnouncement:  updateAnnouncement,
		InstructionsMessage: instructionsMessage,
		InstructionsNext:    instructionsNext,
		UpdateInstructions:  updateInstructions,
	}
}

func normalizedAnnouncementStateValue(in *model.DeferredAnnouncementState) model.DeferredAnnouncementState {
	if in == nil {
		return model.DeferredAnnouncementState{
			AnnouncedSearchableCanonicalNames: []string{},
		}
	}
	return model.DeferredAnnouncementState{
		AnnouncedSearchableCanonicalNames: normalizeStrings(in.AnnouncedSearchableCanonicalNames),
	}
}

func normalizedInstructionsStateValue(in *model.MCPInstructionsDeltaState) model.MCPInstructionsDeltaState {
	if in == nil {
		return model.MCPInstructionsDeltaState{
			LastAnnouncedSearchableServers:  []string{},
			LastAnnouncedPendingServers:     []string{},
			LastAnnouncedUnavailableServers: []string{},
			LastInstructionsFingerprint:     tools.ComputeMCPInstructionsFingerprint([]string{}, []string{}, []string{}),
		}
	}
	return model.MCPInstructionsDeltaState{
		LastAnnouncedSearchableServers:  normalizeStrings(in.LastAnnouncedSearchableServers),
		LastAnnouncedPendingServers:     normalizeStrings(in.LastAnnouncedPendingServers),
		LastAnnouncedUnavailableServers: normalizeStrings(in.LastAnnouncedUnavailableServers),
		LastInstructionsFingerprint:     in.LastInstructionsFingerprint,
	}
}

func normalizeEntryCanonicalNames(entries []tools.CatalogEntry) []string {
	return tools.NormalizeSearchableCanonicalNames(entries)
}

func normalizeStrings(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := append([]string(nil), in...)
	sort.Strings(out)
	deduped := out[:0]
	for _, item := range out {
		if item == "" {
			continue
		}
		if len(deduped) > 0 && deduped[len(deduped)-1] == item {
			continue
		}
		deduped = append(deduped, item)
	}
	if len(deduped) == 0 {
		return []string{}
	}
	return append([]string(nil), deduped...)
}

func normalizeWarnings(in []string) []string {
	return normalizeStrings(in)
}

func diffSortedDebugStrings(previous, current []string) (added, removed []string) {
	i, j := 0, 0
	for i < len(previous) && j < len(current) {
		switch {
		case previous[i] == current[j]:
			i++
			j++
		case previous[i] < current[j]:
			removed = append(removed, previous[i])
			i++
		default:
			added = append(added, current[j])
			j++
		}
	}
	for ; i < len(previous); i++ {
		removed = append(removed, previous[i])
	}
	for ; j < len(current); j++ {
		added = append(added, current[j])
	}
	return added, removed
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mcpInstructionsStateMatches(prior *model.MCPInstructionsDeltaState, next *model.MCPInstructionsDeltaState) bool {
	if prior == nil || next == nil {
		return prior == next
	}
	return prior.LastInstructionsFingerprint == next.LastInstructionsFingerprint &&
		equalStringSlices(prior.LastAnnouncedSearchableServers, next.LastAnnouncedSearchableServers) &&
		equalStringSlices(prior.LastAnnouncedPendingServers, next.LastAnnouncedPendingServers) &&
		equalStringSlices(prior.LastAnnouncedUnavailableServers, next.LastAnnouncedUnavailableServers)
}

func logDeferredSurfaceComputed(sessionID, mode string, debug DeferredSurfaceDebug) {
	logger.Debug("[DEFERRED_SURFACE] computed",
		"session_id", sessionID,
		"mode", mode,
		"current_config_digest", debug.CurrentConfigDigest,
		"bundle_config_digest", debug.BundleConfigDigest,
		"bundle_generation", debug.BundleGeneration,
		"searchable_pool", debug.SearchablePoolCanonicalNames,
		"loadable_pool", debug.LoadablePoolCanonicalNames,
		"effective_discovered", debug.EffectiveDiscoveredCanonicalNames,
		"current_loaded", debug.CurrentLoadedCanonicalNames,
		"tool_search_current_loaded", debug.ToolSearchCurrentLoadedCanonicalNames,
		"pending_mcp_servers", debug.PendingMCPServers,
		"tool_search_visible", debug.ToolSearchVisible,
		"announcement_state", debug.AnnouncementState.AnnouncedSearchableCanonicalNames,
		"announcement_mode", debug.AnnouncementPreview.Mode,
		"announcement_added", debug.AnnouncementPreview.Added,
		"announcement_removed", debug.AnnouncementPreview.Removed,
		"announcement_will_emit", debug.AnnouncementPreview.WillEmit,
		"instructions_state_searchable", debug.InstructionsState.LastAnnouncedSearchableServers,
		"instructions_state_pending", debug.InstructionsState.LastAnnouncedPendingServers,
		"instructions_state_unavailable", debug.InstructionsState.LastAnnouncedUnavailableServers,
		"instructions_state_fingerprint", debug.InstructionsState.LastInstructionsFingerprint,
		"instructions_summary_searchable", debug.InstructionsSummary.SearchableServers,
		"instructions_summary_pending", debug.InstructionsSummary.PendingServers,
		"instructions_summary_unavailable", debug.InstructionsSummary.UnavailableServers,
		"instructions_summary_fingerprint", debug.InstructionsSummary.Fingerprint,
		"instructions_will_emit", debug.InstructionsSummary.WillEmit,
		"has_config_snapshot_error", debug.ConfigSnapshotError != "",
		"build_warnings", debug.BuildWarnings,
	)
}

func logDeferredSurfaceCommitted(sessionID, mode string, announcement *model.DeferredAnnouncementState, updateAnnouncement bool, instructions *model.MCPInstructionsDeltaState, updateInstructions bool) {
	announcementState := normalizedAnnouncementStateValue(announcement)
	instructionsState := normalizedInstructionsStateValue(instructions)
	logger.Debug("[DEFERRED_SURFACE] committed",
		"session_id", sessionID,
		"mode", mode,
		"update_announcement", updateAnnouncement,
		"announcement_state", announcementState.AnnouncedSearchableCanonicalNames,
		"update_instructions", updateInstructions,
		"instructions_state_searchable", instructionsState.LastAnnouncedSearchableServers,
		"instructions_state_pending", instructionsState.LastAnnouncedPendingServers,
		"instructions_state_unavailable", instructionsState.LastAnnouncedUnavailableServers,
		"instructions_state_fingerprint", instructionsState.LastInstructionsFingerprint,
	)
}
