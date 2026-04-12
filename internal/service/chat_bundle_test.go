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
		wg.Add(1)
		go func() {
			defer wg.Done()
			bundle, err := chat.ensureBundleReadyForNewRun(context.Background())
			if err != nil {
				errs <- err
				return
			}
			if bundle == nil || bundle.Generation != 7 {
				errs <- fmt.Errorf("unexpected bundle result: %#v", bundle)
			}
		}()
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
	newBundle := &RunnerBundle{
		Generation:                 2,
		ConfigDigest:               digest,
		DefaultRunner:              &adk.Runner{},
		PlanRunner:                 &adk.Runner{},
		LastFreshnessCheckAt:       time.Time{},
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
		_, err := chat.ensureBundleReadyForNewRun(context.Background())
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
	if !newBundle.LastFreshnessCheckAt.IsZero() {
		t.Fatalf("expected stale probe result to be dropped, got freshness timestamp %v", newBundle.LastFreshnessCheckAt)
	}
}

func TestEnsureBundleReadyForNewRunProbeFailureKeepsCurrentBundleStale(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)

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

	got, err := chat.ensureBundleReadyForNewRun(context.Background())
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
					HasToolMetadata: true,
					ActionEntries:   []tools.CatalogEntry{entry},
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
