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
	requests, err := chat.ListToolPermissionRequests("")
	if err != nil {
		t.Fatalf("list permission requests: %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("expected resolved request to be removed from queue, got %#v", requests)
	}
}

func TestListToolPermissionRequestsFiltersAndSorts(t *testing.T) {
	chat := NewChatService(nil)
	chat.permissionMu.Lock()
	chat.permissionRequests["late"] = &runtimePermissionRequest{
		request: tools.ToolPermissionRequest{RequestID: "late", SessionID: "sess-a", ToolName: "Write", CreatedAt: 30},
		result:  make(chan tools.ToolPermissionResolution, 1),
	}
	chat.permissionRequests["early"] = &runtimePermissionRequest{
		request: tools.ToolPermissionRequest{RequestID: "early", SessionID: "sess-a", ToolName: "Bash", CreatedAt: 10},
		result:  make(chan tools.ToolPermissionResolution, 1),
	}
	chat.permissionRequests["other"] = &runtimePermissionRequest{
		request: tools.ToolPermissionRequest{RequestID: "other", SessionID: "sess-b", ToolName: "Edit", CreatedAt: 20},
		result:  make(chan tools.ToolPermissionResolution, 1),
	}
	chat.permissionMu.Unlock()

	requests, err := chat.ListToolPermissionRequests("sess-a")
	if err != nil {
		t.Fatalf("list permission requests: %v", err)
	}
	if len(requests) != 2 || requests[0].RequestID != "early" || requests[1].RequestID != "late" {
		t.Fatalf("expected sorted sess-a requests, got %#v", requests)
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

func TestListAndRevokeToolPermissionGrants(t *testing.T) {
	chat := NewChatService(nil)
	chat.addPermissionGrant("sess-perm", tools.CatalogEntry{
		CanonicalName: "Bash",
		ToolClass:     tools.ToolClassRuntimeExec,
		Source:        tools.ToolSourceRuntime,
	})

	grants, err := chat.ListToolPermissionGrants("sess-perm")
	if err != nil {
		t.Fatalf("list permission grants: %v", err)
	}
	if len(grants) != 1 || grants[0].ToolName != "Bash" {
		t.Fatalf("expected Bash grant, got %#v", grants)
	}

	if err := chat.RevokeToolPermissionGrant("sess-perm", "Bash"); err != nil {
		t.Fatalf("revoke permission grant: %v", err)
	}
	grants, err = chat.ListToolPermissionGrants("sess-perm")
	if err != nil {
		t.Fatalf("list permission grants after revoke: %v", err)
	}
	if len(grants) != 0 {
		t.Fatalf("expected revoked grants to be empty, got %#v", grants)
	}
}
