package service

import (
	"encoding/json"
	"strings"
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

func TestTimelineToolResultContentKeepsLargeWorktreeDiffJSONParseable(t *testing.T) {
	result := tools.WorktreeDiffOutput{
		WorkspacePath:      "/workspace",
		WorktreePath:       "/workspace/.starxo/worktrees/a",
		WorktreeBranch:     "starxo/a",
		Status:             strings.Repeat(" M file.go\n", 200),
		DiffStat:           strings.Repeat(" file.go | 1 +\n", 200),
		Diff:               strings.Repeat("+changed line\n", 5000),
		UntrackedDiff:      strings.Repeat("+untracked line\n", 1000),
		Truncated:          false,
		UntrackedTruncated: false,
		Message:            "Active worktree changes are ready for review.",
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if len(raw) <= timelineToolResultLimit(tools.RuntimeToolWorktreeDiff) {
		t.Fatalf("test input should exceed timeline limit, got %d", len(raw))
	}

	content := timelineToolResultContent(tools.RuntimeToolWorktreeDiff, string(raw))
	if len(content) > timelineToolResultLimit(tools.RuntimeToolWorktreeDiff) {
		t.Fatalf("expected content within timeline limit, got %d", len(content))
	}
	var parsed tools.WorktreeDiffOutput
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("timeline content should remain valid JSON: %v\n%s", err, content)
	}
	if parsed.WorktreeBranch != result.WorktreeBranch || parsed.Status == "" || parsed.DiffStat == "" {
		t.Fatalf("timeline content lost review metadata: %#v", parsed)
	}
	if parsed.Diff == "" || !parsed.Truncated {
		t.Fatalf("expected truncated diff to remain available, got %#v", parsed)
	}
	if parsed.UntrackedDiff != "" {
		t.Fatalf("expected duplicate untracked diff to be omitted from timeline JSON")
	}
}

func TestTimelineToolResultContentKeepsLargeEditJSONParseable(t *testing.T) {
	result := tools.EditOutput{
		FilePath:     "/workspace/main.go",
		Replacements: 1,
		LinesAdded:   500,
		LinesRemoved: 500,
		Patch:        strings.Repeat("-old line\n", 2000) + strings.Repeat("+new line\n", 2000),
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if len(raw) <= timelineToolResultLimit(tools.RuntimeToolEdit) {
		t.Fatalf("test input should exceed timeline limit, got %d", len(raw))
	}

	content := timelineToolResultContent(tools.RuntimeToolEdit, string(raw))
	if len(content) > timelineToolResultLimit(tools.RuntimeToolEdit) {
		t.Fatalf("expected content within timeline limit, got %d", len(content))
	}
	var parsed tools.EditOutput
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("timeline content should remain valid JSON: %v\n%s", err, content)
	}
	if parsed.FilePath != result.FilePath || parsed.Patch == "" || !parsed.Truncated {
		t.Fatalf("unexpected parsed edit result: %#v", parsed)
	}
}

func TestTimelineToolResultContentKeepsLargeWriteJSONParseable(t *testing.T) {
	result := tools.WriteOutput{
		FilePath:     "/workspace/main.go",
		Created:      false,
		Bytes:        50000,
		LinesAdded:   1000,
		LinesRemoved: 900,
		Patch:        strings.Repeat("-old line\n", 2000) + strings.Repeat("+new line\n", 2000),
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if len(raw) <= timelineToolResultLimit(tools.RuntimeToolWrite) {
		t.Fatalf("test input should exceed timeline limit, got %d", len(raw))
	}

	content := timelineToolResultContent(tools.RuntimeToolWrite, string(raw))
	if len(content) > timelineToolResultLimit(tools.RuntimeToolWrite) {
		t.Fatalf("expected content within timeline limit, got %d", len(content))
	}
	var parsed tools.WriteOutput
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("timeline content should remain valid JSON: %v\n%s", err, content)
	}
	if parsed.FilePath != result.FilePath || parsed.Patch == "" || !parsed.Truncated {
		t.Fatalf("unexpected parsed write result: %#v", parsed)
	}
}
