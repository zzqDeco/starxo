package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
)

type fakeRuntimeOperator struct {
	files map[string]string
}

func (o *fakeRuntimeOperator) ReadFile(ctx context.Context, path string) (string, error) {
	if o.files == nil {
		o.files = make(map[string]string)
	}
	content, ok := o.files[path]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return content, nil
}

func (o *fakeRuntimeOperator) WriteFile(ctx context.Context, path string, content string) error {
	if o.files == nil {
		o.files = make(map[string]string)
	}
	o.files[path] = content
	return nil
}

func (o *fakeRuntimeOperator) IsDirectory(ctx context.Context, path string) (bool, error) {
	return false, nil
}

func (o *fakeRuntimeOperator) Exists(ctx context.Context, path string) (bool, error) {
	_, ok := o.files[path]
	return ok, nil
}

func (o *fakeRuntimeOperator) RunCommand(ctx context.Context, command []string) (*commandline.CommandOutput, error) {
	return &commandline.CommandOutput{Stdout: "", Stderr: "", ExitCode: 0}, nil
}

type fakeRuntimeTaskManager struct {
	persistedPath string
	persistedSize int64
}

type fakeWorkspaceManager struct {
	workspace string
}

func (m fakeWorkspaceManager) CurrentWorkspace(ctx context.Context, defaultWorkspace string) string {
	if m.workspace == "" {
		return defaultWorkspace
	}
	return m.workspace
}

func (m fakeWorkspaceManager) EnterWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, name string) (WorktreeOutput, error) {
	return WorktreeOutput{}, fmt.Errorf("not implemented")
}

func (m fakeWorkspaceManager) ExitWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, action string, discardChanges bool) (WorktreeOutput, error) {
	return WorktreeOutput{}, fmt.Errorf("not implemented")
}

func (m fakeWorkspaceManager) DiffWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace string, includePatch bool, maxBytes int) (WorktreeDiffOutput, error) {
	return WorktreeDiffOutput{}, fmt.Errorf("not implemented")
}

func (m fakeWorkspaceManager) MergeWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace string, commitMessage string, removeWorktree bool) (WorktreeMergeOutput, error) {
	return WorktreeMergeOutput{}, fmt.Errorf("not implemented")
}

func (m *fakeRuntimeTaskManager) StartShellTask(ctx context.Context, sessionID, command, description string, runner RuntimeTaskRunner) (RuntimeTaskRef, error) {
	return RuntimeTaskRef{}, fmt.Errorf("not implemented")
}

func (m *fakeRuntimeTaskManager) ReadTaskOutput(ctx context.Context, taskID string, offset, limit int) (RuntimeTaskOutput, error) {
	return RuntimeTaskOutput{}, fmt.Errorf("not implemented")
}

func (m *fakeRuntimeTaskManager) StopTask(ctx context.Context, taskID string) (RuntimeTaskSnapshot, error) {
	return RuntimeTaskSnapshot{}, fmt.Errorf("not implemented")
}

func (m *fakeRuntimeTaskManager) PersistToolResult(ctx context.Context, sessionID, prefix, content string) (string, int64, error) {
	m.persistedPath = "/tmp/starxo-result.txt"
	m.persistedSize = int64(len(content))
	return m.persistedPath, m.persistedSize, nil
}

func TestRuntimeCoreCatalogEntriesExposeAliasesAndPlanGate(t *testing.T) {
	entries, err := NewRuntimeCoreCatalogEntries(&fakeRuntimeOperator{}, "/workspace", nil, nil)
	if err != nil {
		t.Fatalf("runtime core entries: %v", err)
	}
	catalog := NewToolCatalog()
	for _, entry := range entries {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register %s: %v", entry.CanonicalName, err)
		}
	}

	for alias, canonical := range map[string]string{
		"shell_execute":      RuntimeToolBash,
		"read_file":          RuntimeToolRead,
		"write_file":         RuntimeToolWrite,
		"str_replace_editor": RuntimeToolEdit,
		"list_files":         RuntimeToolGlob,
	} {
		entry, ok := catalog.LookupExact(alias)
		if !ok || entry.CanonicalName != canonical {
			t.Fatalf("expected alias %s -> %s, got %#v", alias, canonical, entry)
		}
	}

	state := ComputeDeferredMCPState(catalog, nil, ToolPermissionContext{Mode: "plan"})
	assertCatalogNames(t, state.CurrentLoadedTools, []string{
		RuntimeToolExitPlanMode,
		RuntimeToolGlob,
		RuntimeToolGrep,
		RuntimeToolRead,
		RuntimeToolTaskOutput,
	})
	assertCatalogNames(t, state.SearchablePoolForMode, []string{
		RuntimeToolWorktreeDiff,
	})

	byName := map[string]CatalogEntry{}
	for _, entry := range entries {
		byName[entry.CanonicalName] = entry
	}
	if diff := byName[RuntimeToolWorktreeDiff]; !diff.ShouldDefer || !diff.ReadOnlyTrusted {
		t.Fatalf("expected WorktreeDiff to be read-only deferred, got %#v", diff)
	}
	if merge := byName[RuntimeToolWorktreeMerge]; !merge.ShouldDefer || merge.ReadOnlyTrusted {
		t.Fatalf("expected WorktreeMerge to require write permission, got %#v", merge)
	}
}

func TestRuntimeEditToolUpdatesFileAndPatch(t *testing.T) {
	op := &fakeRuntimeOperator{files: map[string]string{"/workspace/main.go": "before\nold\n"}}
	entries, err := NewRuntimeCoreCatalogEntries(op, "/workspace", nil, nil)
	if err != nil {
		t.Fatalf("runtime core entries: %v", err)
	}
	var edit CatalogEntry
	for _, entry := range entries {
		if entry.CanonicalName == RuntimeToolEdit {
			edit = entry
			break
		}
	}
	if edit.Tool == nil {
		t.Fatalf("edit tool not found")
	}
	invokable, ok := edit.Tool.(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	if !ok {
		t.Fatalf("edit tool is not invokable: %T", edit.Tool)
	}
	payload, _ := json.Marshal(EditInput{FilePath: "main.go", OldString: "old", NewString: "new"})
	result, err := invokable.InvokableRun(context.Background(), string(payload))
	if err != nil {
		t.Fatalf("edit tool run: %v", err)
	}
	if got := op.files["/workspace/main.go"]; got != "before\nnew\n" {
		t.Fatalf("expected edited file, got %q", got)
	}
	if !strings.Contains(result, `"replacements":1`) || !strings.Contains(result, "-old") || !strings.Contains(result, "+new") {
		t.Fatalf("expected structured edit result with patch, got %s", result)
	}
}

func TestRuntimeReadToolUsesCurrentWorkspace(t *testing.T) {
	op := &fakeRuntimeOperator{files: map[string]string{"/workspace/.starxo/worktrees/feat/main.go": "package main\n"}}
	entries, err := NewRuntimeCoreCatalogEntries(op, "/workspace", nil, fakeWorkspaceManager{workspace: "/workspace/.starxo/worktrees/feat"})
	if err != nil {
		t.Fatalf("runtime core entries: %v", err)
	}
	var read CatalogEntry
	for _, entry := range entries {
		if entry.CanonicalName == RuntimeToolRead {
			read = entry
			break
		}
	}
	invokable, ok := read.Tool.(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	if !ok {
		t.Fatalf("read tool is not invokable: %T", read.Tool)
	}
	result, err := invokable.InvokableRun(context.Background(), `{"file_path":"main.go"}`)
	if err != nil {
		t.Fatalf("read tool run: %v", err)
	}
	if !strings.Contains(result, "/workspace/.starxo/worktrees/feat/main.go") || !strings.Contains(result, "package main") {
		t.Fatalf("expected read from worktree workspace, got %s", result)
	}
}

func TestWorkspaceFilePathUsesActiveWorktreeBoundary(t *testing.T) {
	manager := fakeWorkspaceManager{workspace: "/repo/.starxo/worktrees/feat"}
	ctx := context.Background()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "relative", path: "main.go", want: "/repo/.starxo/worktrees/feat/main.go"},
		{name: "workspace alias", path: "/workspace/main.go", want: "/repo/.starxo/worktrees/feat/main.go"},
		{name: "active absolute", path: "/repo/.starxo/worktrees/feat/main.go", want: "/repo/.starxo/worktrees/feat/main.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := workspaceFilePath(ctx, tt.path, "/repo", manager)
			if err != nil {
				t.Fatalf("workspaceFilePath(%q): %v", tt.path, err)
			}
			if got != tt.want {
				t.Fatalf("workspaceFilePath(%q) = %q; want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestWorkspaceFilePathRejectsParentAbsolutePathInWorktree(t *testing.T) {
	manager := fakeWorkspaceManager{workspace: "/repo/.starxo/worktrees/feat"}

	got, err := workspaceFilePath(context.Background(), "/repo/main.go", "/repo", manager)
	if err == nil {
		t.Fatalf("expected parent workspace path to be rejected, got %q", got)
	}
	if !strings.Contains(err.Error(), "/repo/.starxo/worktrees/feat") {
		t.Fatalf("expected error to name active worktree boundary, got %v", err)
	}
}

func TestWorkspaceFilePathAllowsActiveWorktreeAbsolutePathUnderWorkspaceRoot(t *testing.T) {
	manager := fakeWorkspaceManager{workspace: "/workspace/.starxo/worktrees/feat"}

	got, err := workspaceFilePath(context.Background(), "/workspace/.starxo/worktrees/feat/main.go", "/workspace", manager)
	if err != nil {
		t.Fatalf("workspaceFilePath active absolute: %v", err)
	}
	if want := "/workspace/.starxo/worktrees/feat/main.go"; got != want {
		t.Fatalf("workspaceFilePath active absolute = %q; want %q", got, want)
	}
}

func TestRuntimeDeferredEntriesMetadata(t *testing.T) {
	entries, err := NewRuntimeDeferredCatalogEntries(&fakeRuntimeOperator{}, "/workspace", nil, nil)
	if err != nil {
		t.Fatalf("runtime deferred entries: %v", err)
	}
	got := map[string]CatalogEntry{}
	for _, entry := range entries {
		got[entry.CanonicalName] = entry
		if entry.AlwaysLoad {
			t.Fatalf("expected %s to be deferred, got always-load", entry.CanonicalName)
		}
		if !entry.ShouldDefer {
			t.Fatalf("expected %s to opt into deferred loading", entry.CanonicalName)
		}
	}
	for _, name := range []string{RuntimeToolLSP, RuntimeToolSkill, RuntimeToolNotebookEdit} {
		if _, ok := got[name]; !ok {
			t.Fatalf("missing deferred runtime tool %s", name)
		}
	}
	for _, name := range []string{RuntimeToolLSP, RuntimeToolSkill} {
		entry := got[name]
		if !entry.ReadOnlyHint || !entry.ReadOnlyTrusted {
			t.Fatalf("expected %s to be read-only trusted, got %#v", name, entry)
		}
	}
	if entry := got[RuntimeToolNotebookEdit]; entry.ReadOnlyHint || entry.ReadOnlyTrusted {
		t.Fatalf("expected NotebookEdit to require write permission, got %#v", entry)
	}

	webFetch := RuntimeWebFetchCatalogEntry(nil)
	webSearch := RuntimeWebSearchCatalogEntry(nil)
	for _, entry := range []CatalogEntry{webFetch, webSearch} {
		if entry.AlwaysLoad || !entry.ShouldDefer {
			t.Fatalf("expected %s to be deferred, got %#v", entry.CanonicalName, entry)
		}
		if !entry.ReadOnlyHint || !entry.ReadOnlyTrusted {
			t.Fatalf("expected %s to be read-only trusted, got %#v", entry.CanonicalName, entry)
		}
	}
}

func TestRuntimeSearchPathGuard(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "default", path: "", want: "."},
		{name: "workspace prefix", path: "/workspace/src", want: "src"},
		{name: "relative clean", path: "src/../pkg", want: "pkg"},
		{name: "absolute outside", path: "/etc", wantErr: true},
		{name: "parent traversal", path: "../secret", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safeSearchPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("safeSearchPath(%q) = %q, %v; want %q", tt.path, got, err, tt.want)
			}
		})
	}
}

func TestMaybePersistLargeBashOutput(t *testing.T) {
	manager := &fakeRuntimeTaskManager{}
	out := maybePersistLargeBashOutput(context.Background(), manager, BashOutput{
		Stdout: strings.Repeat("x", runtimeLargeOutputThreshold+1),
	})
	if out.PersistedOutputPath != manager.persistedPath || out.PersistedOutputSize != manager.persistedSize {
		t.Fatalf("expected persisted output metadata, got %#v", out)
	}
	if !strings.Contains(out.Stdout, "[truncated; full output persisted]") {
		t.Fatalf("expected truncated stdout, got %q", out.Stdout)
	}
}
