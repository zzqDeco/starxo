package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type fakeDeferredProvider struct {
	state   DeferredMCPState
	catalog *ToolCatalog
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

func TestDynamicMCPSurface_AnnouncementUsesCanonicalNamesOnly(t *testing.T) {
	msg := buildDeferredAnnouncement(DeferredMCPState{
		SearchablePoolForMode: []CatalogEntry{
			{CanonicalName: "mcp__fs__grep", SearchHint: "ignore me"},
			{CanonicalName: "mcp__git__status", SearchHint: "also hidden"},
		},
	})
	if msg == nil {
		t.Fatal("expected announcement message")
	}
	if !strings.Contains(msg.Content, "mcp__fs__grep") || !strings.Contains(msg.Content, "mcp__git__status") {
		t.Fatalf("announcement missing canonical names: %q", msg.Content)
	}
	if strings.Contains(msg.Content, "ignore me") || strings.Contains(msg.Content, "also hidden") {
		t.Fatalf("announcement leaked search hints: %q", msg.Content)
	}
}

func TestDynamicMCPSurface_EnsureToolCallable(t *testing.T) {
	catalog := NewToolCatalog()
	loaded := stubCatalogEntry("mcp__fs__grep")
	hidden := stubCatalogEntry("mcp__fs__find")
	blocked := stubCatalogEntry("mcp__fs__write")
	for _, entry := range []CatalogEntry{loaded, hidden, blocked} {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register %s: %v", entry.CanonicalName, err)
		}
	}

	provider := &fakeDeferredProvider{
		catalog: catalog,
		state: DeferredMCPState{
			SearchablePoolForMode: []CatalogEntry{hidden},
			CurrentLoadedTools:    []CatalogEntry{loaded},
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
	if err := mw.ensureToolCallable(context.Background(), hidden.CanonicalName); err == nil || !strings.Contains(err.Error(), "use tool_search first") {
		t.Fatalf("expected deferred tool rejection, got %v", err)
	}
	if err := mw.ensureToolCallable(context.Background(), blocked.CanonicalName); err == nil || !strings.Contains(err.Error(), "not read-only in plan mode") {
		t.Fatalf("expected blocked tool rejection, got %v", err)
	}

	provider.state = DeferredMCPState{}
	if err := mw.ensureToolCallable(context.Background(), "tool_search"); err == nil || !strings.Contains(err.Error(), "currently searchable") {
		t.Fatalf("expected hidden tool_search rejection, got %v", err)
	}
}
