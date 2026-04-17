package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"starxo/internal/model"
	"starxo/internal/storage"
)

func newPlanModeStateHarness(t *testing.T) (*storage.SessionStore, *ChatService, *SessionService, *model.Session) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	sess, err := sessionStore.Create("Plan Mode State")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)
	ss.activeSession = sess
	chat.SetActiveSessionID(sess.ID)

	return sessionStore, chat, ss, sess
}

func waitForLoadedSessionData(t *testing.T, sessionStore *storage.SessionStore, sessionID string, predicate func(*model.SessionData) bool) *model.SessionData {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, err := sessionStore.LoadSessionData(sessionID)
		if err == nil && predicate(data) {
			return data
		}
		time.Sleep(20 * time.Millisecond)
	}

	data, err := sessionStore.LoadSessionData(sessionID)
	if err != nil {
		t.Fatalf("load session data: %v", err)
	}
	t.Fatalf("timed out waiting for session data predicate, last value %#v", data)
	return nil
}

func sessionDataFilePath(sessionID string) string {
	return filepath.Join(os.Getenv("HOME"), ".starxo", "sessions", sessionID, "session_data.json")
}

func readRawSessionDataFile(t *testing.T, sessionID string) *model.SessionData {
	t.Helper()
	bytes, err := os.ReadFile(sessionDataFilePath(sessionID))
	if err != nil {
		t.Fatalf("read raw session data: %v", err)
	}
	var data model.SessionData
	if err := json.Unmarshal(bytes, &data); err != nil {
		t.Fatalf("unmarshal raw session data: %v", err)
	}
	return &data
}

func TestPlanModeStateSnapshotIncludesPlanStateAndClones(t *testing.T) {
	chat := NewChatService(nil)

	chat.mu.Lock()
	run := chat.getOrCreateRun("sess-plan-state")
	chat.mu.Unlock()

	run.stateMu.Lock()
	run.mode = model.ModePlan
	run.planDocument = &model.PlanDocument{Markdown: "draft", UpdatedAt: 10}
	run.pendingPlanApproval = &model.PendingPlanApproval{RequestedAt: 20}
	run.pendingPlanAttachment = &model.PendingPlanAttachment{
		Kind:      model.PendingPlanAttachmentKindApproved,
		Markdown:  "approved plan",
		Feedback:  "ok",
		CreatedAt: 30,
	}
	run.stateMu.Unlock()

	snapshot := run.snapshot()
	if snapshot == nil || snapshot.SessionData == nil {
		t.Fatal("expected session snapshot")
	}
	if snapshot.SessionData.Version != model.SessionDataVersion {
		t.Fatalf("expected snapshot version %d, got %d", model.SessionDataVersion, snapshot.SessionData.Version)
	}
	if snapshot.SessionData.Mode != model.ModePlan {
		t.Fatalf("expected snapshot mode %q, got %q", model.ModePlan, snapshot.SessionData.Mode)
	}
	if snapshot.SessionData.PlanDocument == nil || snapshot.SessionData.PendingPlanApproval == nil || snapshot.SessionData.PendingPlanAttachment == nil {
		t.Fatalf("expected snapshot plan state, got %#v", snapshot.SessionData)
	}

	snapshot.SessionData.PlanDocument.Markdown = "mutated"
	snapshot.SessionData.PendingPlanApproval.RequestedAt = 99
	snapshot.SessionData.PendingPlanAttachment.Markdown = "mutated"

	run.stateMu.RLock()
	defer run.stateMu.RUnlock()
	if run.planDocument.Markdown != "draft" {
		t.Fatalf("expected run plan document to stay unchanged, got %#v", run.planDocument)
	}
	if run.pendingPlanApproval.RequestedAt != 20 {
		t.Fatalf("expected run pending approval to stay unchanged, got %#v", run.pendingPlanApproval)
	}
	if run.pendingPlanAttachment.Markdown != "approved plan" {
		t.Fatalf("expected run pending attachment to stay unchanged, got %#v", run.pendingPlanAttachment)
	}
}

func TestPlanModeStateRestoreSessionDataNilResetsDefaultRuntimeState(t *testing.T) {
	chat := NewChatService(nil)

	chat.RestoreSessionData("sess-nil-reset", &model.SessionData{
		Version:      model.SessionDataVersion,
		Mode:         model.ModePlan,
		PlanDocument: &model.PlanDocument{Markdown: "draft"},
		PendingPlanApproval: &model.PendingPlanApproval{
			RequestedAt: 10,
		},
		PendingPlanAttachment: &model.PendingPlanAttachment{
			Kind:     model.PendingPlanAttachmentKindRejected,
			Markdown: "draft",
		},
	})
	chat.RestoreSessionData("sess-nil-reset", nil)

	run := chat.GetOrCreateRun("sess-nil-reset")
	run.stateMu.RLock()
	defer run.stateMu.RUnlock()
	if run.mode != model.ModeDefault {
		t.Fatalf("expected default mode after nil restore, got %q", run.mode)
	}
	if run.planDocument != nil || run.pendingPlanApproval != nil || run.pendingPlanAttachment != nil {
		t.Fatalf("expected plan state to be cleared after nil restore, got %#v", run)
	}
}

func TestPlanModeStateExportSnapshotMissingSessionReturnsV4Defaults(t *testing.T) {
	chat := NewChatService(nil)

	snapshot, err := chat.ExportSessionSnapshot("missing-session")
	if err != nil {
		t.Fatalf("export missing session snapshot: %v", err)
	}
	if snapshot == nil || snapshot.SessionData == nil {
		t.Fatal("expected missing-session snapshot")
	}
	if snapshot.SessionData.Version != model.SessionDataVersion {
		t.Fatalf("expected version %d, got %d", model.SessionDataVersion, snapshot.SessionData.Version)
	}
	if snapshot.SessionData.Mode != model.ModeDefault {
		t.Fatalf("expected missing-session mode %q, got %q", model.ModeDefault, snapshot.SessionData.Mode)
	}
	if snapshot.SessionData.PlanDocument != nil || snapshot.SessionData.PendingPlanApproval != nil || snapshot.SessionData.PendingPlanAttachment != nil {
		t.Fatalf("expected nil plan state defaults, got %#v", snapshot.SessionData)
	}
}

func TestPlanModeStateExportSnapshotNormalizesOldRestoreInput(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-old-restore"

	chat.RestoreSessionData(sessionID, &model.SessionData{
		Version: 1,
		Mode:    "bogus",
		PendingPlanAttachment: &model.PendingPlanAttachment{
			Kind:     "bad",
			Markdown: "stale",
		},
	})

	snapshot, err := chat.ExportSessionSnapshot(sessionID)
	if err != nil {
		t.Fatalf("export session snapshot: %v", err)
	}
	if snapshot == nil || snapshot.SessionData == nil {
		t.Fatal("expected snapshot")
	}
	if snapshot.SessionData.Version != model.SessionDataVersion {
		t.Fatalf("expected version %d, got %d", model.SessionDataVersion, snapshot.SessionData.Version)
	}
	if snapshot.SessionData.Mode != model.ModeDefault {
		t.Fatalf("expected normalized default mode, got %q", snapshot.SessionData.Mode)
	}
	if snapshot.SessionData.PendingPlanAttachment != nil {
		t.Fatalf("expected invalid attachment to be dropped, got %#v", snapshot.SessionData.PendingPlanAttachment)
	}
}

func TestPlanModeStateSetModePersistsAfterAsyncSaveAndRestore(t *testing.T) {
	sessionStore, chat, _, sess := newPlanModeStateHarness(t)

	if err := chat.SetMode(model.ModePlan); err != nil {
		t.Fatalf("set mode: %v", err)
	}

	saved := waitForLoadedSessionData(t, sessionStore, sess.ID, func(data *model.SessionData) bool {
		return data != nil && data.Version == model.SessionDataVersion && data.Mode == model.ModePlan
	})
	if saved.Mode != model.ModePlan {
		t.Fatalf("expected saved mode %q, got %#v", model.ModePlan, saved)
	}

	reloaded := NewChatService(nil)
	reloaded.SetActiveSessionID(sess.ID)
	reloaded.RestoreSessionData(sess.ID, saved)
	if got := reloaded.GetMode(); got != model.ModePlan {
		t.Fatalf("expected restored GetMode %q, got %q", model.ModePlan, got)
	}
	_, _, mode, _ := reloaded.GetSessionRunSnapshot(sess.ID)
	if mode != model.ModePlan {
		t.Fatalf("expected restored session snapshot mode %q, got %q", model.ModePlan, mode)
	}
}

func TestPlanModeStateSetModeNoopDoesNotPersist(t *testing.T) {
	_, chat, _, sess := newPlanModeStateHarness(t)

	if err := chat.SetMode(model.ModeDefault); err != nil {
		t.Fatalf("set default mode: %v", err)
	}

	time.Sleep(250 * time.Millisecond)
	if _, err := os.Stat(sessionDataFilePath(sess.ID)); err == nil {
		t.Fatalf("expected no session_data.json write for no-op mode change")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat session_data.json: %v", err)
	}
}

func TestPlanModeStateClearHistoryClearsPlanStateWithoutChangingModeAndPersists(t *testing.T) {
	sessionStore, chat, _, sess := newPlanModeStateHarness(t)

	chat.mu.Lock()
	run := chat.getOrCreateRun(sess.ID)
	chat.mu.Unlock()

	run.stateMu.Lock()
	run.mode = model.ModePlan
	run.planDocument = &model.PlanDocument{Markdown: "draft", UpdatedAt: 10}
	run.pendingPlanApproval = &model.PendingPlanApproval{RequestedAt: 20}
	run.pendingPlanAttachment = &model.PendingPlanAttachment{
		Kind:      model.PendingPlanAttachmentKindRejected,
		Markdown:  "draft",
		Feedback:  "needs changes",
		CreatedAt: 30,
	}
	run.stateMu.Unlock()
	run.addUserMessage("hello")

	if err := chat.ClearHistory(); err != nil {
		t.Fatalf("clear history: %v", err)
	}

	if got := chat.GetMode(); got != model.ModePlan {
		t.Fatalf("expected mode to remain %q after clear history, got %q", model.ModePlan, got)
	}

	saved := waitForLoadedSessionData(t, sessionStore, sess.ID, func(data *model.SessionData) bool {
		return data != nil &&
			data.Version == model.SessionDataVersion &&
			data.Mode == model.ModePlan &&
			data.PlanDocument == nil &&
			data.PendingPlanApproval == nil &&
			data.PendingPlanAttachment == nil &&
			len(data.Messages) == 0
	})
	if saved.PlanDocument != nil || saved.PendingPlanApproval != nil || saved.PendingPlanAttachment != nil {
		t.Fatalf("expected persisted plan state to be cleared, got %#v", saved)
	}
}

func TestPlanModeStateEnsureDefaultSessionRestoresPersistedModeForStartupChain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	sess, err := sessionStore.Create("Startup Restore")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := sessionStore.SaveSessionData(sess.ID, &model.SessionData{
		Version: model.SessionDataVersion,
		Mode:    model.ModePlan,
	}); err != nil {
		t.Fatalf("save session data: %v", err)
	}

	chat := NewChatService(nil)
	ss := NewSessionService(sessionStore, nil)
	ss.SetChatService(chat)
	chat.SetSessionService(ss)

	if err := ss.EnsureDefaultSession(); err != nil {
		t.Fatalf("ensure default session: %v", err)
	}
	active := ss.GetActiveSession()
	if active == nil {
		t.Fatal("expected active session")
	}
	chat.SetActiveSessionID(active.ID)

	if got := chat.GetMode(); got != model.ModePlan {
		t.Fatalf("expected startup GetMode %q, got %q", model.ModePlan, got)
	}
	_, _, mode, _ := chat.GetSessionRunSnapshot(active.ID)
	if mode != model.ModePlan {
		t.Fatalf("expected startup session snapshot mode %q, got %q", model.ModePlan, mode)
	}
}

func TestPlanModeStateSaveSessionByIDBlockingPersistsAndRestores(t *testing.T) {
	sessionStore, chat, ss, sess := newPlanModeStateHarness(t)

	chat.mu.Lock()
	run := chat.getOrCreateRun(sess.ID)
	chat.mu.Unlock()

	run.stateMu.Lock()
	run.mode = model.ModePlan
	run.planDocument = &model.PlanDocument{Markdown: "draft", UpdatedAt: 42}
	run.pendingPlanApproval = &model.PendingPlanApproval{RequestedAt: 84}
	run.pendingPlanAttachment = &model.PendingPlanAttachment{
		Kind:      model.PendingPlanAttachmentKindApproved,
		Markdown:  "approved plan",
		CreatedAt: 126,
	}
	run.stateMu.Unlock()

	if err := ss.SaveSessionByIDBlocking(sess.ID); err != nil {
		t.Fatalf("save session by id blocking: %v", err)
	}

	raw := readRawSessionDataFile(t, sess.ID)
	if raw.Version != model.SessionDataVersion {
		t.Fatalf("expected raw version %d, got %d", model.SessionDataVersion, raw.Version)
	}
	if raw.Mode != model.ModePlan {
		t.Fatalf("expected raw mode %q, got %q", model.ModePlan, raw.Mode)
	}
	if raw.PlanDocument == nil || raw.PendingPlanApproval == nil || raw.PendingPlanAttachment == nil {
		t.Fatalf("expected raw plan state to be persisted, got %#v", raw)
	}

	loaded, err := sessionStore.LoadSessionData(sess.ID)
	if err != nil {
		t.Fatalf("load session data: %v", err)
	}
	reloaded := NewChatService(nil)
	reloaded.SetActiveSessionID(sess.ID)
	reloaded.RestoreSessionData(sess.ID, loaded)
	if got := reloaded.GetMode(); got != model.ModePlan {
		t.Fatalf("expected blocking-save restore mode %q, got %q", model.ModePlan, got)
	}
}

func TestPlanModeStateOldSessionUpgradesToV4OnModeMutation(t *testing.T) {
	sessionStore, chat, _, sess := newPlanModeStateHarness(t)

	if err := sessionStore.SaveSessionData(sess.ID, &model.SessionData{
		Version: 1,
	}); err != nil {
		t.Fatalf("save old-format session data: %v", err)
	}

	loaded, err := sessionStore.LoadSessionData(sess.ID)
	if err != nil {
		t.Fatalf("load session data: %v", err)
	}
	chat.RestoreSessionData(sess.ID, loaded)

	if err := chat.SetMode(model.ModePlan); err != nil {
		t.Fatalf("set mode: %v", err)
	}

	waitForLoadedSessionData(t, sessionStore, sess.ID, func(data *model.SessionData) bool {
		return data != nil && data.Version == model.SessionDataVersion && data.Mode == model.ModePlan
	})

	raw := readRawSessionDataFile(t, sess.ID)
	if raw.Version != model.SessionDataVersion {
		t.Fatalf("expected raw session data version %d, got %d", model.SessionDataVersion, raw.Version)
	}
	if raw.Mode != model.ModePlan {
		t.Fatalf("expected raw persisted mode %q, got %q", model.ModePlan, raw.Mode)
	}
}
