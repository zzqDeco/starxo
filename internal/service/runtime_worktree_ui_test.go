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

func TestRuntimeToolWorkspaceChangePath(t *testing.T) {
	writeRaw, err := json.Marshal(tools.WriteOutput{FilePath: "/workspace/a.txt"})
	if err != nil {
		t.Fatalf("marshal write: %v", err)
	}
	path, ok := runtimeToolWorkspaceChangePath(tools.RuntimeToolWrite, "", string(writeRaw))
	if !ok || path != "/workspace/a.txt" {
		t.Fatalf("expected write workspace change path, got path=%q ok=%v", path, ok)
	}

	legacyWriteRaw, err := json.Marshal(tools.WriteFileOutput{Success: true})
	if err != nil {
		t.Fatalf("marshal legacy write: %v", err)
	}
	path, ok = runtimeToolWorkspaceChangePath("write_file", `{"path":"/workspace/legacy.txt"}`, string(legacyWriteRaw))
	if !ok || path != "/workspace/legacy.txt" {
		t.Fatalf("expected legacy write to use args path, got path=%q ok=%v", path, ok)
	}

	path, ok = runtimeToolWorkspaceChangePath("str_replace_editor", `{"command":"str_replace","path":"/workspace/edit.txt"}`, "File has been edited")
	if !ok || path != "/workspace/edit.txt" {
		t.Fatalf("expected legacy str_replace_editor to use args path, got path=%q ok=%v", path, ok)
	}
	if path, ok = runtimeToolWorkspaceChangePath("str_replace_editor", `{"command":"view","path":"/workspace/edit.txt"}`, "file contents"); ok || path != "" {
		t.Fatalf("expected str_replace view not to refresh, got path=%q ok=%v", path, ok)
	}

	bashRaw, err := json.Marshal(tools.BashOutput{ExitCode: 0})
	if err != nil {
		t.Fatalf("marshal bash: %v", err)
	}
	path, ok = runtimeToolWorkspaceChangePath(tools.RuntimeToolBash, `{"command":"echo ok > a.txt"}`, string(bashRaw))
	if !ok || path != "" {
		t.Fatalf("expected successful mutating bash to refresh workspace without path, got path=%q ok=%v", path, ok)
	}
	if path, ok = runtimeToolWorkspaceChangePath(tools.RuntimeToolBash, `{"command":"pwd"}`, string(bashRaw)); ok || path != "" {
		t.Fatalf("expected read-only bash not to refresh workspace, got path=%q ok=%v", path, ok)
	}

	failedBashRaw, err := json.Marshal(tools.BashOutput{ExitCode: 1})
	if err != nil {
		t.Fatalf("marshal failed bash: %v", err)
	}
	if path, ok = runtimeToolWorkspaceChangePath(tools.RuntimeToolBash, `{"command":"touch a.txt"}`, string(failedBashRaw)); ok || path != "" {
		t.Fatalf("expected failed bash not to emit workspace change, got path=%q ok=%v", path, ok)
	}
	if path, ok = runtimeToolWorkspaceChangePath(tools.RuntimeToolWrite, "", "recoverable write error"); ok || path != "" {
		t.Fatalf("expected unparseable write result not to emit workspace change, got path=%q ok=%v", path, ok)
	}

	lspRaw, err := json.Marshal(tools.LSPEditOutput{EditCount: 2, ChangedFiles: []string{"/workspace/a.go", "/workspace/b.go"}})
	if err != nil {
		t.Fatalf("marshal lsp edit: %v", err)
	}
	path, ok = runtimeToolWorkspaceChangePath(tools.RuntimeToolLSPEdit, "", string(lspRaw))
	if !ok || path != "" {
		t.Fatalf("expected multi-file lsp edit to refresh broadly, got path=%q ok=%v", path, ok)
	}

	notebookRaw, err := json.Marshal(tools.NotebookEditOutput{FilePath: "/workspace/a.ipynb", Command: "view"})
	if err != nil {
		t.Fatalf("marshal notebook: %v", err)
	}
	if path, ok = runtimeToolWorkspaceChangePath(tools.RuntimeToolNotebookEdit, "", string(notebookRaw)); ok || path != "/workspace/a.ipynb" {
		t.Fatalf("expected notebook view not to refresh, got path=%q ok=%v", path, ok)
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

func TestTimelineToolResultContentKeepsWorktreeMergeConflictJSONParseable(t *testing.T) {
	result := tools.WorktreeMergeOutput{
		Action:         "merge_conflict",
		WorkspacePath:  "/workspace",
		WorktreePath:   "/workspace/.starxo/worktrees/a",
		WorktreeBranch: "starxo/a",
		CommitMessage:  "merge a",
		Conflicted:     true,
		ConflictFiles:  []string{"main.go", "pkg/app.go"},
		MergeOutput:    strings.Repeat("CONFLICT (content): Merge conflict in main.go\n", 300),
		RecoveryHint:   "Parent merge was aborted and the active worktree was preserved.",
		Message:        "Worktree merge hit conflicts; parent merge was aborted and the worktree remains active.",
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if len(raw) <= timelineToolResultLimit(tools.RuntimeToolWorktreeMerge) {
		t.Fatalf("test input should exceed timeline limit, got %d", len(raw))
	}

	content := timelineToolResultContent(tools.RuntimeToolWorktreeMerge, string(raw))
	if len(content) > timelineToolResultLimit(tools.RuntimeToolWorktreeMerge) {
		t.Fatalf("expected content within timeline limit, got %d", len(content))
	}
	var parsed tools.WorktreeMergeOutput
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("timeline content should remain valid JSON: %v\n%s", err, content)
	}
	if !parsed.Conflicted || parsed.Action != "merge_conflict" || len(parsed.ConflictFiles) != 2 || parsed.MergeOutput == "" {
		t.Fatalf("timeline content lost conflict metadata: %#v", parsed)
	}
	if !strings.Contains(parsed.MergeOutput, "truncated for timeline") {
		t.Fatalf("expected merge output to be JSON-aware truncated, got %q", parsed.MergeOutput)
	}
}

func TestTimelineToolResultContentCapsLongWorktreeMergeConflictPaths(t *testing.T) {
	files := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		files = append(files, "very/long/"+strings.Repeat("nested-directory/", 20)+"file.go")
	}
	result := tools.WorktreeMergeOutput{
		Action:         "merge_conflict",
		WorkspacePath:  "/workspace",
		WorktreePath:   "/workspace/.starxo/worktrees/a",
		WorktreeBranch: "starxo/a",
		CommitMessage:  "merge a",
		Conflicted:     true,
		ConflictFiles:  files,
		RecoveryHint:   "Parent merge was aborted and the active worktree was preserved.",
		Message:        "Worktree merge hit conflicts; parent merge was aborted and the worktree remains active.",
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if len(raw) <= timelineToolResultLimit(tools.RuntimeToolWorktreeMerge) {
		t.Fatalf("test input should exceed timeline limit, got %d", len(raw))
	}

	content := timelineToolResultContent(tools.RuntimeToolWorktreeMerge, string(raw))
	if len(content) > timelineToolResultLimit(tools.RuntimeToolWorktreeMerge) {
		t.Fatalf("expected content within timeline limit, got %d", len(content))
	}
	var parsed tools.WorktreeMergeOutput
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("timeline content should remain valid JSON: %v\n%s", err, content)
	}
	if !parsed.Conflicted || len(parsed.ConflictFiles) == 0 || !strings.Contains(parsed.ConflictFiles[0], "...") {
		t.Fatalf("expected conflict metadata with capped paths, got %#v", parsed)
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
