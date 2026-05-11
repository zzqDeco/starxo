package service

import (
	"context"
	"fmt"
	"path"
	"strconv"
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

func (m *runtimeWorkspaceManager) DiffWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace string, includePatch bool, maxBytes int) (tools.WorktreeDiffOutput, error) {
	if op == nil {
		return tools.WorktreeDiffOutput{}, fmt.Errorf("sandbox operator is not available")
	}
	state, ok := m.activeState(ctx)
	if !ok {
		return tools.WorktreeDiffOutput{}, fmt.Errorf("no active worktree for this session")
	}
	if maxBytes <= 0 {
		maxBytes = 20000
	}
	base, err := runRuntimeWorktreeCommand(ctx, op, runtimeWorktreeDiffBaseCommand(state.OriginalWorkspace, state.WorktreePath))
	if err != nil {
		return tools.WorktreeDiffOutput{}, err
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return tools.WorktreeDiffOutput{}, fmt.Errorf("could not resolve original workspace HEAD")
	}
	status, err := runRuntimeWorktreeCommand(ctx, op, "git -C "+shellQuoteRuntime(state.WorktreePath)+" status --short")
	if err != nil {
		return tools.WorktreeDiffOutput{}, err
	}
	trackedStat, err := runRuntimeWorktreeCommand(ctx, op, "git -C "+shellQuoteRuntime(state.WorktreePath)+" diff --stat --no-ext-diff "+shellQuoteRuntime(base))
	if err != nil {
		return tools.WorktreeDiffOutput{}, err
	}
	untrackedStat, err := runRuntimeWorktreeCommand(ctx, op, runtimeUntrackedStatCommand(state.WorktreePath))
	if err != nil {
		return tools.WorktreeDiffOutput{}, err
	}
	out := tools.WorktreeDiffOutput{
		WorkspacePath:  state.OriginalWorkspace,
		WorktreePath:   state.WorktreePath,
		WorktreeBranch: state.WorktreeBranch,
		Status:         strings.TrimRight(status, "\n"),
		DiffStat:       strings.TrimRight(combineRuntimeDiffs(trackedStat, untrackedStat), "\n"),
		Message:        "Active worktree changes are ready for review.",
	}
	if includePatch {
		// MergeWorktree commits with git add -A, so review output must include
		// untracked file contents in addition to the tracked diff.
		untracked, err := runRuntimeWorktreeCommand(ctx, op, limitRuntimeWorktreeOutputCommand(runtimeUntrackedPatchCommand(state.WorktreePath), maxBytes))
		if err != nil {
			return tools.WorktreeDiffOutput{}, err
		}
		diff, err := runRuntimeWorktreeCommand(ctx, op, limitRuntimeWorktreeOutputCommand("git -C "+shellQuoteRuntime(state.WorktreePath)+" diff --no-ext-diff "+shellQuoteRuntime(base), maxBytes))
		if err != nil {
			return tools.WorktreeDiffOutput{}, err
		}
		out.UntrackedDiff, out.UntrackedTruncated = limitRuntimeDiff(untracked, maxBytes)
		diff = combineRuntimeDiffs(untracked, diff)
		if len(diff) > maxBytes {
			out.Diff = diff[:maxBytes]
			out.Truncated = true
		} else {
			out.Diff = diff
		}
	}
	if strings.TrimSpace(out.Status) == "" && strings.TrimSpace(out.DiffStat) == "" {
		out.Message = "Active worktree has no uncommitted changes."
	}
	return out, nil
}

func (m *runtimeWorkspaceManager) MergeWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace string, commitMessage string, removeWorktree bool) (tools.WorktreeMergeOutput, error) {
	if op == nil {
		return tools.WorktreeMergeOutput{}, fmt.Errorf("sandbox operator is not available")
	}
	state, ok := m.activeState(ctx)
	if !ok {
		return tools.WorktreeMergeOutput{}, fmt.Errorf("no active worktree for this session")
	}
	commitMessage = strings.TrimSpace(commitMessage)
	if commitMessage == "" {
		commitMessage = "Starxo runtime worktree merge"
	}
	prepareCmd := strings.Join([]string{
		"set -e",
		"parent_status=$(git -C " + shellQuoteRuntime(state.OriginalWorkspace) + " status --porcelain)",
		"if [ -n \"$parent_status\" ]; then printf '%s\\n' \"parent workspace has uncommitted changes\" >&2; exit 2; fi",
		"current_branch=$(git -C " + shellQuoteRuntime(state.WorktreePath) + " branch --show-current)",
		"if [ \"$current_branch\" != " + shellQuoteRuntime(state.WorktreeBranch) + " ]; then printf 'worktree branch changed; expected %s got %s\\n' " + shellQuoteRuntime(state.WorktreeBranch) + " \"$current_branch\" >&2; exit 3; fi",
		"worktree_status=$(git -C " + shellQuoteRuntime(state.WorktreePath) + " status --porcelain)",
		"if [ -n \"$worktree_status\" ]; then git -C " + shellQuoteRuntime(state.WorktreePath) + " add -A && git -C " + shellQuoteRuntime(state.WorktreePath) + " -c user.name=Starxo -c user.email=starxo@local commit -m " + shellQuoteRuntime(commitMessage) + "; fi",
		"current_branch=$(git -C " + shellQuoteRuntime(state.WorktreePath) + " branch --show-current)",
		"if [ \"$current_branch\" != " + shellQuoteRuntime(state.WorktreeBranch) + " ]; then printf 'worktree branch changed after commit; expected %s got %s\\n' " + shellQuoteRuntime(state.WorktreeBranch) + " \"$current_branch\" >&2; exit 3; fi",
		"printf 'STARXO_WORKTREE_HEAD=%s\\n' \"$(git -C " + shellQuoteRuntime(state.WorktreePath) + " rev-parse HEAD)\"",
	}, "\n")
	prepareOut, err := op.RunCommand(ctx, []string{"sh", "-lc", prepareCmd})
	if err != nil {
		return tools.WorktreeMergeOutput{}, err
	}
	if prepareOut.ExitCode != 0 {
		return tools.WorktreeMergeOutput{}, fmt.Errorf("failed to prepare worktree merge: %s", strings.TrimSpace(prepareOut.Stderr))
	}
	mergeRef := runtimeWorktreeHeadFromPrepareOutput(prepareOut.Stdout)
	if mergeRef == "" {
		return tools.WorktreeMergeOutput{}, fmt.Errorf("failed to prepare worktree merge: could not resolve worktree HEAD")
	}
	mergeCmd := "git -C " + shellQuoteRuntime(state.OriginalWorkspace) + " merge --no-ff " + shellQuoteRuntime(mergeRef) + " -m " + shellQuoteRuntime(commitMessage)
	output, err := op.RunCommand(ctx, []string{"sh", "-lc", mergeCmd})
	if err != nil {
		abortRuntimeWorktreeMerge(ctx, op, state.OriginalWorkspace)
		return tools.WorktreeMergeOutput{}, err
	}
	if output.ExitCode != 0 {
		mergeOutput := runtimeWorktreeMergeOutputText(output)
		if runtimeWorktreeLooksLikeConflict(mergeOutput) {
			conflictFiles := runtimeWorktreeConflictFiles(ctx, op, state.OriginalWorkspace)
			abortRuntimeWorktreeMerge(ctx, op, state.OriginalWorkspace)
			return tools.WorktreeMergeOutput{
				Action:         "merge_conflict",
				WorkspacePath:  state.OriginalWorkspace,
				WorktreePath:   state.WorktreePath,
				WorktreeBranch: state.WorktreeBranch,
				CommitMessage:  commitMessage,
				Conflicted:     true,
				ConflictFiles:  conflictFiles,
				MergeOutput:    limitRuntimeWorktreeMergeOutput(mergeOutput, 6000),
				RecoveryHint:   "Parent merge was aborted and the active worktree was preserved. Reconcile the listed files in the worktree, review again, then retry merge.",
				Message:        "Worktree merge hit conflicts; parent merge was aborted and the worktree remains active.",
			}, nil
		}
		abortRuntimeWorktreeMerge(ctx, op, state.OriginalWorkspace)
		return tools.WorktreeMergeOutput{}, fmt.Errorf("failed to merge worktree: %s", strings.TrimSpace(mergeOutput))
	}
	removed := false
	if removeWorktree {
		removeCmd := strings.Join([]string{
			"set -e",
			"git -C " + shellQuoteRuntime(state.OriginalWorkspace) + " worktree remove " + shellQuoteRuntime(state.WorktreePath),
			"git -C " + shellQuoteRuntime(state.OriginalWorkspace) + " branch -d " + shellQuoteRuntime(state.WorktreeBranch) + " >/dev/null 2>&1 || true",
		}, "\n")
		removeOut, err := op.RunCommand(ctx, []string{"sh", "-lc", removeCmd})
		if err != nil {
			return tools.WorktreeMergeOutput{}, err
		}
		if removeOut.ExitCode != 0 {
			return tools.WorktreeMergeOutput{}, fmt.Errorf("merged worktree but failed to remove it: %s", strings.TrimSpace(removeOut.Stderr))
		}
		removed = true
	}
	sessionID := workspaceSessionID(ctx)
	if sessionID == "" {
		sessionID = "global"
	}
	m.mu.Lock()
	delete(m.states, sessionID)
	m.mu.Unlock()

	message := "Merged worktree into the original sandbox workspace and restored the original workspace."
	if removed {
		message = "Merged worktree, removed it, and restored the original sandbox workspace."
	}
	return tools.WorktreeMergeOutput{
		Action:         "merge",
		WorkspacePath:  state.OriginalWorkspace,
		WorktreePath:   state.WorktreePath,
		WorktreeBranch: state.WorktreeBranch,
		CommitMessage:  commitMessage,
		Removed:        removed,
		Message:        message,
	}, nil
}

func (m *runtimeWorkspaceManager) activeState(ctx context.Context) (runtimeWorktreeState, bool) {
	sessionID := workspaceSessionID(ctx)
	if sessionID == "" {
		sessionID = "global"
	}
	m.mu.RLock()
	state, ok := m.states[sessionID]
	m.mu.RUnlock()
	return state, ok
}

func abortRuntimeWorktreeMerge(ctx context.Context, op commandline.Operator, workspace string) {
	if op == nil || strings.TrimSpace(workspace) == "" {
		return
	}
	_, _ = op.RunCommand(ctx, []string{"sh", "-lc", "git -C " + shellQuoteRuntime(workspace) + " merge --abort >/dev/null 2>&1 || true"})
}

func runtimeWorktreeMergeOutputText(output *commandline.CommandOutput) string {
	if output == nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimSpace(output.Stdout) + "\n" + strings.TrimSpace(output.Stderr))
}

func runtimeWorktreeLooksLikeConflict(output string) bool {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	return strings.Contains(trimmed, "CONFLICT") ||
		strings.Contains(lower, "automatic merge failed") ||
		strings.Contains(lower, "fix conflicts")
}

func runtimeWorktreeConflictFiles(ctx context.Context, op commandline.Operator, workspace string) []string {
	if op == nil || strings.TrimSpace(workspace) == "" {
		return nil
	}
	cmd := "git -C " + shellQuoteRuntime(workspace) + " diff --name-only --diff-filter=U"
	output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd})
	if err != nil || output.ExitCode != 0 {
		return nil
	}
	lines := strings.Split(output.Stdout, "\n")
	files := make([]string, 0, len(lines))
	seen := make(map[string]struct{})
	for _, line := range lines {
		file := strings.TrimSpace(line)
		if file == "" {
			continue
		}
		if _, ok := seen[file]; ok {
			continue
		}
		seen[file] = struct{}{}
		files = append(files, file)
	}
	return files
}

func limitRuntimeWorktreeMergeOutput(output string, maxBytes int) string {
	if maxBytes <= 0 || len(output) <= maxBytes {
		return output
	}
	return output[:maxBytes] + "\n...truncated..."
}

func runRuntimeWorktreeCommand(ctx context.Context, op commandline.Operator, cmd string) (string, error) {
	output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd})
	if err != nil {
		return "", err
	}
	if output.ExitCode != 0 {
		return "", fmt.Errorf("worktree command failed: %s", strings.TrimSpace(output.Stderr))
	}
	return output.Stdout, nil
}

func runtimeWorktreeDiffBaseCommand(originalWorkspace string, worktreePath string) string {
	original := shellQuoteRuntime(originalWorkspace)
	worktree := shellQuoteRuntime(worktreePath)
	return strings.Join([]string{
		"original=" + original,
		"worktree=" + worktree,
		"worktree_head=$(git -C \"$worktree\" rev-parse HEAD)",
		"git -C \"$original\" merge-base HEAD \"$worktree_head\"",
	}, "\n")
}

func runtimeWorktreeHeadFromPrepareOutput(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STARXO_WORKTREE_HEAD=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "STARXO_WORKTREE_HEAD="))
		}
	}
	return ""
}

func runtimeUntrackedStatCommand(worktreePath string) string {
	quoted := shellQuoteRuntime(worktreePath)
	return strings.Join([]string{
		"wt=" + quoted,
		"git -C " + quoted + " ls-files --others --exclude-standard | while IFS= read -r file; do",
		"  [ -n \"$file\" ] || continue",
		"  [ -f \"$wt/$file\" ] || continue",
		"  lines=$(wc -l < \"$wt/$file\" | tr -d ' ')",
		"  [ -n \"$lines\" ] || lines=0",
		"  printf ' %s | %s +\\n' \"$file\" \"$lines\"",
		"done",
	}, "\n")
}

func runtimeUntrackedPatchCommand(worktreePath string) string {
	quoted := shellQuoteRuntime(worktreePath)
	return strings.Join([]string{
		"wt=" + quoted,
		"git -C " + quoted + " ls-files --others --exclude-standard | while IFS= read -r file; do",
		"  [ -n \"$file\" ] || continue",
		"  [ -f \"$wt/$file\" ] || continue",
		"  printf 'diff --git a/%s b/%s\\nnew file mode 100644\\n--- /dev/null\\n+++ b/%s\\n' \"$file\" \"$file\" \"$file\"",
		"  sed 's/^/+/' \"$wt/$file\"",
		"  printf '\\n'",
		"done",
	}, "\n")
}

func limitRuntimeWorktreeOutputCommand(cmd string, maxBytes int) string {
	if maxBytes <= 0 {
		return cmd
	}
	return "(\n" + cmd + "\n) | head -c " + strconv.Itoa(maxBytes+1)
}

func combineRuntimeDiffs(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimRight(part, "\n")
		if strings.TrimSpace(part) == "" {
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, "\n")
}

func limitRuntimeDiff(diff string, maxBytes int) (string, bool) {
	if maxBytes <= 0 {
		return diff, false
	}
	if len(diff) <= maxBytes {
		return diff, false
	}
	return diff[:maxBytes], true
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
