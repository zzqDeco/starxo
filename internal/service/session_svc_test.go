package service

import (
	"context"
	"strings"
	"testing"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"starxo/internal/model"
	"starxo/internal/storage"
	"starxo/internal/tools"
)

type stubTool struct {
	name string
}

func (t *stubTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name}, nil
}

func (t *stubTool) InvokableRun(context.Context, string, ...einotool.Option) (string, error) {
	return "ok", nil
}

func TestSessionServiceSwitchSessionNotifiesTargetSandboxBinding(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)

	bound, err := ss.CreateSession("Bound")
	if err != nil {
		t.Fatalf("create bound session: %v", err)
	}
	ss.BindContainer("ctr-bound", "/workspace")
	unbound, err := ss.CreateSession("Unbound")
	if err != nil {
		t.Fatalf("create unbound session: %v", err)
	}

	var switched []string
	ss.SetOnSessionSwitch(func(containerRegID string) {
		switched = append(switched, containerRegID)
	})

	if err := ss.SwitchSession(bound.ID); err != nil {
		t.Fatalf("switch to bound session: %v", err)
	}
	if err := ss.SwitchSession(unbound.ID); err != nil {
		t.Fatalf("switch to unbound session: %v", err)
	}
	if len(switched) != 2 {
		t.Fatalf("expected 2 switch callbacks, got %#v", switched)
	}
	if switched[0] != "ctr-bound" {
		t.Fatalf("expected bound session callback to target ctr-bound, got %#v", switched)
	}
	if switched[1] != "" {
		t.Fatalf("expected unbound session callback to detach sandbox, got %#v", switched)
	}
}

func TestChatServiceSendMessageRequiresBoundSandboxForActiveSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)
	chat.SetSandboxService(NewSandboxService(nil, nil))

	if _, err := ss.CreateSession("No sandbox"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	err = chat.SendMessage("create a file")
	if err == nil || !strings.Contains(err.Error(), "activate a sandbox for this session") {
		t.Fatalf("expected session-bound sandbox error, got %v", err)
	}
}

func TestChatServiceSendMessageRejectsMismatchedActiveSandbox(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)
	if _, err := ss.CreateSession("Bound sandbox"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	ss.BindContainer("ctr-bound", "/workspace")

	sandboxSvc := NewSandboxService(nil, nil)
	sandboxSvc.activeContainerRegID = "ctr-other"
	chat.SetSandboxService(sandboxSvc)

	err = chat.SendMessage("create a file")
	if err == nil || !strings.Contains(err.Error(), "active sandbox ctr-other does not match session sandbox ctr-bound") {
		t.Fatalf("expected mismatched sandbox error, got %v", err)
	}
}

func TestChatServiceResumeWithAnswerRequiresMatchingSessionSandbox(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)
	if _, err := ss.CreateSession("Pending interrupt"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	ss.BindContainer("ctr-bound", "/workspace")

	sandboxSvc := NewSandboxService(nil, nil)
	sandboxSvc.activeContainerRegID = "ctr-other"
	chat.SetSandboxService(sandboxSvc)
	chat.mu.Lock()
	run := chat.activeRun()
	run.pendingInterrupt = &PendingInterrupt{
		CheckpointID: runtimeTurnCheckpointID(run.sessionID),
		InterruptID:  "interrupt-1",
		RunnerKind:   RunnerKindDefault,
	}
	pending := run.pendingInterrupt
	chat.mu.Unlock()

	err = chat.ResumeWithAnswer("continue")
	if err == nil || !strings.Contains(err.Error(), "active sandbox ctr-other does not match session sandbox ctr-bound") {
		t.Fatalf("expected mismatched sandbox error, got %v", err)
	}
	chat.mu.Lock()
	defer chat.mu.Unlock()
	if run.pendingInterrupt != pending {
		t.Fatalf("expected pending interrupt to remain after guard failure")
	}
}

func TestChatServiceResumeWithChoiceRequiresMatchingSessionSandbox(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)
	if _, err := ss.CreateSession("Pending choice"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	ss.BindContainer("ctr-bound", "/workspace")

	sandboxSvc := NewSandboxService(nil, nil)
	sandboxSvc.activeContainerRegID = "ctr-other"
	chat.SetSandboxService(sandboxSvc)
	chat.mu.Lock()
	run := chat.activeRun()
	run.pendingInterrupt = &PendingInterrupt{
		CheckpointID: runtimeTurnCheckpointID(run.sessionID),
		InterruptID:  "interrupt-choice",
		RunnerKind:   RunnerKindDefault,
	}
	pending := run.pendingInterrupt
	chat.mu.Unlock()

	err = chat.ResumeWithChoice(0)
	if err == nil || !strings.Contains(err.Error(), "active sandbox ctr-other does not match session sandbox ctr-bound") {
		t.Fatalf("expected mismatched sandbox error, got %v", err)
	}
	chat.mu.Lock()
	defer chat.mu.Unlock()
	if run.pendingInterrupt != pending {
		t.Fatalf("expected pending interrupt to remain after guard failure")
	}
}

func TestChatServiceWorkspaceChangeContainerUsesSessionBinding(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)
	sess, err := ss.CreateSession("Bound session")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	ss.BindContainer("ctr-bound", "/workspace")

	sandboxSvc := NewSandboxService(nil, nil)
	sandboxSvc.activeContainerRegID = "ctr-other"
	chat.SetSandboxService(sandboxSvc)

	if got := chat.activeWorkspaceContainerID(sess.ID); got != "ctr-bound" {
		t.Fatalf("expected session-bound container id, got %q", got)
	}
}

func TestChatServiceStopRunsNotBoundToSandboxCancelsOnlyMismatchedSessions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)

	sessionA, err := ss.CreateSession("Session A")
	if err != nil {
		t.Fatalf("create session A: %v", err)
	}
	ss.BindContainer("ctr-a", "/workspace-a")
	sessionB, err := ss.CreateSession("Session B")
	if err != nil {
		t.Fatalf("create session B: %v", err)
	}
	ss.BindContainer("ctr-b", "/workspace-b")

	cancelledA := false
	cancelledB := false
	chat.mu.Lock()
	runA := chat.getOrCreateRun(sessionA.ID)
	runA.running = true
	runA.cancelFn = func() { cancelledA = true }
	runB := chat.getOrCreateRun(sessionB.ID)
	runB.running = true
	runB.cancelFn = func() { cancelledB = true }
	chat.mu.Unlock()

	stopped, err := chat.StopRunsNotBoundToSandbox("ctr-a")
	if err != nil {
		t.Fatalf("stop mismatched runs: %v", err)
	}
	if len(stopped) != 1 || stopped[0] != sessionB.ID {
		t.Fatalf("expected only session B to stop, got %#v", stopped)
	}
	if cancelledA {
		t.Fatal("session A should keep running for the target sandbox")
	}
	if !cancelledB {
		t.Fatal("session B should be canceled before switching to ctr-a")
	}
}

func TestChatServiceStopRunsNotBoundToSandboxCancelsAllForUnboundTarget(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)

	sessionA, err := ss.CreateSession("Session A")
	if err != nil {
		t.Fatalf("create session A: %v", err)
	}
	ss.BindContainer("ctr-a", "/workspace-a")
	sessionB, err := ss.CreateSession("Session B")
	if err != nil {
		t.Fatalf("create session B: %v", err)
	}
	ss.BindContainer("ctr-b", "/workspace-b")

	cancelled := map[string]bool{}
	chat.mu.Lock()
	runA := chat.getOrCreateRun(sessionA.ID)
	runA.running = true
	runA.cancelFn = func() { cancelled[sessionA.ID] = true }
	runB := chat.getOrCreateRun(sessionB.ID)
	runB.running = true
	runB.cancelFn = func() { cancelled[sessionB.ID] = true }
	chat.mu.Unlock()

	stopped, err := chat.StopRunsNotBoundToSandbox("")
	if err != nil {
		t.Fatalf("stop runs for unbound target: %v", err)
	}
	if len(stopped) != 2 {
		t.Fatalf("expected both sessions to stop, got %#v", stopped)
	}
	if !cancelled[sessionA.ID] || !cancelled[sessionB.ID] {
		t.Fatalf("expected both sessions to be canceled, got %#v", cancelled)
	}
}

func TestFileServiceRejectsUnboundSessionWorkspaceAccess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	ss := NewSessionService(sessionStore, nil)
	if _, err := ss.CreateSession("No sandbox"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	sandboxSvc := NewSandboxService(nil, nil)
	sandboxSvc.activeContainerRegID = "ctr-old"
	fileSvc := NewFileService(sandboxSvc)
	fileSvc.SetSessionService(ss)

	_, err = fileSvc.ensureActiveSessionSandbox()
	if err == nil || !strings.Contains(err.Error(), "activate a sandbox for this session") {
		t.Fatalf("expected unbound session workspace error, got %v", err)
	}
}

func TestSessionServiceSaveSessionByIDPreservesDeferredDiscoveryAcrossModes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	sess, err := sessionStore.Create("Deferred MCP")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)

	catalog := tools.NewToolCatalog()
	readonly := tools.CatalogEntry{
		CanonicalName:   "mcp__alpha__readonly",
		Source:          tools.ToolSourceMCP,
		Server:          "alpha",
		Kind:            tools.ToolKindAction,
		ShouldDefer:     true,
		IsMcp:           true,
		ReadOnlyHint:    true,
		ReadOnlyTrusted: true,
		PermissionSpec: tools.PermissionSpec{
			AllowSearch:  true,
			AllowExecute: true,
		},
		Tool: &stubTool{name: "mcp__alpha__readonly"},
	}
	readwrite := readonly
	readwrite.CanonicalName = "mcp__alpha__write"
	readwrite.ReadOnlyHint = false
	readwrite.ReadOnlyTrusted = false
	readwrite.Tool = &stubTool{name: "mcp__alpha__write"}

	for _, entry := range []tools.CatalogEntry{readonly, readwrite} {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register %s: %v", entry.CanonicalName, err)
		}
	}

	chat.mu.Lock()
	run := chat.getOrCreateRun(sess.ID)
	run.mode = "plan"
	chat.installedBundle = &RunnerBundle{
		Generation:   1,
		ConfigDigest: "test",
		MCPCatalog:   catalog,
		MCPHandles: []*tools.MCPServerHandle{{
			Name:              "alpha",
			State:             tools.MCPServerStateConnected,
			ToolMetadataReady: true,
		}},
	}
	chat.mu.Unlock()

	run.addUserMessage("need readonly mcp")
	run.upsertDiscoveredTool(model.DiscoveredToolRecord{
		CanonicalName: readonly.CanonicalName,
		Server:        "alpha",
		Kind:          tools.ToolKindAction,
		DiscoveredAt:  1,
	})
	run.upsertDiscoveredTool(model.DiscoveredToolRecord{
		CanonicalName: readwrite.CanonicalName,
		Server:        "alpha",
		Kind:          tools.ToolKindAction,
		DiscoveredAt:  2,
	})

	if err := ss.SaveSessionByID(sess.ID); err != nil {
		t.Fatalf("save session by id: %v", err)
	}

	var saved *model.SessionData
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		saved, err = sessionStore.LoadSessionData(sess.ID)
		if err == nil && saved != nil && len(saved.DiscoveredTools) == 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("load session data: %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved session data")
	}
	if len(saved.DiscoveredTools) != 2 {
		t.Fatalf("unexpected saved discovery set: %#v", saved.DiscoveredTools)
	}
	if saved.DiscoveredTools[0].CanonicalName != readonly.CanonicalName || saved.DiscoveredTools[1].CanonicalName != readwrite.CanonicalName {
		t.Fatalf("unexpected saved discovery ordering: %#v", saved.DiscoveredTools)
	}

	memory := run.discoveredToolsSnapshot()
	if len(memory) != 2 {
		t.Fatalf("expected in-memory discovery set to be preserved, got %#v", memory)
	}
	if _, ok := memory[readonly.CanonicalName]; !ok {
		t.Fatalf("expected readonly discovery to remain, got %#v", memory)
	}
	if _, ok := memory[readwrite.CanonicalName]; !ok {
		t.Fatalf("expected readwrite discovery to remain, got %#v", memory)
	}
}

func TestSessionServiceSaveSessionByIDPersistsDeferredAnnouncementState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	sess, err := sessionStore.Create("Deferred Delta")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)

	chat.mu.Lock()
	run := chat.getOrCreateRun(sess.ID)
	chat.mu.Unlock()

	run.addUserMessage("hello")
	run.upsertDiscoveredTool(model.DiscoveredToolRecord{
		CanonicalName: "mcp__alpha__grep",
		Server:        "alpha",
		Kind:          tools.ToolKindAction,
		DiscoveredAt:  1,
	})
	run.setDeferredAnnouncementState(&model.DeferredAnnouncementState{
		AnnouncedSearchableCanonicalNames: []string{"mcp__beta__status", "mcp__alpha__grep"},
	})

	if err := ss.SaveSessionByID(sess.ID); err != nil {
		t.Fatalf("save session by id: %v", err)
	}

	var saved *model.SessionData
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		saved, err = sessionStore.LoadSessionData(sess.ID)
		if err == nil && saved != nil && saved.DeferredAnnouncementState != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("load session data: %v", err)
	}
	if saved == nil || saved.DeferredAnnouncementState == nil {
		t.Fatal("expected deferred announcement state to be saved")
	}
	got := saved.DeferredAnnouncementState.AnnouncedSearchableCanonicalNames
	want := []string{"mcp__beta__status", "mcp__alpha__grep"}
	if len(got) != len(want) {
		t.Fatalf("unexpected announcement state: %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected announcement state ordering/value: got %#v want %#v", got, want)
		}
	}
	if len(saved.DiscoveredTools) != 1 || saved.DiscoveredTools[0].CanonicalName != "mcp__alpha__grep" {
		t.Fatalf("unexpected discovered tools snapshot: %#v", saved.DiscoveredTools)
	}
	if len(saved.Messages) != 1 || saved.Messages[0].Content != "hello" {
		t.Fatalf("unexpected message snapshot: %#v", saved.Messages)
	}
}

func TestSessionServiceSaveSessionByIDPersistsMCPInstructionsDeltaState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	sess, err := sessionStore.Create("MCP Instructions")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)

	chat.mu.Lock()
	run := chat.getOrCreateRun(sess.ID)
	chat.mu.Unlock()

	run.applySyntheticDeltaStates(nil, false, &model.MCPInstructionsDeltaState{
		LastAnnouncedSearchableServers:  []string{"alpha"},
		LastAnnouncedPendingServers:     []string{"beta"},
		LastAnnouncedUnavailableServers: []string{"gamma:failed"},
		LastInstructionsFingerprint:     tools.ComputeMCPInstructionsFingerprint([]string{"alpha"}, []string{"beta"}, []string{"gamma:failed"}),
	}, true)

	if err := ss.SaveSessionByID(sess.ID); err != nil {
		t.Fatalf("save session by id: %v", err)
	}

	var saved *model.SessionData
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		saved, err = sessionStore.LoadSessionData(sess.ID)
		if err == nil && saved != nil && saved.MCPInstructionsDeltaState != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("load session data: %v", err)
	}
	if saved == nil || saved.MCPInstructionsDeltaState == nil {
		t.Fatal("expected MCP instructions delta state to be saved")
	}
	if got := saved.MCPInstructionsDeltaState.LastAnnouncedSearchableServers; len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("unexpected searchable servers: %#v", got)
	}
	if got := saved.MCPInstructionsDeltaState.LastAnnouncedPendingServers; len(got) != 1 || got[0] != "beta" {
		t.Fatalf("unexpected pending servers: %#v", got)
	}
	if got := saved.MCPInstructionsDeltaState.LastAnnouncedUnavailableServers; len(got) != 1 || got[0] != "gamma:failed" {
		t.Fatalf("unexpected unavailable servers: %#v", got)
	}
	if saved.MCPInstructionsDeltaState.LastInstructionsFingerprint == "" {
		t.Fatal("expected persisted fingerprint")
	}
}

func TestSessionServiceSwitchRestoresSessionScopedTodos(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	sessionA, err := sessionStore.Create("Session A")
	if err != nil {
		t.Fatalf("create session-a: %v", err)
	}
	sessionB, err := sessionStore.Create("Session B")
	if err != nil {
		t.Fatalf("create session-b: %v", err)
	}
	t.Cleanup(func() {
		tools.ClearTodos()
		tools.ClearTodosForSession(sessionA.ID)
		tools.ClearTodosForSession(sessionB.ID)
	})

	dataA := model.DefaultSessionData()
	dataA.RuntimeContextCompact = &model.RuntimeContextCompact{
		Version: model.RuntimeContextCompactVersion,
		Todos: []model.RuntimeTodoItem{{
			ID:     "todo-a",
			Title:  "A",
			Status: "pending",
		}},
	}
	if err := sessionStore.SaveSessionData(sessionA.ID, dataA); err != nil {
		t.Fatalf("save session-a data: %v", err)
	}
	dataB := model.DefaultSessionData()
	dataB.RuntimeContextCompact = &model.RuntimeContextCompact{
		Version: model.RuntimeContextCompactVersion,
		Todos: []model.RuntimeTodoItem{{
			ID:     "todo-b",
			Title:  "B",
			Status: "done",
		}},
	}
	if err := sessionStore.SaveSessionData(sessionB.ID, dataB); err != nil {
		t.Fatalf("save session-b data: %v", err)
	}

	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)

	if err := ss.SwitchSession(sessionA.ID); err != nil {
		t.Fatalf("switch to session-a: %v", err)
	}
	todosA := tools.SnapshotTodosForSession(sessionA.ID)
	if len(todosA) != 1 || todosA[0].ID != "todo-a" {
		t.Fatalf("unexpected session-a todos after switch: %#v", todosA)
	}
	if got := tools.SnapshotTodosForSession(sessionB.ID); len(got) != 0 {
		t.Fatalf("expected session-b todos to remain empty before switch, got %#v", got)
	}

	if err := ss.SwitchSession(sessionB.ID); err != nil {
		t.Fatalf("switch to session-b: %v", err)
	}
	todosB := tools.SnapshotTodosForSession(sessionB.ID)
	if len(todosB) != 1 || todosB[0].ID != "todo-b" {
		t.Fatalf("unexpected session-b todos after switch: %#v", todosB)
	}
	if got := tools.SnapshotTodos(); len(got) != 0 {
		t.Fatalf("expected global compatibility todos to remain empty, got %#v", got)
	}
}
