package service

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/commandline"

	"starxo/internal/model"
	"starxo/internal/tools"
)

type runtimeWorktreeState struct {
	SessionID         string
	OriginalWorkspace string
	WorktreePath      string
	WorktreeBranch    string
	Slug              string
}

type runtimeWorkspaceOverrideKey struct{}

type runtimeWorkspaceManager struct {
	mu     sync.RWMutex
	states map[string]runtimeWorktreeState
	now    func() time.Time
}

func newRuntimeWorkspaceManager(now func() time.Time) *runtimeWorkspaceManager {
	if now == nil {
		now = time.Now
	}
	return &runtimeWorkspaceManager{
		states: make(map[string]runtimeWorktreeState),
		now:    now,
	}
}

func (m *runtimeWorkspaceManager) CurrentWorkspace(ctx context.Context, defaultWorkspace string) string {
	if override := runtimeWorkspaceOverride(ctx); override != "" {
		return override
	}
	sessionID := workspaceSessionID(ctx)
	if sessionID == "" {
		return defaultWorkspace
	}
	m.mu.RLock()
	state, ok := m.states[sessionID]
	m.mu.RUnlock()
	if !ok || strings.TrimSpace(state.WorktreePath) == "" {
		return defaultWorkspace
	}
	return state.WorktreePath
}

func (m *runtimeWorkspaceManager) CompactSnapshot(sessionID string, defaultWorkspace string) *model.RuntimeWorkspaceCompact {
	if sessionID == "" {
		return nil
	}
	m.mu.RLock()
	state, ok := m.states[sessionID]
	m.mu.RUnlock()
	if !ok {
		if strings.TrimSpace(defaultWorkspace) == "" {
			return nil
		}
		return &model.RuntimeWorkspaceCompact{
			Active:        false,
			WorkspacePath: cleanRuntimeRemotePath(defaultWorkspace),
		}
	}
	return &model.RuntimeWorkspaceCompact{
		Active:            true,
		WorkspacePath:     state.WorktreePath,
		OriginalWorkspace: state.OriginalWorkspace,
		WorktreePath:      state.WorktreePath,
		WorktreeBranch:    state.WorktreeBranch,
	}
}

func (m *runtimeWorkspaceManager) RestoreCompactSnapshot(sessionID string, compact *model.RuntimeWorkspaceCompact) {
	if sessionID == "" || compact == nil || !compact.Active || strings.TrimSpace(compact.WorktreePath) == "" {
		return
	}
	m.mu.Lock()
	m.states[sessionID] = runtimeWorktreeState{
		SessionID:         sessionID,
		OriginalWorkspace: cleanRuntimeRemotePath(compact.OriginalWorkspace),
		WorktreePath:      cleanRuntimeRemotePath(compact.WorktreePath),
		WorktreeBranch:    compact.WorktreeBranch,
		Slug:              sanitizeWorktreeSlug(strings.TrimPrefix(compact.WorktreeBranch, "starxo/")),
	}
	m.mu.Unlock()
}

func (m *runtimeWorkspaceManager) EnterWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, name string) (tools.WorktreeOutput, error) {
	sessionID := workspaceSessionID(ctx)
	if sessionID == "" {
		sessionID = "global"
	}
	m.mu.RLock()
	_, exists := m.states[sessionID]
	m.mu.RUnlock()
	if exists {
		return tools.WorktreeOutput{}, fmt.Errorf("already in a worktree session")
	}

	state, out, err := m.createWorktree(ctx, op, defaultWorkspace, name)
	if err != nil {
		return tools.WorktreeOutput{}, err
	}
	state.SessionID = sessionID
	m.mu.Lock()
	m.states[sessionID] = state
	m.mu.Unlock()
	return out, nil
}

func (m *runtimeWorkspaceManager) CreateIsolatedWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, name string) (tools.WorktreeOutput, error) {
	_, out, err := m.createWorktree(ctx, op, defaultWorkspace, name)
	if err != nil {
		return tools.WorktreeOutput{}, err
	}
	out.Message = fmt.Sprintf("Created isolated worktree at %s on branch %s. This subagent uses the worktree; the parent session workspace is unchanged.", out.WorktreePath, out.WorktreeBranch)
	return out, nil
}

func (m *runtimeWorkspaceManager) createWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, name string) (runtimeWorktreeState, tools.WorktreeOutput, error) {
	if op == nil {
		return runtimeWorktreeState{}, tools.WorktreeOutput{}, fmt.Errorf("sandbox operator is not available")
	}
	defaultWorkspace = cleanRuntimeRemotePath(defaultWorkspace)
	if defaultWorkspace == "" || defaultWorkspace == "." {
		return runtimeWorktreeState{}, tools.WorktreeOutput{}, fmt.Errorf("sandbox workspace is not active")
	}
	slug := sanitizeWorktreeSlug(name)
	if slug == "" {
		slug = fmt.Sprintf("wt-%d", m.now().Unix())
	}
	worktreePath := path.Join(defaultWorkspace, ".starxo", "worktrees", slug)
	branch := "starxo/" + slug
	cmd := strings.Join([]string{
		"cd " + shellQuoteRuntime(defaultWorkspace),
		"git rev-parse --is-inside-work-tree >/dev/null",
		"exclude_file=$(git rev-parse --git-path info/exclude)",
		"mkdir -p \"$(dirname \"$exclude_file\")\"",
		"touch \"$exclude_file\"",
		"grep -qxF '.starxo/' \"$exclude_file\" || printf '%s\\n' '.starxo/' >> \"$exclude_file\"",
		"mkdir -p .starxo/worktrees",
		"git worktree add -b " + shellQuoteRuntime(branch) + " " + shellQuoteRuntime(worktreePath) + " HEAD",
	}, " && ")
	output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd})
	if err != nil {
		return runtimeWorktreeState{}, tools.WorktreeOutput{}, err
	}
	if output.ExitCode != 0 {
		return runtimeWorktreeState{}, tools.WorktreeOutput{}, fmt.Errorf("failed to create worktree: %s", output.Stderr)
	}

	state := runtimeWorktreeState{
		OriginalWorkspace: defaultWorkspace,
		WorktreePath:      worktreePath,
		WorktreeBranch:    branch,
		Slug:              slug,
	}
	return state, tools.WorktreeOutput{
		Action:         "enter",
		WorkspacePath:  worktreePath,
		WorktreePath:   worktreePath,
		WorktreeBranch: branch,
		Message:        fmt.Sprintf("Created worktree at %s on branch %s. This session now uses the worktree as its workspace.", worktreePath, branch),
	}, nil
}

func (m *runtimeWorkspaceManager) ExitWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, action string, discardChanges bool) (tools.WorktreeOutput, error) {
	if op == nil {
		return tools.WorktreeOutput{}, fmt.Errorf("sandbox operator is not available")
	}
	sessionID := workspaceSessionID(ctx)
	if sessionID == "" {
		sessionID = "global"
	}
	m.mu.RLock()
	state, ok := m.states[sessionID]
	m.mu.RUnlock()
	if !ok {
		return tools.WorktreeOutput{
			Action:        "noop",
			WorkspacePath: cleanRuntimeRemotePath(defaultWorkspace),
			Message:       "No active worktree for this session.",
		}, nil
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "keep"
	}
	if action != "keep" && action != "remove" {
		return tools.WorktreeOutput{}, fmt.Errorf("unsupported action %q; use keep or remove", action)
	}

	if action == "remove" {
		statusCmd := "git -C " + shellQuoteRuntime(state.WorktreePath) + " status --porcelain"
		status, err := op.RunCommand(ctx, []string{"sh", "-lc", statusCmd})
		if err != nil {
			return tools.WorktreeOutput{}, err
		}
		if strings.TrimSpace(status.Stdout) != "" && !discardChanges {
			return tools.WorktreeOutput{}, fmt.Errorf("worktree has uncommitted changes; re-run with discard_changes=true to remove it or action=keep to preserve it")
		}
		force := ""
		branchDeleteFlag := "-d"
		if discardChanges {
			force = "--force"
			branchDeleteFlag = "-D"
		}
		removeCmd := strings.Join([]string{
			"cd " + shellQuoteRuntime(state.OriginalWorkspace),
			"git worktree remove " + force + " " + shellQuoteRuntime(state.WorktreePath),
			"git branch " + branchDeleteFlag + " " + shellQuoteRuntime(state.WorktreeBranch) + " >/dev/null 2>&1 || true",
		}, " && ")
		output, err := op.RunCommand(ctx, []string{"sh", "-lc", removeCmd})
		if err != nil {
			return tools.WorktreeOutput{}, err
		}
		if output.ExitCode != 0 {
			return tools.WorktreeOutput{}, fmt.Errorf("failed to remove worktree: %s", output.Stderr)
		}
	}

	m.mu.Lock()
	delete(m.states, sessionID)
	m.mu.Unlock()

	message := "Kept worktree on disk and restored the original sandbox workspace."
	if action == "remove" {
		message = "Removed worktree and restored the original sandbox workspace."
	}
	return tools.WorktreeOutput{
		Action:         action,
		WorkspacePath:  state.OriginalWorkspace,
		WorktreePath:   state.WorktreePath,
		WorktreeBranch: state.WorktreeBranch,
		Message:        message,
	}, nil
}

func workspaceSessionID(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDCtxKey).(string); ok {
		return v
	}
	if v, ok := ctx.Value("sessionID").(string); ok {
		return v
	}
	return ""
}

func contextWithRuntimeWorkspaceOverride(ctx context.Context, workspace string) context.Context {
	workspace = cleanRuntimeRemotePath(workspace)
	if workspace == "" {
		return ctx
	}
	return context.WithValue(ctx, runtimeWorkspaceOverrideKey{}, workspace)
}

func runtimeWorkspaceOverride(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(runtimeWorkspaceOverrideKey{}).(string); ok {
		return cleanRuntimeRemotePath(v)
	}
	return ""
}

func sanitizeWorktreeSlug(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
		if b.Len() >= 64 {
			break
		}
	}
	return strings.Trim(b.String(), "-_.")
}

func cleanRuntimeRemotePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	cleaned := path.Clean(p)
	if strings.HasPrefix(p, "/") && !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
}

func shellQuoteRuntime(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
