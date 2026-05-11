package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
)

type fakeWorktreeOperator struct {
	commands       []string
	status         string
	base           string
	diffStat       string
	diff           string
	untrackedStat  string
	untrackedPatch string
	stderr         string
	exitCode       int
	prepareStdout  string
	prepareStderr  string
	prepareExit    int
	failMerge      bool
}

func (o *fakeWorktreeOperator) ReadFile(ctx context.Context, path string) (string, error) {
	return "", nil
}

func (o *fakeWorktreeOperator) WriteFile(ctx context.Context, path string, content string) error {
	return nil
}

func (o *fakeWorktreeOperator) IsDirectory(ctx context.Context, path string) (bool, error) {
	return false, nil
}

func (o *fakeWorktreeOperator) Exists(ctx context.Context, path string) (bool, error) {
	return false, nil
}

func (o *fakeWorktreeOperator) RunCommand(ctx context.Context, command []string) (*commandline.CommandOutput, error) {
	joined := strings.Join(command, " ")
	o.commands = append(o.commands, joined)
	if o.exitCode != 0 {
		return &commandline.CommandOutput{Stderr: o.stderr, ExitCode: o.exitCode}, nil
	}
	if o.failMerge && strings.Contains(joined, "merge --no-ff") {
		return &commandline.CommandOutput{Stderr: "merge conflict", ExitCode: 1}, nil
	}
	if strings.Contains(joined, "STARXO_WORKTREE_HEAD=") {
		if o.prepareExit != 0 {
			return &commandline.CommandOutput{Stderr: o.prepareStderr, ExitCode: o.prepareExit}, nil
		}
		stdout := o.prepareStdout
		if stdout == "" {
			stdout = "STARXO_WORKTREE_HEAD=worktreehead123\n"
		}
		return &commandline.CommandOutput{Stdout: stdout, ExitCode: 0}, nil
	}
	if strings.Contains(joined, "merge-base") {
		base := o.base
		if base == "" {
			base = "base123"
		}
		return &commandline.CommandOutput{Stdout: base + "\n", ExitCode: 0}, nil
	}
	if strings.Contains(joined, "rev-parse HEAD") {
		base := o.base
		if base == "" {
			base = "base123"
		}
		return &commandline.CommandOutput{Stdout: base + "\n", ExitCode: 0}, nil
	}
	if strings.Contains(joined, "status --short") || strings.Contains(joined, "status --porcelain") {
		return &commandline.CommandOutput{Stdout: o.status, ExitCode: 0}, nil
	}
	if strings.Contains(joined, "diff --stat") {
		return &commandline.CommandOutput{Stdout: o.diffStat, ExitCode: 0}, nil
	}
	if strings.Contains(joined, "wc -l") {
		return &commandline.CommandOutput{Stdout: o.untrackedStat, ExitCode: 0}, nil
	}
	if strings.Contains(joined, "ls-files --others --exclude-standard") {
		return &commandline.CommandOutput{Stdout: o.untrackedPatch, ExitCode: 0}, nil
	}
	if strings.Contains(joined, "diff --no-ext-diff") {
		return &commandline.CommandOutput{Stdout: o.diff, ExitCode: 0}, nil
	}
	return &commandline.CommandOutput{Stdout: "", Stderr: "", ExitCode: 0}, nil
}

func TestRuntimeWorkspaceManagerEnterAndKeepWorktree(t *testing.T) {
	manager := newRuntimeWorkspaceManager(func() time.Time { return time.Unix(10, 0) })
	op := &fakeWorktreeOperator{}
	ctx := contextWithSessionID(context.Background(), "sess-worktree")

	out, err := manager.EnterWorktree(ctx, op, "/workspace", "feat/a")
	if err != nil {
		t.Fatalf("enter worktree: %v", err)
	}
	if out.WorktreePath != "/workspace/.starxo/worktrees/feat-a" {
		t.Fatalf("unexpected worktree path: %#v", out)
	}
	if got := manager.CurrentWorkspace(ctx, "/workspace"); got != out.WorktreePath {
		t.Fatalf("expected current workspace to switch, got %q", got)
	}
	out, err = manager.ExitWorktree(ctx, op, "/workspace", "keep", false)
	if err != nil {
		t.Fatalf("exit worktree: %v", err)
	}
	if out.Action != "keep" {
		t.Fatalf("expected keep action, got %#v", out)
	}
	if got := manager.CurrentWorkspace(ctx, "/workspace"); got != "/workspace" {
		t.Fatalf("expected current workspace to restore, got %q", got)
	}
}

func TestRuntimeWorkspaceManagerCreateAddsStarxoToInfoExcludeBeforeWorktreeAdd(t *testing.T) {
	manager := newRuntimeWorkspaceManager(func() time.Time { return time.Unix(10, 0) })
	op := &fakeWorktreeOperator{}
	ctx := contextWithSessionID(context.Background(), "sess-worktree")

	if _, err := manager.EnterWorktree(ctx, op, "/workspace", "feat/a"); err != nil {
		t.Fatalf("enter worktree: %v", err)
	}
	if len(op.commands) == 0 {
		t.Fatalf("expected worktree creation command")
	}
	cmd := op.commands[0]
	excludeIndex := strings.Index(cmd, "git rev-parse --git-path info/exclude")
	worktreeIndex := strings.Index(cmd, "git worktree add")
	if excludeIndex < 0 {
		t.Fatalf("expected command to update git info/exclude, got %q", cmd)
	}
	if worktreeIndex < 0 {
		t.Fatalf("expected command to add worktree, got %q", cmd)
	}
	if excludeIndex > worktreeIndex {
		t.Fatalf("expected info/exclude update before worktree add, got %q", cmd)
	}
	if !strings.Contains(cmd, ".starxo/") {
		t.Fatalf("expected command to exclude .starxo/, got %q", cmd)
	}
	if strings.Contains(cmd, ".gitignore") {
		t.Fatalf("expected command not to touch tracked .gitignore, got %q", cmd)
	}
}

func TestRuntimeWorkspaceManagerRefusesDirtyRemoveWithoutDiscard(t *testing.T) {
	manager := newRuntimeWorkspaceManager(time.Now)
	op := &fakeWorktreeOperator{status: " M main.go\n"}
	ctx := contextWithSessionID(context.Background(), "sess-worktree")

	if _, err := manager.EnterWorktree(ctx, op, "/workspace", "feat"); err != nil {
		t.Fatalf("enter worktree: %v", err)
	}
	if _, err := manager.ExitWorktree(ctx, op, "/workspace", "remove", false); err == nil {
		t.Fatalf("expected dirty remove to fail without discard")
	}
	if got := manager.CurrentWorkspace(ctx, "/workspace"); !strings.Contains(got, ".starxo/worktrees/feat") {
		t.Fatalf("expected failed remove to keep active worktree, got %q", got)
	}
}

func TestRuntimeWorkspaceManagerIsolatedWorktreeDoesNotSwitchSession(t *testing.T) {
	manager := newRuntimeWorkspaceManager(func() time.Time { return time.Unix(20, 0) })
	op := &fakeWorktreeOperator{}
	ctx := contextWithSessionID(context.Background(), "sess-worktree")

	out, err := manager.CreateIsolatedWorktree(ctx, op, "/workspace", "agent/a")
	if err != nil {
		t.Fatalf("create isolated worktree: %v", err)
	}
	if out.WorktreePath != "/workspace/.starxo/worktrees/agent-a" {
		t.Fatalf("unexpected isolated worktree path: %#v", out)
	}
	if got := manager.CurrentWorkspace(ctx, "/workspace"); got != "/workspace" {
		t.Fatalf("isolated worktree should not switch session workspace, got %q", got)
	}
	overrideCtx := contextWithRuntimeWorkspaceOverride(ctx, out.WorktreePath)
	if got := manager.CurrentWorkspace(overrideCtx, "/workspace"); got != out.WorktreePath {
		t.Fatalf("expected context override workspace, got %q", got)
	}
}

func TestRuntimeWorkspaceManagerDiffWorktreeIncludesTrackedAndUntrackedPatch(t *testing.T) {
	manager := newRuntimeWorkspaceManager(func() time.Time { return time.Unix(20, 0) })
	op := &fakeWorktreeOperator{
		status:        " M main.go\n",
		diffStat:      " main.go | 2 +-\n",
		diff:          "tracked\n",
		untrackedStat: " new.txt | 1 +\n",
		untrackedPatch: "diff --git a/new.txt b/new.txt\n" +
			"new file mode 100644\n" +
			"--- /dev/null\n" +
			"+++ b/new.txt\n" +
			"+new content\n",
	}
	ctx := contextWithSessionID(context.Background(), "sess-worktree")
	if _, err := manager.EnterWorktree(ctx, op, "/workspace", "feat"); err != nil {
		t.Fatalf("enter worktree: %v", err)
	}

	out, err := manager.DiffWorktree(ctx, op, "/workspace", true, 10000)
	if err != nil {
		t.Fatalf("diff worktree: %v", err)
	}
	if out.WorktreePath != "/workspace/.starxo/worktrees/feat" || out.WorktreeBranch != "starxo/feat" {
		t.Fatalf("unexpected worktree metadata: %#v", out)
	}
	if strings.TrimSpace(out.Status) != "M main.go" || !strings.Contains(out.DiffStat, "main.go") || !strings.Contains(out.DiffStat, "new.txt") {
		t.Fatalf("unexpected diff summary: %#v", out)
	}
	if !strings.Contains(out.Diff, "tracked") || !strings.Contains(out.Diff, "new.txt") || out.Truncated {
		t.Fatalf("expected tracked and untracked patch content, got %#v", out)
	}
	if !strings.Contains(out.UntrackedDiff, "new.txt") || out.UntrackedTruncated {
		t.Fatalf("expected separate untracked patch content, got %#v", out)
	}
	allCommands := strings.Join(op.commands, "\n")
	if !strings.Contains(allCommands, "diff --stat --no-ext-diff 'base123'") ||
		!strings.Contains(allCommands, "diff --no-ext-diff 'base123'") {
		t.Fatalf("expected diff to compare active worktree against merge-base, got:\n%s", allCommands)
	}
	if !strings.Contains(allCommands, "merge-base HEAD") {
		t.Fatalf("expected diff base to use merge-base, got:\n%s", allCommands)
	}
	if !strings.Contains(allCommands, "ls-files --others --exclude-standard") {
		t.Fatalf("expected diff to include untracked file patch content, got:\n%s", allCommands)
	}
	if !strings.Contains(allCommands, "wc -l") {
		t.Fatalf("expected diff stat to include untracked files, got:\n%s", allCommands)
	}
	if !strings.Contains(allCommands, "head -c 10001") {
		t.Fatalf("expected patch commands to cap remote output before collection, got:\n%s", allCommands)
	}
}

func TestRuntimeWorkspaceManagerMergeWorktreeCommitsMergesRemovesAndRestoresWorkspace(t *testing.T) {
	manager := newRuntimeWorkspaceManager(func() time.Time { return time.Unix(20, 0) })
	op := &fakeWorktreeOperator{status: " M main.go\n"}
	ctx := contextWithSessionID(context.Background(), "sess-worktree")
	if _, err := manager.EnterWorktree(ctx, op, "/workspace", "feat"); err != nil {
		t.Fatalf("enter worktree: %v", err)
	}

	out, err := manager.MergeWorktree(ctx, op, "/workspace", "merge feat", true)
	if err != nil {
		t.Fatalf("merge worktree: %v", err)
	}
	if !out.Removed || out.WorkspacePath != "/workspace" || out.CommitMessage != "merge feat" {
		t.Fatalf("unexpected merge output: %#v", out)
	}
	if got := manager.CurrentWorkspace(ctx, "/workspace"); got != "/workspace" {
		t.Fatalf("expected workspace restored after merge, got %q", got)
	}
	allCommands := strings.Join(op.commands, "\n")
	for _, want := range []string{"parent workspace has uncommitted changes", "branch --show-current", "worktree branch changed", "rev-parse HEAD", "git -C '/workspace/.starxo/worktrees/feat' add -A", "commit -m 'merge feat'", "merge --no-ff 'worktreehead123'", "worktree remove '/workspace/.starxo/worktrees/feat'"} {
		if !strings.Contains(allCommands, want) {
			t.Fatalf("expected merge command to contain %q, got:\n%s", want, allCommands)
		}
	}
}

func TestRuntimeWorkspaceManagerMergeWorktreeRejectsChangedBranch(t *testing.T) {
	manager := newRuntimeWorkspaceManager(func() time.Time { return time.Unix(20, 0) })
	op := &fakeWorktreeOperator{
		prepareExit:   3,
		prepareStderr: "worktree branch changed; expected starxo/feat got other",
	}
	ctx := contextWithSessionID(context.Background(), "sess-worktree")
	if _, err := manager.EnterWorktree(ctx, op, "/workspace", "feat"); err != nil {
		t.Fatalf("enter worktree: %v", err)
	}

	_, err := manager.MergeWorktree(ctx, op, "/workspace", "merge feat", false)
	if err == nil || !strings.Contains(err.Error(), "worktree branch changed") {
		t.Fatalf("expected changed branch failure, got %v", err)
	}
	allCommands := strings.Join(op.commands, "\n")
	if strings.Contains(allCommands, "merge --no-ff") {
		t.Fatalf("changed branch must not run merge, got:\n%s", allCommands)
	}
	if got := manager.CurrentWorkspace(ctx, "/workspace"); got != "/workspace/.starxo/worktrees/feat" {
		t.Fatalf("expected failed prepare to keep active worktree, got %q", got)
	}
}

func TestRuntimeWorkspaceManagerMergeWorktreeAbortsFailedMergeAndKeepsActiveState(t *testing.T) {
	manager := newRuntimeWorkspaceManager(func() time.Time { return time.Unix(20, 0) })
	op := &fakeWorktreeOperator{failMerge: true}
	ctx := contextWithSessionID(context.Background(), "sess-worktree")
	if _, err := manager.EnterWorktree(ctx, op, "/workspace", "feat"); err != nil {
		t.Fatalf("enter worktree: %v", err)
	}

	_, err := manager.MergeWorktree(ctx, op, "/workspace", "merge feat", false)
	if err == nil || !strings.Contains(err.Error(), "merge conflict") {
		t.Fatalf("expected merge conflict error, got %v", err)
	}
	if got := manager.CurrentWorkspace(ctx, "/workspace"); got != "/workspace/.starxo/worktrees/feat" {
		t.Fatalf("expected failed merge to keep active worktree, got %q", got)
	}
	allCommands := strings.Join(op.commands, "\n")
	if !strings.Contains(allCommands, "merge --abort") {
		t.Fatalf("expected failed merge to abort parent merge state, got:\n%s", allCommands)
	}
}
