package service

import (
	"testing"

	"starxo/internal/tools"
)

func TestGetRuntimeWorktreeStateUsesActiveSession(t *testing.T) {
	chat := NewChatService(nil)
	chat.SetActiveSessionID("sess-a")
	chat.runtimeWorkspaces.mu.Lock()
	chat.runtimeWorkspaces.states["sess-a"] = runtimeWorktreeState{
		SessionID:         "sess-a",
		OriginalWorkspace: "/workspace",
		WorktreePath:      "/workspace/.starxo/worktrees/a",
		WorktreeBranch:    "starxo/a",
	}
	chat.runtimeWorkspaces.mu.Unlock()

	state, err := chat.GetRuntimeWorktreeState("")
	if err != nil {
		t.Fatalf("get worktree state: %v", err)
	}
	if !state.Active || state.SessionID != "sess-a" || state.CurrentPath != "/workspace/.starxo/worktrees/a" || state.WorktreeBranch != "starxo/a" {
		t.Fatalf("unexpected worktree state: %#v", state)
	}
}

func TestFileServiceWorkspacePathFollowsRuntimeWorktree(t *testing.T) {
	chat := NewChatService(nil)
	chat.SetActiveSessionID("sess-a")
	chat.runtimeWorkspaces.mu.Lock()
	chat.runtimeWorkspaces.states["sess-a"] = runtimeWorktreeState{
		SessionID:         "sess-a",
		OriginalWorkspace: "/workspace",
		WorktreePath:      "/workspace/.starxo/worktrees/a",
		WorktreeBranch:    "starxo/a",
	}
	chat.runtimeWorkspaces.mu.Unlock()
	files := NewFileService(nil, chat)

	if got := files.workspacePath(); got != "/workspace/.starxo/worktrees/a" {
		t.Fatalf("expected file service to follow active worktree, got %q", got)
	}
}

func TestTimelineToolResultLimitKeepsWorktreeDiffReview(t *testing.T) {
	if got := timelineToolResultLimit(tools.RuntimeToolWorktreeDiff); got < 20000 {
		t.Fatalf("expected large worktree diff timeline limit, got %d", got)
	}
	if got := timelineToolResultLimit(tools.RuntimeToolBash); got != 1000 {
		t.Fatalf("expected default timeline limit, got %d", got)
	}
}
