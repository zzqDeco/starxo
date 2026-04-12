package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"

	"starxo/internal/config"
	"starxo/internal/model"
	"starxo/internal/tools"
)

func newTestConfigStore(t *testing.T) *config.Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	store, err := config.NewStore()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}
	return store
}

func setTestMCPServers(t *testing.T, store *config.Store, servers []config.MCPServerConfig) {
	t.Helper()
	if err := store.Update(func(cfg *config.AppConfig) {
		cfg.MCP.Servers = append([]config.MCPServerConfig(nil), servers...)
	}); err != nil {
		t.Fatalf("update MCP config: %v", err)
	}
}

func updateTestLLMModel(t *testing.T, store *config.Store, model string) {
	t.Helper()
	if err := store.Update(func(cfg *config.AppConfig) {
		cfg.LLM.Model = model
	}); err != nil {
		t.Fatalf("update LLM config: %v", err)
	}
}

func markSessionStarting(chat *ChatService, sessionID string) {
	chat.mu.Lock()
	defer chat.mu.Unlock()
	run := chat.getOrCreateRun(sessionID)
	run.starting = true
}

func TestResolvePendingRunnerLockedUsesInterruptBundleGenerationAndKind(t *testing.T) {
	chat := NewChatService(nil)

	retiredDefault := &adk.Runner{}
	retiredPlan := &adk.Runner{}
	installedDefault := &adk.Runner{}
	installedPlan := &adk.Runner{}

	chat.mu.Lock()
	run := chat.getOrCreateRun("sess-1")
	run.mode = "plan"
	run.pendingInterrupt = &PendingInterrupt{
		BundleGeneration: 1,
		RunnerKind:       RunnerKindDefault,
	}
	chat.installedBundle = &RunnerBundle{
		Generation:    2,
		DefaultRunner: installedDefault,
		PlanRunner:    installedPlan,
	}
	chat.retiredBundles = []*RunnerBundle{{
		Generation:    1,
		DefaultRunner: retiredDefault,
		PlanRunner:    retiredPlan,
	}}

	bundle, runner, _, err := chat.resolvePendingRunnerLocked(run)
	chat.mu.Unlock()
	if err != nil {
		t.Fatalf("resolve pending runner: %v", err)
	}
	if bundle.Generation != 1 {
		t.Fatalf("expected retired bundle generation 1, got %d", bundle.Generation)
	}
	if runner != retiredDefault {
		t.Fatalf("expected retired default runner, got %#v", runner)
	}

	chat.mu.Lock()
	run.mode = "default"
	run.pendingInterrupt = &PendingInterrupt{
		BundleGeneration: 1,
		RunnerKind:       RunnerKindPlan,
	}
	bundle, runner, _, err = chat.resolvePendingRunnerLocked(run)
	chat.mu.Unlock()
	if err != nil {
		t.Fatalf("resolve pending plan runner: %v", err)
	}
	if bundle.Generation != 1 {
		t.Fatalf("expected retired bundle generation 1 for plan resume, got %d", bundle.Generation)
	}
	if runner != retiredPlan {
		t.Fatalf("expected retired plan runner, got %#v", runner)
	}
}

func TestResolvePendingRunnerLockedFailsWithoutFallbackWhenBundleMissing(t *testing.T) {
	chat := NewChatService(nil)

	chat.mu.Lock()
	run := chat.getOrCreateRun("sess-1")
	run.mode = "default"
	run.pendingInterrupt = &PendingInterrupt{
		BundleGeneration: 9,
		RunnerKind:       RunnerKindPlan,
	}
	chat.installedBundle = &RunnerBundle{
		Generation:    10,
		DefaultRunner: &adk.Runner{},
		PlanRunner:    &adk.Runner{},
	}
	_, _, _, err := chat.resolvePendingRunnerLocked(run)
	chat.mu.Unlock()
	if err == nil {
		t.Fatal("expected missing bundle error")
	}
}

func TestCleanupRetiredBundlesLockedHonorsPendingInterruptReferences(t *testing.T) {
	chat := NewChatService(nil)
	bundle := &RunnerBundle{Generation: 3}

	chat.mu.Lock()
	run := chat.getOrCreateRun("sess-1")
	run.pendingInterrupt = &PendingInterrupt{BundleGeneration: 3, RunnerKind: RunnerKindDefault}
	chat.retiredBundles = []*RunnerBundle{bundle}
	chat.cleanupRetiredBundlesLocked()
	if len(chat.retiredBundles) != 1 {
		chat.mu.Unlock()
		t.Fatalf("expected retired bundle to be retained, got %d", len(chat.retiredBundles))
	}

	run.pendingInterrupt = nil
	chat.cleanupRetiredBundlesLocked()
	if len(chat.retiredBundles) != 0 {
		chat.mu.Unlock()
		t.Fatalf("expected retired bundle to be cleaned up, got %d", len(chat.retiredBundles))
	}
	chat.mu.Unlock()
}

func TestEnsureBundleReadyForNewRunSingleflightByInstalledBundleKey(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	markSessionStarting(chat, "sess-1")
	markSessionStarting(chat, "sess-2")

	_, digest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("config snapshot: %v", err)
	}

	chat.mu.Lock()
	chat.installedBundle = &RunnerBundle{
		Generation:                 7,
		ConfigDigest:               digest,
		DefaultRunner:              &adk.Runner{},
		PlanRunner:                 &adk.Runner{},
		LastFreshnessCheckAt:       time.Time{},
		SurfaceRelevantFingerprint: "same-fp",
	}
	chat.mu.Unlock()

	var probeCalls int32
	releaseProbe := make(chan struct{})
	enteredProbe := make(chan struct{}, 1)
	chat.probeBundleSurfaceFn = func(context.Context, *config.AppConfig, map[string]cachedMCPServerSurface) (*runnerBundleSurface, error) {
		if atomic.AddInt32(&probeCalls, 1) == 1 {
			enteredProbe <- struct{}{}
		}
		<-releaseProbe
		return &runnerBundleSurface{
			ActionCatalog:                 tools.NewToolCatalog(),
			CachedSurfaceMetadataByServer: map[string]cachedMCPServerSurface{},
			SurfaceRelevantFingerprint:    "same-fp",
		}, nil
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		sessionID := fmt.Sprintf("sess-%d", i+1)
		wg.Add(1)
		go func(sessionID string) {
			defer wg.Done()
			bundle, err := chat.ensureBundleReadyForNewRun(context.Background(), sessionID)
			if err != nil {
				errs <- err
				return
			}
			if bundle == nil || bundle.Generation != 7 {
				errs <- fmt.Errorf("unexpected bundle result: %#v", bundle)
			}
		}(sessionID)
	}

	<-enteredProbe
	close(releaseProbe)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("ensure bundle failed: %v", err)
		}
	}
	if got := atomic.LoadInt32(&probeCalls); got != 1 {
		t.Fatalf("expected exactly one freshness probe, got %d", got)
	}
}

func TestEnsureBundleReadyForNewRunStaleNoChangeDoesNotRewriteFreshnessOnNewBundle(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	markSessionStarting(chat, "sess-1")

	_, digest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("config snapshot: %v", err)
	}

	oldBundle := &RunnerBundle{
		Generation:                 1,
		ConfigDigest:               digest,
		DefaultRunner:              &adk.Runner{},
		PlanRunner:                 &adk.Runner{},
		LastFreshnessCheckAt:       time.Time{},
		SurfaceRelevantFingerprint: "same-fp",
	}
	expectedFreshness := time.Now()
	newBundle := &RunnerBundle{
		Generation:                 2,
		ConfigDigest:               digest,
		DefaultRunner:              &adk.Runner{},
		PlanRunner:                 &adk.Runner{},
		LastFreshnessCheckAt:       expectedFreshness,
		SurfaceRelevantFingerprint: "same-fp",
	}

	chat.mu.Lock()
	chat.installedBundle = oldBundle
	chat.mu.Unlock()

	releaseProbe := make(chan struct{})
	enteredProbe := make(chan struct{}, 1)
	chat.probeBundleSurfaceFn = func(context.Context, *config.AppConfig, map[string]cachedMCPServerSurface) (*runnerBundleSurface, error) {
		enteredProbe <- struct{}{}
		<-releaseProbe
		return &runnerBundleSurface{
			ActionCatalog:                 tools.NewToolCatalog(),
			CachedSurfaceMetadataByServer: map[string]cachedMCPServerSurface{},
			SurfaceRelevantFingerprint:    "same-fp",
		}, nil
	}

	done := make(chan error, 1)
	go func() {
		_, err := chat.ensureBundleReadyForNewRun(context.Background(), "sess-1")
		done <- err
	}()

	<-enteredProbe
	chat.mu.Lock()
	chat.installedBundle = newBundle
	chat.mu.Unlock()
	close(releaseProbe)

	if err := <-done; err != nil {
		t.Fatalf("ensure bundle failed: %v", err)
	}
	if got, want := newBundle.LastFreshnessCheckAt, expectedFreshness; !got.Equal(want) {
		t.Fatalf("expected stale probe result to leave new bundle freshness unchanged, got %v want %v", got, want)
	}
}

func TestEnsureBundleReadyForNewRunProbeFailureKeepsCurrentBundleStale(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	markSessionStarting(chat, "sess-1")

	_, digest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("config snapshot: %v", err)
	}

	bundle := &RunnerBundle{
		Generation:                 4,
		ConfigDigest:               digest,
		DefaultRunner:              &adk.Runner{},
		PlanRunner:                 &adk.Runner{},
		LastFreshnessCheckAt:       time.Time{},
		SurfaceRelevantFingerprint: "same-fp",
	}

	chat.mu.Lock()
	chat.installedBundle = bundle
	chat.mu.Unlock()

	chat.probeBundleSurfaceFn = func(context.Context, *config.AppConfig, map[string]cachedMCPServerSurface) (*runnerBundleSurface, error) {
		return nil, fmt.Errorf("network down")
	}

	got, err := chat.ensureBundleReadyForNewRun(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("ensure bundle should fall back to installed bundle, got %v", err)
	}
	if got != bundle {
		t.Fatalf("expected current installed bundle fallback, got %#v", got)
	}
	if !bundle.LastFreshnessCheckAt.IsZero() {
		t.Fatalf("expected failed probe to leave bundle stale, got %v", bundle.LastFreshnessCheckAt)
	}
}

func TestDeferredPermissionContextKeepsPendingServerSearchableWithCachedMetadata(t *testing.T) {
	chat := NewChatService(nil)
	entry := stubToolSearchCatalogEntry("mcp__alpha__grep", "alpha")
	cfgDigest, err := mcpServerConfigIdentityDigest(config.MCPServerConfig{
		Name:      "alpha",
		Transport: "stdio",
		Command:   "alpha",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("compute config identity digest: %v", err)
	}

	catalog := tools.NewToolCatalog()
	if err := catalog.Register(entry); err != nil {
		t.Fatalf("register cached entry: %v", err)
	}

	provider := &deferredMCPProvider{
		chat: chat,
		bundle: &RunnerBundle{
			MCPCatalog: catalog,
			MCPHandles: []*tools.MCPServerHandle{{
				Name:  "alpha",
				State: tools.MCPServerStatePending,
			}},
			CachedSurfaceMetadataByServer: map[string]cachedMCPServerSurface{
				"alpha": {
					ConfigIdentityDigest: cfgDigest,
					HasToolMetadata:      true,
					ActionEntries:        []tools.CatalogEntry{entry},
				},
			},
		},
	}

	state := tools.ComputeDeferredMCPState(catalog, nil, provider.permissionContext("sess-1", "default"))
	if len(state.SearchablePoolForMode) != 1 || state.SearchablePoolForMode[0].CanonicalName != entry.CanonicalName {
		t.Fatalf("expected pending cached tool to stay searchable, got %#v", state.SearchablePoolForMode)
	}
	if len(state.LoadablePoolForMode) != 0 {
		t.Fatalf("expected pending cached tool to remain unloadable, got %#v", state.LoadablePoolForMode)
	}
}

func TestMCPServerConfigIdentityDigest_IsDeterministic(t *testing.T) {
	base := config.MCPServerConfig{
		Name:      "alpha",
		Transport: "stdio",
		Command:   "alpha",
		Args:      []string{"--one", "--two"},
		URL:       "",
		Env: map[string]string{
			"B": "2",
			"A": "1",
		},
		Enabled: true,
	}
	same := config.MCPServerConfig{
		Name:      "alpha",
		Transport: "stdio",
		Command:   "alpha",
		Args:      []string{"--one", "--two"},
		URL:       "",
		Env: map[string]string{
			"A": "1",
			"B": "2",
		},
		Enabled: true,
	}
	reorderedArgs := same
	reorderedArgs.Args = []string{"--two", "--one"}
	nilCollections := config.MCPServerConfig{
		Name:      "beta",
		Transport: "http",
		URL:       "http://example.invalid",
		Enabled:   true,
	}
	emptyCollections := nilCollections
	emptyCollections.Args = []string{}
	emptyCollections.Env = map[string]string{}

	baseDigest, err := mcpServerConfigIdentityDigest(base)
	if err != nil {
		t.Fatalf("base digest: %v", err)
	}
	sameDigest, err := mcpServerConfigIdentityDigest(same)
	if err != nil {
		t.Fatalf("same digest: %v", err)
	}
	reorderedDigest, err := mcpServerConfigIdentityDigest(reorderedArgs)
	if err != nil {
		t.Fatalf("reordered digest: %v", err)
	}
	nilDigest, err := mcpServerConfigIdentityDigest(nilCollections)
	if err != nil {
		t.Fatalf("nil digest: %v", err)
	}
	emptyDigest, err := mcpServerConfigIdentityDigest(emptyCollections)
	if err != nil {
		t.Fatalf("empty digest: %v", err)
	}

	if baseDigest != sameDigest {
		t.Fatalf("expected identical config digests, got %q vs %q", baseDigest, sameDigest)
	}
	if baseDigest == reorderedDigest {
		t.Fatalf("expected args order to affect digest")
	}
	if nilDigest != emptyDigest {
		t.Fatalf("expected nil and empty collections to normalize equally, got %q vs %q", nilDigest, emptyDigest)
	}
}

func TestMatchingSurfaceCacheEntry_RequiresMatchingConfigIdentityDigest(t *testing.T) {
	currentDigest, err := mcpServerConfigIdentityDigest(config.MCPServerConfig{
		Name:      "alpha",
		Transport: "stdio",
		Command:   "alpha-new",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("current digest: %v", err)
	}
	oldDigest, err := mcpServerConfigIdentityDigest(config.MCPServerConfig{
		Name:      "alpha",
		Transport: "stdio",
		Command:   "alpha-old",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("old digest: %v", err)
	}

	cache := map[string]cachedMCPServerSurface{
		"alpha": {
			ConfigIdentityDigest: oldDigest,
			HasToolMetadata:      true,
			ActionEntries:        []tools.CatalogEntry{stubToolSearchCatalogEntry("mcp__alpha__grep", "alpha")},
		},
	}

	if _, ok := matchingSurfaceCacheEntry(cache, "alpha", currentDigest); ok {
		t.Fatal("expected mismatched config identity to reject cached metadata reuse")
	}
	if _, ok := matchingSurfaceCacheEntry(cache, "alpha", oldDigest); !ok {
		t.Fatal("expected matching config identity to allow cached metadata reuse")
	}
}

func TestPruneDiscoveredToolsForSave_NoInstalledBundleFailOpens(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)

	records := []model.DiscoveredToolRecord{
		{CanonicalName: "mcp__alpha__write", Server: "alpha", Kind: tools.ToolKindAction, DiscoveredAt: 2},
		{CanonicalName: "", Server: "alpha", Kind: tools.ToolKindAction, DiscoveredAt: 1},
		{CanonicalName: "mcp__alpha__read", Server: "alpha", Kind: tools.ToolKindAction, DiscoveredAt: 1},
		{CanonicalName: "mcp__alpha__write", Server: "alpha", Kind: tools.ToolKindAction, DiscoveredAt: 3},
	}

	got := chat.PruneDiscoveredToolsForSave("sess-1", records)
	if len(got) != 2 {
		t.Fatalf("expected deduped records to be preserved, got %#v", got)
	}
	if got[0].CanonicalName != "mcp__alpha__read" || got[1].CanonicalName != "mcp__alpha__write" {
		t.Fatalf("expected sorted preserved discovery set, got %#v", got)
	}
}

func TestPruneDiscoveredToolsForSave_ResourceDiscoveryWithEmptyServerSurvives(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	catalog := tools.NewToolCatalog()
	resource := stubDeferredResourceEntry(tools.ReadMCPResourceName)
	if err := catalog.Register(resource); err != nil {
		t.Fatalf("register resource entry: %v", err)
	}

	chat.mu.Lock()
	chat.installedBundle = &RunnerBundle{
		Generation:   1,
		ConfigDigest: "digest",
		MCPCatalog:   catalog,
	}
	chat.mu.Unlock()

	got := chat.PruneDiscoveredToolsForSave("sess-1", []model.DiscoveredToolRecord{{
		CanonicalName: resource.CanonicalName,
		Server:        "",
		Kind:          tools.ToolKindResourceRead,
		DiscoveredAt:  1,
	}})
	if len(got) != 1 || got[0].CanonicalName != resource.CanonicalName {
		t.Fatalf("expected resource discovery to remain, got %#v", got)
	}
}

func TestPruneDiscoveredToolsForSave_KeepsRecordsWhenRuntimeMetadataShrinks(t *testing.T) {
	store := newTestConfigStore(t)
	setTestMCPServers(t, store, []config.MCPServerConfig{{
		Name:      "alpha",
		Transport: "stdio",
		Command:   "alpha",
		Enabled:   true,
	}})
	chat := NewChatService(store)
	record := model.DiscoveredToolRecord{
		CanonicalName: "mcp__alpha__write",
		Server:        "alpha",
		Kind:          tools.ToolKindAction,
		DiscoveredAt:  1,
	}

	states := []tools.MCPServerState{
		tools.MCPServerStatePending,
		tools.MCPServerStateFailed,
		tools.MCPServerStateNeedsAuth,
	}
	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			catalog := tools.NewToolCatalog()
			chat.mu.Lock()
			chat.installedBundle = &RunnerBundle{
				Generation:   1,
				ConfigDigest: "digest",
				MCPCatalog:   catalog,
				MCPHandles: []*tools.MCPServerHandle{{
					Name:  "alpha",
					State: state,
				}},
				CachedSurfaceMetadataByServer: map[string]cachedMCPServerSurface{
					"alpha": {},
				},
			}
			chat.mu.Unlock()

			got := chat.PruneDiscoveredToolsForSave("sess-1", []model.DiscoveredToolRecord{record})
			if len(got) != 1 || got[0].CanonicalName != record.CanonicalName {
				t.Fatalf("expected discovery to be retained for state %s, got %#v", state, got)
			}
		})
	}
}

func TestPruneDiscoveredToolsForSave_IgnoresMismatchedCacheForDeletion(t *testing.T) {
	store := newTestConfigStore(t)
	setTestMCPServers(t, store, []config.MCPServerConfig{{
		Name:      "alpha",
		Transport: "stdio",
		Command:   "new-alpha",
		Enabled:   true,
	}})
	chat := NewChatService(store)
	oldIdentity, err := mcpServerConfigIdentityDigest(config.MCPServerConfig{
		Name:      "alpha",
		Transport: "stdio",
		Command:   "old-alpha",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("old identity digest: %v", err)
	}

	chat.mu.Lock()
	chat.installedBundle = &RunnerBundle{
		Generation:   1,
		ConfigDigest: "digest",
		MCPCatalog:   tools.NewToolCatalog(),
		CachedSurfaceMetadataByServer: map[string]cachedMCPServerSurface{
			"alpha": {
				ConfigIdentityDigest: oldIdentity,
				HasToolMetadata:      true,
				ActionEntries:        []tools.CatalogEntry{},
			},
		},
	}
	chat.mu.Unlock()

	record := model.DiscoveredToolRecord{
		CanonicalName: "mcp__alpha__write",
		Server:        "alpha",
		Kind:          tools.ToolKindAction,
		DiscoveredAt:  1,
	}
	got := chat.PruneDiscoveredToolsForSave("sess-1", []model.DiscoveredToolRecord{record})
	if len(got) != 1 || got[0].CanonicalName != record.CanonicalName {
		t.Fatalf("expected mismatched cache to be ignored, got %#v", got)
	}
}

func TestPruneDiscoveredToolsForSave_DeletesOnlyWhenClearlyInvalid(t *testing.T) {
	store := newTestConfigStore(t)
	setTestMCPServers(t, store, []config.MCPServerConfig{
		{
			Name:      "alpha",
			Transport: "stdio",
			Command:   "alpha",
			Enabled:   true,
		},
	})
	chat := NewChatService(store)
	identity, err := mcpServerConfigIdentityDigest(config.MCPServerConfig{
		Name:      "alpha",
		Transport: "stdio",
		Command:   "alpha",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("identity digest: %v", err)
	}

	catalog := tools.NewToolCatalog()
	if err := catalog.Register(stubDeferredResourceEntry(tools.ReadMCPResourceName)); err != nil {
		t.Fatalf("register resource entry: %v", err)
	}

	chat.mu.Lock()
	chat.installedBundle = &RunnerBundle{
		Generation:   1,
		ConfigDigest: "digest",
		MCPCatalog:   catalog,
		CachedSurfaceMetadataByServer: map[string]cachedMCPServerSurface{
			"alpha": {
				ConfigIdentityDigest: identity,
				HasToolMetadata:      true,
				ActionEntries: []tools.CatalogEntry{
					stubToolSearchCatalogEntry("mcp__alpha__read", "alpha"),
				},
			},
		},
	}
	chat.mu.Unlock()

	removedServerRecord := model.DiscoveredToolRecord{CanonicalName: "mcp__beta__write", Server: "beta", Kind: tools.ToolKindAction, DiscoveredAt: 1}
	removedCanonicalRecord := model.DiscoveredToolRecord{CanonicalName: "mcp__alpha__write", Server: "alpha", Kind: tools.ToolKindAction, DiscoveredAt: 2}
	keptResourceRecord := model.DiscoveredToolRecord{CanonicalName: tools.ReadMCPResourceName, Server: "", Kind: tools.ToolKindResourceRead, DiscoveredAt: 3}
	got := chat.PruneDiscoveredToolsForSave("sess-1", []model.DiscoveredToolRecord{
		removedServerRecord,
		removedCanonicalRecord,
		keptResourceRecord,
	})
	if len(got) != 1 || got[0].CanonicalName != keptResourceRecord.CanonicalName {
		t.Fatalf("expected only clearly valid records to remain, got %#v", got)
	}
}

func TestEnsureBundleReadyForNewRun_ConfigDigestChangeRebuildsEvenWhenFingerprintMatches(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	markSessionStarting(chat, "sess-1")

	_, oldDigest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("old config snapshot: %v", err)
	}
	updateTestLLMModel(t, store, "gpt-5.4")
	_, newDigest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("new config snapshot: %v", err)
	}

	chat.mu.Lock()
	chat.installedBundle = &RunnerBundle{
		Generation:                 3,
		ConfigDigest:               oldDigest,
		DefaultRunner:              &adk.Runner{},
		PlanRunner:                 &adk.Runner{},
		LastFreshnessCheckAt:       time.Now(),
		SurfaceRelevantFingerprint: "same-fp",
	}
	chat.mu.Unlock()

	var prepareCalls int32
	chat.probeBundleSurfaceFn = func(context.Context, *config.AppConfig, map[string]cachedMCPServerSurface) (*runnerBundleSurface, error) {
		return &runnerBundleSurface{
			ActionCatalog:                 tools.NewToolCatalog(),
			CachedSurfaceMetadataByServer: map[string]cachedMCPServerSurface{},
			SurfaceRelevantFingerprint:    "same-fp",
		}, nil
	}
	chat.prepareBundleFromSurfaceFn = func(_ context.Context, _ *config.AppConfig, digest string, _ *runnerBundleSurface) (*RunnerBundle, error) {
		atomic.AddInt32(&prepareCalls, 1)
		return &RunnerBundle{
			ConfigDigest:  digest,
			DefaultRunner: &adk.Runner{},
			PlanRunner:    &adk.Runner{},
		}, nil
	}

	bundle, err := chat.ensureBundleReadyForNewRun(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("ensure bundle: %v", err)
	}
	if got := atomic.LoadInt32(&prepareCalls); got != 1 {
		t.Fatalf("expected one rebuild preparation, got %d", got)
	}
	if bundle.ConfigDigest != newDigest {
		t.Fatalf("expected rebuilt bundle digest %q, got %q", newDigest, bundle.ConfigDigest)
	}
}

func TestEnsureBundleReadyForNewRun_ConfigVersionTaskMismatchRechecksAfterWait(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	markSessionStarting(chat, "sess-old")
	markSessionStarting(chat, "sess-new")

	_, oldDigest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("old config snapshot: %v", err)
	}
	chat.mu.Lock()
	chat.installedBundle = &RunnerBundle{
		Generation:                 9,
		ConfigDigest:               oldDigest,
		DefaultRunner:              &adk.Runner{},
		PlanRunner:                 &adk.Runner{},
		LastFreshnessCheckAt:       time.Time{},
		SurfaceRelevantFingerprint: "same-fp",
	}
	chat.mu.Unlock()

	var probeCalls int32
	firstProbeEntered := make(chan struct{}, 1)
	releaseFirstProbe := make(chan struct{})
	chat.probeBundleSurfaceFn = func(context.Context, *config.AppConfig, map[string]cachedMCPServerSurface) (*runnerBundleSurface, error) {
		call := atomic.AddInt32(&probeCalls, 1)
		if call == 1 {
			firstProbeEntered <- struct{}{}
			<-releaseFirstProbe
		}
		return &runnerBundleSurface{
			ActionCatalog:                 tools.NewToolCatalog(),
			CachedSurfaceMetadataByServer: map[string]cachedMCPServerSurface{},
			SurfaceRelevantFingerprint:    "same-fp",
		}, nil
	}
	chat.prepareBundleFromSurfaceFn = func(_ context.Context, _ *config.AppConfig, digest string, _ *runnerBundleSurface) (*RunnerBundle, error) {
		return &RunnerBundle{
			ConfigDigest:  digest,
			DefaultRunner: &adk.Runner{},
			PlanRunner:    &adk.Runner{},
		}, nil
	}

	oldDone := make(chan error, 1)
	go func() {
		_, err := chat.ensureBundleReadyForNewRun(context.Background(), "sess-old")
		oldDone <- err
	}()

	<-firstProbeEntered
	updateTestLLMModel(t, store, "gpt-5.4")
	_, newDigest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("new config snapshot: %v", err)
	}

	newBundleCh := make(chan *RunnerBundle, 1)
	newErrCh := make(chan error, 1)
	go func() {
		bundle, err := chat.ensureBundleReadyForNewRun(context.Background(), "sess-new")
		newBundleCh <- bundle
		newErrCh <- err
	}()

	close(releaseFirstProbe)
	if err := <-oldDone; err != nil {
		t.Fatalf("old ensure bundle: %v", err)
	}
	newBundle := <-newBundleCh
	if err := <-newErrCh; err != nil {
		t.Fatalf("new ensure bundle: %v", err)
	}
	if newBundle == nil {
		t.Fatal("expected rebuilt bundle for new config request")
	}
	if newBundle.ConfigDigest != newDigest {
		t.Fatalf("expected new request to return config digest %q, got %q", newDigest, newBundle.ConfigDigest)
	}
	if got := atomic.LoadInt32(&probeCalls); got < 2 {
		t.Fatalf("expected a second probe after config changed, got %d", got)
	}
}

func TestEnsureBundleReadyForNewRun_PendingStartReferencePreventsEarlyCleanup(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	markSessionStarting(chat, "sess-1")

	_, digest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("config snapshot: %v", err)
	}
	oldBundle := &RunnerBundle{
		Generation:           1,
		ConfigDigest:         digest,
		DefaultRunner:        &adk.Runner{},
		PlanRunner:           &adk.Runner{},
		LastFreshnessCheckAt: time.Now(),
	}

	chat.mu.Lock()
	chat.installedBundle = oldBundle
	chat.nextGeneration = oldBundle.Generation
	chat.mu.Unlock()

	bundle, err := chat.ensureBundleReadyForNewRun(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("ensure bundle: %v", err)
	}
	if bundle != oldBundle {
		t.Fatalf("expected existing bundle, got %#v", bundle)
	}

	chat.mu.Lock()
	run := chat.sessions["sess-1"]
	if run.pendingStartBundleGeneration != oldBundle.Generation {
		chat.mu.Unlock()
		t.Fatalf("expected pending start reference for generation %d, got %d", oldBundle.Generation, run.pendingStartBundleGeneration)
	}
	chat.installedBundle = &RunnerBundle{
		Generation:           2,
		ConfigDigest:         digest,
		DefaultRunner:        &adk.Runner{},
		PlanRunner:           &adk.Runner{},
		LastFreshnessCheckAt: time.Now(),
	}
	chat.retiredBundles = []*RunnerBundle{oldBundle}
	chat.cleanupRetiredBundlesLocked()
	if len(chat.retiredBundles) != 1 {
		chat.mu.Unlock()
		t.Fatalf("expected old bundle to stay retired but referenced, got %d retired bundles", len(chat.retiredBundles))
	}
	run.running = true
	run.activeBundleGeneration = oldBundle.Generation
	run.activeRunnerKind = RunnerKindDefault
	run.pendingStartBundleGeneration = 0
	chat.cleanupRetiredBundlesLocked()
	if len(chat.retiredBundles) != 1 {
		chat.mu.Unlock()
		t.Fatalf("expected active bundle reference to keep old bundle, got %d retired bundles", len(chat.retiredBundles))
	}
	run.running = false
	run.activeBundleGeneration = 0
	run.activeRunnerKind = ""
	chat.cleanupRetiredBundlesLocked()
	if len(chat.retiredBundles) != 0 {
		chat.mu.Unlock()
		t.Fatalf("expected retired bundle to be cleaned up after references cleared, got %d", len(chat.retiredBundles))
	}
	chat.mu.Unlock()
}

func TestPendingStartReferenceIsClearedWhenStartupIsAbandonedOrSessionRemoved(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	markSessionStarting(chat, "sess-1")

	_, digest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("config snapshot: %v", err)
	}
	oldBundle := &RunnerBundle{
		Generation:           1,
		ConfigDigest:         digest,
		DefaultRunner:        &adk.Runner{},
		PlanRunner:           &adk.Runner{},
		LastFreshnessCheckAt: time.Now(),
	}

	chat.mu.Lock()
	chat.installedBundle = oldBundle
	chat.mu.Unlock()

	if _, err := chat.ensureBundleReadyForNewRun(context.Background(), "sess-1"); err != nil {
		t.Fatalf("ensure bundle: %v", err)
	}

	chat.mu.Lock()
	chat.installedBundle = &RunnerBundle{
		Generation:           2,
		ConfigDigest:         digest,
		DefaultRunner:        &adk.Runner{},
		PlanRunner:           &adk.Runner{},
		LastFreshnessCheckAt: time.Now(),
	}
	chat.retiredBundles = []*RunnerBundle{oldBundle}
	chat.cleanupRetiredBundlesLocked()
	if len(chat.retiredBundles) != 1 {
		chat.mu.Unlock()
		t.Fatalf("expected pending start reference to retain retired bundle, got %d", len(chat.retiredBundles))
	}
	sessions := chat.sessions
	if run := sessions["sess-1"]; run == nil || run.pendingStartBundleGeneration != oldBundle.Generation {
		chat.mu.Unlock()
		t.Fatalf("expected pending start reference before abandonment, got %#v", run)
	}
	chat.clearPendingStartLocked("sess-1")
	if run := chat.sessions["sess-1"]; run == nil || run.pendingStartBundleGeneration != 0 || run.starting {
		chat.mu.Unlock()
		t.Fatalf("expected pending start to be cleared after abandonment, got %#v", run)
	}
	if len(chat.retiredBundles) != 0 {
		chat.mu.Unlock()
		t.Fatalf("expected retired bundle cleanup after abandonment, got %d", len(chat.retiredBundles))
	}
	chat.mu.Unlock()

	markSessionStarting(chat, "sess-2")
	chat.mu.Lock()
	chat.installedBundle = oldBundle
	chat.mu.Unlock()
	if _, err := chat.ensureBundleReadyForNewRun(context.Background(), "sess-2"); err != nil {
		t.Fatalf("ensure bundle for deleted session path: %v", err)
	}
	chat.mu.Lock()
	chat.installedBundle = &RunnerBundle{
		Generation:           3,
		ConfigDigest:         digest,
		DefaultRunner:        &adk.Runner{},
		PlanRunner:           &adk.Runner{},
		LastFreshnessCheckAt: time.Now(),
	}
	chat.retiredBundles = []*RunnerBundle{oldBundle}
	chat.cleanupRetiredBundlesLocked()
	if len(chat.retiredBundles) != 1 {
		chat.mu.Unlock()
		t.Fatalf("expected retired bundle to stay referenced before session removal, got %d", len(chat.retiredBundles))
	}
	chat.mu.Unlock()

	chat.RemoveSession("sess-2")
	chat.mu.Lock()
	if len(chat.retiredBundles) != 0 {
		chat.mu.Unlock()
		t.Fatalf("expected session removal to trigger cleanup, got %d retired bundles", len(chat.retiredBundles))
	}
	chat.mu.Unlock()
}

func stubToolSearchCatalogEntry(canonicalName, server string) tools.CatalogEntry {
	return tools.CatalogEntry{
		CanonicalName: canonicalName,
		Server:        server,
		Source:        tools.ToolSourceMCP,
		Kind:          tools.ToolKindAction,
		ShouldDefer:   true,
		IsMcp:         true,
		PermissionSpec: tools.PermissionSpec{
			AllowSearch:  true,
			AllowExecute: true,
		},
		Tool: &stubTool{name: canonicalName},
	}
}

func stubDeferredResourceEntry(canonicalName string) tools.CatalogEntry {
	return tools.CatalogEntry{
		CanonicalName:   canonicalName,
		Source:          tools.ToolSourceMCP,
		Kind:            tools.ToolKindResourceRead,
		ShouldDefer:     true,
		IsMcp:           true,
		IsResourceTool:  true,
		ReadOnlyHint:    true,
		ReadOnlyTrusted: true,
		PermissionSpec: tools.PermissionSpec{
			AllowSearch:  true,
			AllowExecute: true,
		},
		Tool: &stubTool{name: canonicalName},
	}
}
