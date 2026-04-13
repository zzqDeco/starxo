package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	starxomodel "starxo/internal/model"
)

type DeferredSyntheticMessages struct {
	Messages []*schema.Message
	Commit   func()
}

type MCPInstructionsSummary struct {
	SearchableServers  []string
	PendingServers     []string
	UnavailableServers []string
	Fingerprint        string
}

type DeferredMCPProvider interface {
	DeferredMCPState(ctx context.Context) (DeferredMCPState, error)
	LookupCatalogEntry(name string) (CatalogEntry, bool)
	PrepareDeferredSyntheticMessages(ctx context.Context) (*DeferredSyntheticMessages, error)
}

func NewDynamicMCPSurfaceMiddleware(provider DeferredMCPProvider) adk.ChatModelAgentMiddleware {
	return &dynamicMCPSurfaceMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		provider:                     provider,
	}
}

type dynamicMCPSurfaceMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	provider DeferredMCPProvider
}

func (m *dynamicMCPSurfaceMiddleware) WrapModel(ctx context.Context, cm model.BaseChatModel, mc *adk.ModelContext) (model.BaseChatModel, error) {
	state, err := m.provider.DeferredMCPState(ctx)
	if err != nil {
		return nil, err
	}
	return &dynamicMCPModelWrapper{
		base:            cm,
		allTools:        mc.Tools,
		state:           state,
		catalogProvider: m.provider,
	}, nil
}

func (m *dynamicMCPSurfaceMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		if err := m.ensureToolCallable(ctx, tCtx.Name); err != nil {
			return "", err
		}
		return endpoint(ctx, argumentsInJSON, opts...)
	}, nil
}

func (m *dynamicMCPSurfaceMiddleware) WrapStreamableToolCall(ctx context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		if err := m.ensureToolCallable(ctx, tCtx.Name); err != nil {
			return nil, err
		}
		return endpoint(ctx, argumentsInJSON, opts...)
	}, nil
}

func (m *dynamicMCPSurfaceMiddleware) ensureToolCallable(ctx context.Context, toolName string) error {
	state, err := m.provider.DeferredMCPState(ctx)
	if err != nil {
		return err
	}
	if toolName == "tool_search" {
		if toolSearchVisible(state) {
			return nil
		}
		return fmt.Errorf("tool_search is unavailable because no deferred MCP tools are currently searchable")
	}
	entry, ok := m.provider.LookupCatalogEntry(toolName)
	if !ok {
		return nil
	}
	if state.IsCurrentlyLoaded(entry.CanonicalName) {
		return nil
	}
	if state.IsCurrentlySearchable(entry.CanonicalName) {
		return fmt.Errorf("tool %s is not currently loaded; use tool_search first", entry.CanonicalName)
	}
	if decision, ok := state.SearchDecisions[entry.CanonicalName]; ok && decision.Reason != "" {
		return fmt.Errorf("tool %s is unavailable in the current mode or runtime: %s", entry.CanonicalName, decision.Reason)
	}
	return fmt.Errorf("tool %s is unavailable in the current mode or runtime", entry.CanonicalName)
}

type dynamicMCPModelWrapper struct {
	base            model.BaseChatModel
	allTools        []*schema.ToolInfo
	state           DeferredMCPState
	catalogProvider DeferredMCPProvider
}

func (w *dynamicMCPModelWrapper) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	prepared, err := w.catalogProvider.PrepareDeferredSyntheticMessages(ctx)
	if err != nil {
		return nil, err
	}
	msgs := injectSyntheticMessages(input, prepared)
	tools := filterVisibleToolInfos(w.allTools, w.state, w.catalogProvider)
	msg, err := w.base.Generate(ctx, msgs, append(opts, model.WithTools(tools))...)
	if err != nil {
		return nil, err
	}
	if prepared != nil && prepared.Commit != nil {
		prepared.Commit()
	}
	return msg, nil
}

func (w *dynamicMCPModelWrapper) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	prepared, err := w.catalogProvider.PrepareDeferredSyntheticMessages(ctx)
	if err != nil {
		return nil, err
	}
	msgs := injectSyntheticMessages(input, prepared)
	tools := filterVisibleToolInfos(w.allTools, w.state, w.catalogProvider)
	stream, err := w.base.Stream(ctx, msgs, append(opts, model.WithTools(tools))...)
	if err != nil {
		return nil, err
	}
	if prepared != nil && prepared.Commit != nil {
		prepared.Commit()
	}
	return stream, nil
}

func toolSearchVisible(state DeferredMCPState) bool {
	return len(state.SearchablePoolForMode) > 0 || len(state.PendingMCPServers) > 0
}

func NormalizeSearchableCanonicalNames(entries []CatalogEntry) []string {
	if len(entries) == 0 {
		return []string{}
	}
	names := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.CanonicalName == "" {
			continue
		}
		if _, ok := seen[entry.CanonicalName]; ok {
			continue
		}
		seen[entry.CanonicalName] = struct{}{}
		names = append(names, entry.CanonicalName)
	}
	if len(names) == 0 {
		return []string{}
	}
	sort.Strings(names)
	return names
}

func BuildDeferredToolsDeltaMessage(mode string, added, removed []string) *schema.Message {
	var b strings.Builder
	b.WriteString("<deferred-tools-delta>\n")
	b.WriteString("mode: ")
	b.WriteString(mode)
	b.WriteString("\n")
	b.WriteString("added:\n")
	for _, name := range added {
		b.WriteString(name)
		b.WriteString("\n")
	}
	b.WriteString("removed:\n")
	for _, name := range removed {
		b.WriteString(name)
		b.WriteString("\n")
	}
	b.WriteString("</deferred-tools-delta>")
	return schema.UserMessage(b.String())
}

func CloneNormalizedCanonicalNames(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func NormalizedAnnouncementState(names []string) *starxomodel.DeferredAnnouncementState {
	return &starxomodel.DeferredAnnouncementState{
		AnnouncedSearchableCanonicalNames: CloneNormalizedCanonicalNames(names),
	}
}

func BuildDeferredAnnouncementDelta(current []string, prior *starxomodel.DeferredAnnouncementState) (*schema.Message, *starxomodel.DeferredAnnouncementState) {
	next := NormalizedAnnouncementState(current)
	if prior == nil {
		if len(current) == 0 {
			return nil, next
		}
		return BuildDeferredToolsDeltaMessage("bootstrap", current, []string{}), next
	}
	previous := CloneNormalizedCanonicalNames(prior.AnnouncedSearchableCanonicalNames)
	added, removed := diffSortedStrings(previous, current)
	if len(added) == 0 && len(removed) == 0 {
		return nil, next
	}
	return BuildDeferredToolsDeltaMessage("delta", added, removed), next
}

func NormalizeMCPInstructionsSummary(state DeferredMCPState, permCtx ToolPermissionContext) MCPInstructionsSummary {
	searchableSet := make(map[string]struct{})
	for _, entry := range state.SearchablePoolForMode {
		if !entry.IsMcp || entry.Server == "" {
			continue
		}
		searchableSet[entry.Server] = struct{}{}
	}

	searchable := make([]string, 0, len(searchableSet))
	pending := make([]string, 0, len(permCtx.Servers))
	unavailable := make([]string, 0, len(permCtx.Servers))
	for serverName, server := range permCtx.Servers {
		switch server.State {
		case MCPServerStatePending:
			pending = append(pending, serverName)
		case MCPServerStateNeedsAuth, MCPServerStateDisabled, MCPServerStateFailed:
			unavailable = append(unavailable, fmt.Sprintf("%s:%s", serverName, mcpUnavailableReasonClass(server.State)))
		}
	}
	for serverName := range searchableSet {
		searchable = append(searchable, serverName)
	}
	sort.Strings(searchable)
	sort.Strings(pending)
	sort.Strings(unavailable)
	return MCPInstructionsSummary{
		SearchableServers:  searchable,
		PendingServers:     pending,
		UnavailableServers: unavailable,
		Fingerprint:        ComputeMCPInstructionsFingerprint(searchable, pending, unavailable),
	}
}

func ComputeMCPInstructionsFingerprint(searchable, pending, unavailable []string) string {
	var b strings.Builder
	b.WriteString("searchable:")
	b.WriteString(strings.Join(searchable, "\n"))
	b.WriteString("\npending:")
	b.WriteString(strings.Join(pending, "\n"))
	b.WriteString("\nunavailable:")
	b.WriteString(strings.Join(unavailable, "\n"))
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func NormalizedMCPInstructionsState(summary MCPInstructionsSummary) *starxomodel.MCPInstructionsDeltaState {
	return &starxomodel.MCPInstructionsDeltaState{
		LastAnnouncedSearchableServers:  CloneNormalizedCanonicalNames(summary.SearchableServers),
		LastAnnouncedPendingServers:     CloneNormalizedCanonicalNames(summary.PendingServers),
		LastAnnouncedUnavailableServers: CloneNormalizedCanonicalNames(summary.UnavailableServers),
		LastInstructionsFingerprint:     summary.Fingerprint,
	}
}

func BuildMCPInstructionsDeltaMessage(summary MCPInstructionsSummary, prior *starxomodel.MCPInstructionsDeltaState) (*schema.Message, *starxomodel.MCPInstructionsDeltaState) {
	next := NormalizedMCPInstructionsState(summary)
	if prior == nil {
		if len(summary.SearchableServers) == 0 && len(summary.PendingServers) == 0 && len(summary.UnavailableServers) == 0 {
			return nil, next
		}
		return buildMCPInstructionsSummaryMessage(summary), next
	}
	if mcpInstructionsStateEqual(prior, next) {
		return nil, next
	}
	return buildMCPInstructionsSummaryMessage(summary), next
}

func buildMCPInstructionsSummaryMessage(summary MCPInstructionsSummary) *schema.Message {
	var b strings.Builder
	b.WriteString("<mcp-instructions-delta>\n")
	b.WriteString("searchable_servers:\n")
	for _, name := range summary.SearchableServers {
		b.WriteString(name)
		b.WriteString("\n")
	}
	b.WriteString("pending_servers:\n")
	for _, name := range summary.PendingServers {
		b.WriteString(name)
		b.WriteString("\n")
	}
	b.WriteString("unavailable_servers:\n")
	for _, name := range summary.UnavailableServers {
		b.WriteString(name)
		b.WriteString("\n")
	}
	b.WriteString("</mcp-instructions-delta>")
	return schema.UserMessage(b.String())
}

func mcpUnavailableReasonClass(state MCPServerState) string {
	switch state {
	case MCPServerStateNeedsAuth:
		return "needs_auth"
	case MCPServerStateDisabled:
		return "disabled"
	case MCPServerStateFailed:
		return "failed"
	default:
		return string(state)
	}
}

func mcpInstructionsStateEqual(left, right *starxomodel.MCPInstructionsDeltaState) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.LastInstructionsFingerprint == right.LastInstructionsFingerprint &&
		slices.Equal(left.LastAnnouncedSearchableServers, right.LastAnnouncedSearchableServers) &&
		slices.Equal(left.LastAnnouncedPendingServers, right.LastAnnouncedPendingServers) &&
		slices.Equal(left.LastAnnouncedUnavailableServers, right.LastAnnouncedUnavailableServers)
}

func injectSyntheticMessages(input []*schema.Message, prepared *DeferredSyntheticMessages) []*schema.Message {
	if prepared == nil || len(prepared.Messages) == 0 {
		return input
	}
	if len(input) == 0 {
		return append([]*schema.Message(nil), prepared.Messages...)
	}

	out := make([]*schema.Message, 0, len(input)+len(prepared.Messages))
	if input[0].Role == schema.System {
		out = append(out, input[0])
		out = append(out, prepared.Messages...)
		out = append(out, input[1:]...)
		return out
	}
	out = append(out, prepared.Messages...)
	out = append(out, input...)
	return out
}

func diffSortedStrings(previous, current []string) (added, removed []string) {
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

func filterVisibleToolInfos(all []*schema.ToolInfo, state DeferredMCPState, provider DeferredMCPProvider) []*schema.ToolInfo {
	loaded := make(map[string]struct{}, len(state.CurrentLoadedTools))
	for _, entry := range state.CurrentLoadedTools {
		loaded[entry.CanonicalName] = struct{}{}
	}

	visible := make([]*schema.ToolInfo, 0, len(all))
	for _, info := range all {
		if info == nil {
			continue
		}
		if info.Name == "tool_search" {
			if toolSearchVisible(state) {
				visible = append(visible, info)
			}
			continue
		}
		if entry, ok := provider.LookupCatalogEntry(info.Name); ok {
			if _, exists := loaded[entry.CanonicalName]; exists {
				visible = append(visible, info)
			}
			continue
		}
		visible = append(visible, info)
	}
	return visible
}
