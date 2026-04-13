package tools

import (
	"testing"

	"starxo/internal/model"
)

func TestComputeDeferredMCPState_DefaultModeRespectsSearchableAndLoadablePools(t *testing.T) {
	catalog := NewToolCatalog()

	connected := stubCatalogEntry("mcp__alpha__connected")
	connected.Server = "alpha"

	pendingCached := stubCatalogEntry("mcp__beta__cached")
	pendingCached.Server = "beta"

	pendingNoCache := stubCatalogEntry("mcp__gamma__cold")
	pendingNoCache.Server = "gamma"

	alwaysLoaded := stubCatalogEntry("mcp__alpha__always")
	alwaysLoaded.Server = "alpha"
	alwaysLoaded.AlwaysLoad = true

	nonDeferred := stubCatalogEntry("mcp__alpha__direct")
	nonDeferred.Server = "alpha"
	nonDeferred.ShouldDefer = false

	resource := CatalogEntry{
		CanonicalName:   ReadMCPResourceName,
		Source:          ToolSourceMCP,
		Kind:            ToolKindResourceRead,
		ShouldDefer:     true,
		IsMcp:           true,
		IsResourceTool:  true,
		ReadOnlyHint:    true,
		ReadOnlyTrusted: true,
		PermissionSpec: PermissionSpec{
			AllowSearch:  true,
			AllowExecute: true,
		},
		Tool: &stubInvokableTool{name: ReadMCPResourceName},
	}

	for _, entry := range []CatalogEntry{connected, pendingCached, pendingNoCache, alwaysLoaded, nonDeferred, resource} {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register %s: %v", entry.CanonicalName, err)
		}
	}

	state := ComputeDeferredMCPState(catalog, map[string]model.DiscoveredToolRecord{
		connected.CanonicalName:      {CanonicalName: connected.CanonicalName},
		pendingCached.CanonicalName:  {CanonicalName: pendingCached.CanonicalName},
		pendingNoCache.CanonicalName: {CanonicalName: pendingNoCache.CanonicalName},
		nonDeferred.CanonicalName:    {CanonicalName: nonDeferred.CanonicalName},
	}, ToolPermissionContext{
		SessionID: "sess-1",
		Mode:      "default",
		Servers: map[string]MCPServerPermissionState{
			"alpha": {State: MCPServerStateConnected, HasCachedToolMetadata: true, SupportsResources: true},
			"beta":  {State: MCPServerStatePending, HasCachedToolMetadata: true},
			"gamma": {State: MCPServerStatePending, HasCachedToolMetadata: false},
		},
	})

	assertCatalogNames(t, state.SearchablePoolForMode, []string{
		connected.CanonicalName,
		pendingCached.CanonicalName,
		ReadMCPResourceName,
	})
	assertCatalogNames(t, state.LoadablePoolForMode, []string{
		connected.CanonicalName,
		ReadMCPResourceName,
	})
	assertCatalogNames(t, state.EffectiveDiscovered, []string{
		connected.CanonicalName,
	})
	assertCatalogNames(t, state.CurrentLoadedTools, []string{
		alwaysLoaded.CanonicalName,
		connected.CanonicalName,
	})
	assertStrings(t, state.PendingMCPServers, []string{"beta", "gamma"})
	if _, ok := state.SearchDecisions[alwaysLoaded.CanonicalName]; !ok {
		t.Fatalf("expected always-loaded entry to retain search decision coverage")
	}
	if _, ok := state.LoadDecisions[nonDeferred.CanonicalName]; !ok {
		t.Fatalf("expected non-deferred entry to retain load decision coverage")
	}
}

func TestComputeDeferredMCPState_PlanModeOnlyAllowsTrustedReadOnlyTools(t *testing.T) {
	catalog := NewToolCatalog()

	readOnly := stubCatalogEntry("mcp__alpha__readonly")
	readOnly.Server = "alpha"
	readOnly.ReadOnlyHint = true
	readOnly.ReadOnlyTrusted = true

	untrusted := stubCatalogEntry("mcp__alpha__untrusted")
	untrusted.Server = "alpha"
	untrusted.ReadOnlyHint = true
	untrusted.ReadOnlyTrusted = false

	readWrite := stubCatalogEntry("mcp__alpha__write")
	readWrite.Server = "alpha"

	for _, entry := range []CatalogEntry{readOnly, untrusted, readWrite} {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register %s: %v", entry.CanonicalName, err)
		}
	}

	state := ComputeDeferredMCPState(catalog, map[string]model.DiscoveredToolRecord{
		readOnly.CanonicalName:  {CanonicalName: readOnly.CanonicalName},
		untrusted.CanonicalName: {CanonicalName: untrusted.CanonicalName},
		readWrite.CanonicalName: {CanonicalName: readWrite.CanonicalName},
	}, ToolPermissionContext{
		SessionID: "sess-plan",
		Mode:      "plan",
		Servers: map[string]MCPServerPermissionState{
			"alpha": {State: MCPServerStateConnected, HasCachedToolMetadata: true},
		},
	})

	assertCatalogNames(t, state.SearchablePoolForMode, []string{readOnly.CanonicalName})
	assertCatalogNames(t, state.LoadablePoolForMode, []string{readOnly.CanonicalName})
	assertCatalogNames(t, state.EffectiveDiscovered, []string{readOnly.CanonicalName})
	assertCatalogNames(t, state.CurrentLoadedTools, []string{readOnly.CanonicalName})
}

func TestComputeDeferredMCPState_PlanModeDoesNotFilterHiddenNonMCPSample(t *testing.T) {
	catalog := NewToolCatalog()

	hiddenSample := stubDeferredBuiltinSample("hidden_builtin_sample")
	if err := catalog.Register(hiddenSample); err != nil {
		t.Fatalf("register hidden sample: %v", err)
	}

	state := ComputeDeferredMCPState(catalog, map[string]model.DiscoveredToolRecord{
		hiddenSample.CanonicalName: {CanonicalName: hiddenSample.CanonicalName},
	}, ToolPermissionContext{
		SessionID: "sess-plan",
		Mode:      "plan",
		Servers:   map[string]MCPServerPermissionState{},
	})

	assertCatalogNames(t, state.SearchablePoolForMode, []string{hiddenSample.CanonicalName})
	assertCatalogNames(t, state.LoadablePoolForMode, []string{hiddenSample.CanonicalName})
	assertCatalogNames(t, state.EffectiveDiscovered, []string{hiddenSample.CanonicalName})
	assertCatalogNames(t, state.CurrentLoadedTools, []string{hiddenSample.CanonicalName})
}

func assertCatalogNames(t *testing.T, entries []CatalogEntry, want []string) {
	t.Helper()
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.CanonicalName)
	}
	assertStrings(t, got, want)
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected length: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected value at %d: got=%v want=%v", i, got, want)
		}
	}
}
