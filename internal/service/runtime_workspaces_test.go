package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
)

type fakeWorktreeOperator struct {
	commands []string
	status   string
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
	if strings.Contains(joined, "status --porcelain") {
		return &commandline.CommandOutput{Stdout: o.status, ExitCode: 0}, nil
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
