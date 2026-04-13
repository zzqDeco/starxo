package service

import (
	"context"
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
