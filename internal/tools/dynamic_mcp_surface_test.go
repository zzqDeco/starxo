package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	model2 "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	starxomodel "starxo/internal/model"
)

type fakeDeferredProvider struct {
	state   DeferredMCPState
	catalog *ToolCatalog
	prepare func(context.Context) (*DeferredSyntheticMessages, error)
}

func (p *fakeDeferredProvider) DeferredMCPState(context.Context) (DeferredMCPState, error) {
	return p.state, nil
}

func (p *fakeDeferredProvider) LookupCatalogEntry(name string) (CatalogEntry, bool) {
	if p.catalog == nil {
		return CatalogEntry{}, false
	}
	return p.catalog.LookupExact(name)
}

func (p *fakeDeferredProvider) PrepareDeferredSyntheticMessages(ctx context.Context) (*DeferredSyntheticMessages, error) {
	if p.prepare == nil {
		return nil, nil
	}
	return p.prepare(ctx)
}

type fakeBaseChatModel struct {
	generateFn func(context.Context, []*schema.Message, ...model2.Option) (*schema.Message, error)
	streamFn   func(context.Context, []*schema.Message, ...model2.Option) (*schema.StreamReader[*schema.Message], error)
}

func (m *fakeBaseChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model2.Option) (*schema.Message, error) {
	if m.generateFn != nil {
		return m.generateFn(ctx, input, opts...)
	}
	return schema.AssistantMessage("ok", nil), nil
}

func (m *fakeBaseChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model2.Option) (*schema.StreamReader[*schema.Message], error) {
	if m.streamFn != nil {
		return m.streamFn(ctx, input, opts...)
	}
	return schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("ok", nil)}), nil
}

func TestDynamicMCPSurface_FilterVisibleToolInfos(t *testing.T) {
	catalog := NewToolCatalog()
	loaded := stubCatalogEntry("mcp__fs__grep")
	hidden := stubCatalogEntry("mcp__fs__find")
	for _, entry := range []CatalogEntry{loaded, hidden} {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register %s: %v", entry.CanonicalName, err)
		}
	}

	visible := filterVisibleToolInfos([]*schema.ToolInfo{
		{Name: "ask_user"},
		{Name: "tool_search"},
		{Name: loaded.CanonicalName},
		{Name: hidden.CanonicalName},
	}, DeferredMCPState{
		SearchablePoolForMode: []CatalogEntry{loaded, hidden},
		CurrentLoadedTools:    []CatalogEntry{loaded},
	}, &fakeDeferredProvider{catalog: catalog})

	got := make([]string, 0, len(visible))
	for _, info := range visible {
		got = append(got, info.Name)
	}
	assertStrings(t, got, []string{"ask_user", "tool_search", loaded.CanonicalName})
}

func TestDynamicMCPSurface_FilterVisibleToolInfos_HidesToolSearchWithoutDeferredButKeepsAlwaysLoadedVisible(t *testing.T) {
	catalog := NewToolCatalog()
	alwaysLoaded := stubCatalogEntry("mcp__alpha__always")
	alwaysLoaded.AlwaysLoad = true
	nonDeferred := stubCatalogEntry("mcp__alpha__direct")
	nonDeferred.ShouldDefer = false

	for _, entry := range []CatalogEntry{alwaysLoaded, nonDeferred} {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register %s: %v", entry.CanonicalName, err)
		}
	}

	visible := filterVisibleToolInfos([]*schema.ToolInfo{
		{Name: "ask_user"},
		{Name: "tool_search"},
		{Name: alwaysLoaded.CanonicalName},
		{Name: nonDeferred.CanonicalName},
	}, DeferredMCPState{
		CurrentLoadedTools: []CatalogEntry{alwaysLoaded},
	}, &fakeDeferredProvider{catalog: catalog})

	got := make([]string, 0, len(visible))
	for _, info := range visible {
		got = append(got, info.Name)
	}
	assertStrings(t, got, []string{"ask_user", alwaysLoaded.CanonicalName})
}

func TestDynamicMCPSurface_NormalizeSearchableCanonicalNamesSortsAndDedupes(t *testing.T) {
	got := NormalizeSearchableCanonicalNames([]CatalogEntry{
		{CanonicalName: "mcp__git__status"},
		{CanonicalName: "mcp__fs__grep"},
		{CanonicalName: "mcp__git__status"},
		{CanonicalName: ""},
	})
	assertStrings(t, got, []string{"mcp__fs__grep", "mcp__git__status"})
}

func TestDynamicMCPSurface_BuildDeferredAnnouncementDeltaBootstrapUsesCanonicalNamesOnly(t *testing.T) {
	msg, next := BuildDeferredAnnouncementDelta([]string{"mcp__fs__grep", "mcp__git__status"}, nil)
	if msg == nil {
		t.Fatal("expected announcement message")
	}
	if !strings.Contains(msg.Content, "mcp__fs__grep") || !strings.Contains(msg.Content, "mcp__git__status") {
		t.Fatalf("announcement missing canonical names: %q", msg.Content)
	}
	if !strings.Contains(msg.Content, "mode: bootstrap") {
		t.Fatalf("expected bootstrap mode, got %q", msg.Content)
	}
	if !strings.Contains(msg.Content, "removed:\n</deferred-tools-delta>") {
		t.Fatalf("expected empty removed section to be preserved, got %q", msg.Content)
	}
	if next == nil {
		t.Fatal("expected next state")
	}
	assertStrings(t, next.AnnouncedSearchableCanonicalNames, []string{"mcp__fs__grep", "mcp__git__status"})
}

func TestDynamicMCPSurface_BuildDeferredAnnouncementDeltaDeltaOnlyWritesChanges(t *testing.T) {
	msg, next := BuildDeferredAnnouncementDelta(
		[]string{"mcp__fs__grep", "mcp__git__status"},
		&starxomodel.DeferredAnnouncementState{AnnouncedSearchableCanonicalNames: []string{"mcp__docker__logs", "mcp__git__status"}},
	)
	if msg == nil {
		t.Fatal("expected delta message")
	}
	if !strings.Contains(msg.Content, "mode: delta") {
		t.Fatalf("expected delta mode, got %q", msg.Content)
	}
	if !strings.Contains(msg.Content, "added:\nmcp__fs__grep\n") {
		t.Fatalf("expected added canonical, got %q", msg.Content)
	}
	if !strings.Contains(msg.Content, "removed:\nmcp__docker__logs\n") {
		t.Fatalf("expected removed canonical, got %q", msg.Content)
	}
	assertStrings(t, next.AnnouncedSearchableCanonicalNames, []string{"mcp__fs__grep", "mcp__git__status"})
}

func TestDynamicMCPSurface_EnsureToolCallable(t *testing.T) {
	catalog := NewToolCatalog()
	loaded := stubCatalogEntry("mcp__fs__grep")
	hidden := stubCatalogEntry("mcp__fs__find")
	blocked := stubCatalogEntry("mcp__fs__write")
	alwaysLoaded := stubCatalogEntry("mcp__fs__resource_index")
	alwaysLoaded.AlwaysLoad = true
	for _, entry := range []CatalogEntry{loaded, hidden, blocked, alwaysLoaded} {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register %s: %v", entry.CanonicalName, err)
		}
	}

	provider := &fakeDeferredProvider{
		catalog: catalog,
		state: DeferredMCPState{
			SearchablePoolForMode: []CatalogEntry{hidden},
			CurrentLoadedTools:    []CatalogEntry{alwaysLoaded, loaded},
			SearchDecisions: map[string]PermissionDecision{
				blocked.CanonicalName: {Allowed: false, Reason: "tool is not read-only in plan mode"},
			},
		},
	}
	mw := &dynamicMCPSurfaceMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		provider:                     provider,
	}

	if err := mw.ensureToolCallable(context.Background(), loaded.CanonicalName); err != nil {
		t.Fatalf("expected loaded tool to be callable, got %v", err)
	}
	if err := mw.ensureToolCallable(context.Background(), alwaysLoaded.CanonicalName); err != nil {
		t.Fatalf("expected always-loaded tool to remain callable, got %v", err)
	}
	if err := mw.ensureToolCallable(context.Background(), hidden.CanonicalName); err == nil || !strings.Contains(err.Error(), "use tool_search first") {
		t.Fatalf("expected deferred tool rejection, got %v", err)
	}
	if err := mw.ensureToolCallable(context.Background(), blocked.CanonicalName); err == nil || !strings.Contains(err.Error(), "not read-only in plan mode") {
		t.Fatalf("expected blocked tool rejection, got %v", err)
	}

	provider.state = DeferredMCPState{}
	if err := mw.ensureToolCallable(context.Background(), "tool_search"); err == nil || err.Error() != ToolSearchUnavailableNoDeferredMessage {
		t.Fatalf("expected hidden tool_search rejection %q, got %v", ToolSearchUnavailableNoDeferredMessage, err)
	}
}

func TestDynamicMCPSurface_GenerateCommitsPreparedStateOnlyOnSuccess(t *testing.T) {
	var commits int
	provider := &fakeDeferredProvider{
		prepare: func(context.Context) (*DeferredSyntheticMessages, error) {
			return &DeferredSyntheticMessages{
				Messages: []*schema.Message{schema.UserMessage("<deferred-tools-delta>\nmode: bootstrap\nadded:\na\nremoved:\n</deferred-tools-delta>")},
				Commit:   func() { commits++ },
			}, nil
		},
	}
	var captured []*schema.Message
	wrapper := &dynamicMCPModelWrapper{
		base: &fakeBaseChatModel{
			generateFn: func(_ context.Context, input []*schema.Message, _ ...model2.Option) (*schema.Message, error) {
				captured = input
				return schema.AssistantMessage("ok", nil), nil
			},
		},
		allTools:        []*schema.ToolInfo{{Name: "ask_user"}},
		state:           DeferredMCPState{},
		catalogProvider: provider,
	}

	msg, err := wrapper.Generate(context.Background(), []*schema.Message{schema.SystemMessage("sys"), schema.UserMessage("hi")})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if msg == nil {
		t.Fatal("expected response message")
	}
	if commits != 1 {
		t.Fatalf("expected commit once, got %d", commits)
	}
	if len(captured) < 2 || captured[1].Role != schema.User {
		t.Fatalf("expected synthetic user message after system, got %#v", captured)
	}

	provider.prepare = func(context.Context) (*DeferredSyntheticMessages, error) {
		return &DeferredSyntheticMessages{
			Messages: []*schema.Message{schema.UserMessage("<deferred-tools-delta>\nmode: bootstrap\nadded:\na\nremoved:\n</deferred-tools-delta>")},
			Commit:   func() { commits++ },
		}, nil
	}
	wrapper.base = &fakeBaseChatModel{
		generateFn: func(_ context.Context, _ []*schema.Message, _ ...model2.Option) (*schema.Message, error) {
			return nil, errors.New("boom")
		},
	}
	if _, err := wrapper.Generate(context.Background(), []*schema.Message{schema.SystemMessage("sys")}); err == nil {
		t.Fatal("expected generate error")
	}
	if commits != 1 {
		t.Fatalf("expected failed generate not to commit, got %d", commits)
	}
}

func TestDynamicMCPSurface_BuildMCPInstructionsDeltaMessageBootstrapAndStableFingerprint(t *testing.T) {
	summary := NormalizeMCPInstructionsSummary(DeferredMCPState{
		SearchablePoolForMode: []CatalogEntry{
			{CanonicalName: "mcp__alpha__grep", Server: "alpha", IsMcp: true},
			{CanonicalName: "mcp__alpha__status", Server: "alpha", IsMcp: true},
		},
	}, ToolPermissionContext{
		Servers: map[string]MCPServerPermissionState{
			"alpha": {State: MCPServerStateConnected},
			"beta":  {State: MCPServerStatePending},
			"gamma": {State: MCPServerStateFailed},
		},
	})
	msg, next := BuildMCPInstructionsDeltaMessage(summary, nil)
	if msg == nil {
		t.Fatal("expected bootstrap instructions message")
	}
	if !strings.Contains(msg.Content, "searchable_servers:\nalpha\n") {
		t.Fatalf("expected searchable server snapshot, got %q", msg.Content)
	}
	if !strings.Contains(msg.Content, "pending_servers:\nbeta\n") {
		t.Fatalf("expected pending server snapshot, got %q", msg.Content)
	}
	if !strings.Contains(msg.Content, "unavailable_servers:\ngamma:failed\n") {
		t.Fatalf("expected unavailable server snapshot, got %q", msg.Content)
	}
	if next == nil || next.LastInstructionsFingerprint == "" {
		t.Fatalf("expected normalized instructions state with fingerprint, got %#v", next)
	}

	same, nextSame := BuildMCPInstructionsDeltaMessage(summary, next)
	if same != nil {
		t.Fatalf("expected same summary not to emit a second message, got %q", same.Content)
	}
	if nextSame == nil || nextSame.LastInstructionsFingerprint != next.LastInstructionsFingerprint {
		t.Fatalf("expected stable fingerprint, got %#v", nextSame)
	}
}
