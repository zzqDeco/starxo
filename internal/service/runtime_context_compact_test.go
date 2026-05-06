package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"starxo/internal/model"
	"starxo/internal/tools"
)

func TestRuntimeContextCompactSnapshotAndRestorePreservesRuntimeState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(tools.ClearTodos)
	now := time.UnixMilli(1700000000000)
	chat := NewChatService(nil)
	chat.now = func() time.Time { return now }
	sessionID := "sess-compact"

	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	chat.mu.Unlock()

	for i := 0; i < 16; i++ {
		run.addUserMessage("message " + strings.Repeat("x", i+1))
	}
	run.upsertDiscoveredTool(model.DiscoveredToolRecord{
		CanonicalName: tools.RuntimeToolWebSearch,
		Kind:          tools.ToolKindAction,
		DiscoveredAt:  now.UnixMilli(),
	})
	run.setDeferredAnnouncementState(&model.DeferredAnnouncementState{
		AnnouncedSearchableCanonicalNames: []string{tools.RuntimeToolWebSearch},
	})
	run.applySyntheticDeltaStates(nil, false, &model.MCPInstructionsDeltaState{
		LastAnnouncedSearchableServers: []string{"alpha"},
	}, true)
	run.stateMu.Lock()
	run.permissionGrants[tools.RuntimeToolBash] = model.RuntimePermissionGrant{
		ToolName:  tools.RuntimeToolBash,
		Decision:  tools.ToolPermissionDecisionAllowSession,
		CreatedAt: now.UnixMilli(),
	}
	run.planDocument = &model.PlanDocument{Markdown: "ship compact", UpdatedAt: now.UnixMilli()}
	run.stateMu.Unlock()

	readResult, _ := json.Marshal(tools.ReadOutput{
		FilePath:   "/workspace/main.go",
		Content:    "package main\n",
		NumLines:   1,
		StartLine:  1,
		TotalLines: 9,
	})
	run.recordRuntimeToolResult(tools.RuntimeToolRead, `{"file_path":"main.go"}`, string(readResult), now.UnixMilli())
	editResult, _ := json.Marshal(tools.EditOutput{
		FilePath:     "/workspace/main.go",
		Replacements: 1,
		LinesAdded:   2,
		LinesRemoved: 1,
		Patch:        "-old\n+new",
	})
	run.recordRuntimeToolResult("str_replace_editor", `{"file_path":"main.go"}`, string(editResult), now.UnixMilli())

	outputPath := filepath.Join(t.TempDir(), "task.out")
	if err := os.WriteFile(outputPath, []byte("task output"), 0644); err != nil {
		t.Fatalf("write task output: %v", err)
	}
	chat.runtimeTasks.mu.Lock()
	chat.runtimeTasks.tasks["task-1"] = &runtimeTask{snapshot: tools.RuntimeTaskSnapshot{
		ID:          "task-1",
		SessionID:   sessionID,
		Type:        "bash",
		Status:      runtimeTaskStatusRunning,
		Description: "long command",
		OutputPath:  outputPath,
		StartedAt:   now.UnixMilli(),
	}}
	chat.runtimeTasks.mu.Unlock()
	chat.runtimeWorkspaces.mu.Lock()
	chat.runtimeWorkspaces.states[sessionID] = runtimeWorktreeState{
		SessionID:         sessionID,
		OriginalWorkspace: "/workspace",
		WorktreePath:      "/workspace/.starxo/worktrees/a",
		WorktreeBranch:    "starxo/a",
	}
	chat.runtimeWorkspaces.mu.Unlock()
	tools.RestoreTodos([]model.RuntimeTodoItem{{
		ID:     "todo-1",
		Title:  "compact",
		Status: "in_progress",
	}})

	snapshot, err := chat.ExportSessionSnapshot(sessionID)
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	compact := snapshot.SessionData.RuntimeContextCompact
	if compact == nil {
		t.Fatal("expected runtime compact state")
	}
	if compact.OmittedMessageCount != 4 {
		t.Fatalf("expected 4 omitted messages, got %#v", compact)
	}
	if len(compact.ToolSearch.DiscoveredTools) != 1 || compact.ToolSearch.DiscoveredTools[0].CanonicalName != tools.RuntimeToolWebSearch {
		t.Fatalf("unexpected discovered tools: %#v", compact.ToolSearch.DiscoveredTools)
	}
	if len(compact.PermissionGrants) != 1 || compact.PermissionGrants[0].ToolName != tools.RuntimeToolBash {
		t.Fatalf("unexpected grants: %#v", compact.PermissionGrants)
	}
	if len(compact.FileReadState) != 1 || compact.FileReadState[0].FilePath != "/workspace/main.go" || compact.FileReadState[0].ContentHash == "" {
		t.Fatalf("unexpected file read state: %#v", compact.FileReadState)
	}
	if len(compact.DiffSummaries) != 1 || compact.DiffSummaries[0].Replacements != 1 {
		t.Fatalf("unexpected diff summaries: %#v", compact.DiffSummaries)
	}
	if len(compact.Tasks) != 1 || compact.Tasks[0].ID != "task-1" {
		t.Fatalf("unexpected task snapshots: %#v", compact.Tasks)
	}
	if len(compact.Todos) != 1 || compact.Todos[0].Status != "in_progress" {
		t.Fatalf("unexpected todos: %#v", compact.Todos)
	}
	if compact.Workspace == nil || !compact.Workspace.Active || compact.Workspace.WorktreeBranch != "starxo/a" {
		t.Fatalf("unexpected workspace compact: %#v", compact.Workspace)
	}

	reloaded := NewChatService(nil)
	reloaded.now = func() time.Time { return now.Add(time.Second) }
	reloaded.RestoreSessionData(sessionID, snapshot.SessionData)
	reloadedRun := reloaded.GetOrCreateRun(sessionID)
	reloadedRun.stateMu.RLock()
	if len(reloadedRun.fileReadState) != 1 || len(reloadedRun.diffSummaries) != 1 || reloadedRun.runtimeContextCompact == nil {
		t.Fatalf("expected restored runtime compact internals, reads=%#v diffs=%#v compact=%#v", reloadedRun.fileReadState, reloadedRun.diffSummaries, reloadedRun.runtimeContextCompact)
	}
	reloadedRun.stateMu.RUnlock()

	tasks := reloaded.runtimeTasks.List(sessionID)
	if len(tasks) != 1 || tasks[0].Status != runtimeTaskStatusFailed {
		t.Fatalf("expected restored running task to be visible as failed, got %#v", tasks)
	}
	out, err := reloaded.ReadRuntimeTaskOutput("task-1", 0, 100)
	if err != nil {
		t.Fatalf("read restored task output: %v", err)
	}
	if out.Content != "task output" {
		t.Fatalf("unexpected restored task output: %#v", out)
	}
}

func TestPrepareMessagesForRunInjectsRuntimeCompact(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-compact-prompt"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	chat.mu.Unlock()
	for i := 0; i < 14; i++ {
		run.addUserMessage("turn")
	}
	run.upsertDiscoveredTool(model.DiscoveredToolRecord{
		CanonicalName: tools.RuntimeToolLSP,
		Kind:          tools.ToolKindAction,
	})

	messages := chat.prepareMessagesForRun(sessionID, run)
	found := false
	for _, msg := range messages {
		if strings.Contains(msg.Content, "[Runtime context compact]") && strings.Contains(msg.Content, tools.RuntimeToolLSP) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected compact message with discovered tool, got %#v", messages)
	}
}

func TestRuntimeTaskRestoreKeepsCompletedTaskStatus(t *testing.T) {
	manager := newRuntimeTaskManager(func() time.Time { return time.UnixMilli(1000) }, nil)
	manager.RestoreCompactTasks("sess", []model.RuntimeTaskCompact{{
		ID:         "done",
		SessionID:  "sess",
		Status:     runtimeTaskStatusCompleted,
		OutputPath: "/tmp/out",
		StartedAt:  1,
		FinishedAt: 2,
	}})
	tasks := manager.List("sess")
	if len(tasks) != 1 || tasks[0].Status != runtimeTaskStatusCompleted {
		t.Fatalf("expected completed task status to be preserved, got %#v", tasks)
	}
}

func TestTodoSnapshotAndRestore(t *testing.T) {
	tools.ClearTodos()
	t.Cleanup(tools.ClearTodos)
	tools.RestoreTodos([]model.RuntimeTodoItem{{
		ID:        "a",
		Title:     "A",
		Status:    "pending",
		DependsOn: []string{"root"},
	}})
	snapshot := tools.SnapshotTodos()
	if len(snapshot) != 1 || snapshot[0].DependsOn[0] != "root" {
		t.Fatalf("unexpected todo snapshot: %#v", snapshot)
	}
	snapshot[0].DependsOn[0] = "mutated"
	again := tools.SnapshotTodos()
	if again[0].DependsOn[0] != "root" {
		t.Fatalf("expected snapshot clone, got %#v", again)
	}

	tools.ClearTodos()
	if got := tools.SnapshotTodos(); len(got) != 0 {
		t.Fatalf("expected cleared todos, got %#v", got)
	}
}

func TestRuntimeToolResultFallbackUsesArguments(t *testing.T) {
	run := &SessionRun{fileReadState: make(map[string]model.RuntimeFileReadState)}
	run.recordRuntimeToolResult("read_file", `{"file_path":"relative.go"}`, `not-json`, 123)
	run.stateMu.RLock()
	defer run.stateMu.RUnlock()
	if got := run.fileReadState["relative.go"]; got.FilePath != "relative.go" || got.LastReadAt != 123 {
		t.Fatalf("expected fallback read state from args, got %#v", run.fileReadState)
	}
}

func TestRuntimeTaskRestoreIgnoresOtherSessions(t *testing.T) {
	manager := newRuntimeTaskManager(func() time.Time { return time.UnixMilli(1000) }, nil)
	manager.RestoreCompactTasks("sess-a", []model.RuntimeTaskCompact{{
		ID:        "other",
		SessionID: "sess-b",
		Status:    runtimeTaskStatusCompleted,
	}})
	if tasks := manager.List("sess-a"); len(tasks) != 0 {
		t.Fatalf("expected other session task to be ignored, got %#v", tasks)
	}
}
