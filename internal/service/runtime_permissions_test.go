package service

import (
	"context"
	"testing"

	"starxo/internal/model"
	"starxo/internal/tools"
)

func TestResolvePermissionRequestAllowsOnce(t *testing.T) {
	chat := NewChatService(nil)
	pending := &runtimePermissionRequest{
		request: tools.ToolPermissionRequest{RequestID: "perm-test", ToolName: "Bash"},
		entry:   tools.CatalogEntry{CanonicalName: "Bash"},
		result:  make(chan tools.ToolPermissionResolution, 1),
	}
	chat.permissionMu.Lock()
	chat.permissionRequests[pending.request.RequestID] = pending
	chat.permissionMu.Unlock()

	if err := chat.ApproveToolPermission("perm-test", tools.ToolPermissionDecisionAllowOnce); err != nil {
		t.Fatalf("approve permission: %v", err)
	}
	got := <-pending.result
	if got.Decision != tools.ToolPermissionDecisionAllowOnce {
		t.Fatalf("expected allow_once, got %#v", got)
	}
}

func TestRequestToolPermissionUsesSessionGrant(t *testing.T) {
	chat := NewChatService(nil)
	chat.mu.Lock()
	run := chat.getOrCreateRun("sess-perm")
	chat.mu.Unlock()
	run.stateMu.Lock()
	run.permissionGrants["Bash"] = model.RuntimePermissionGrant{
		ToolName: "Bash",
		Decision: tools.ToolPermissionDecisionAllowSession,
	}
	run.stateMu.Unlock()

	resolution, err := chat.requestToolPermission(context.Background(), "sess-perm", model.ModeDefault, tools.CatalogEntry{
		CanonicalName: "Bash",
		ToolClass:     tools.ToolClassRuntimeExec,
		Source:        tools.ToolSourceRuntime,
	}, `{"command":"go test ./..."}`)
	if err != nil {
		t.Fatalf("request permission with grant: %v", err)
	}
	if resolution.Decision != tools.ToolPermissionDecisionAllowSession {
		t.Fatalf("expected allow_session from grant, got %#v", resolution)
	}
}

func TestRequestToolPermissionWithoutUIContextFailsClosed(t *testing.T) {
	chat := NewChatService(nil)
	_, err := chat.requestToolPermission(context.Background(), "sess-perm", model.ModeDefault, tools.CatalogEntry{
		CanonicalName: "Bash",
		ToolClass:     tools.ToolClassRuntimeExec,
		Source:        tools.ToolSourceRuntime,
	}, `{"command":"rm -rf build"}`)
	if err == nil {
		t.Fatalf("expected missing UI context to fail closed")
	}
}

func TestSessionSnapshotPersistsPermissionGrants(t *testing.T) {
	chat := NewChatService(nil)
	chat.addPermissionGrant("sess-perm", tools.CatalogEntry{
		CanonicalName: "Bash",
		ToolClass:     tools.ToolClassRuntimeExec,
		Source:        tools.ToolSourceRuntime,
	})
	chat.mu.Lock()
	run := chat.sessions["sess-perm"]
	chat.mu.Unlock()
	if run == nil {
		t.Fatalf("expected session run")
	}
	snapshot := run.snapshot()
	if got := snapshot.SessionData.PermissionGrants; len(got) != 1 || got[0].ToolName != "Bash" {
		t.Fatalf("expected persisted grant, got %#v", got)
	}
}
