# Runtime Worktree Conflict Recovery

## Summary
- Branch: `feature/runtime-worktree-conflict-recovery`.
- Add structured conflict recovery for runtime worktree merges.
- Keep the parent workspace clean by aborting failed parent merges after capturing conflict metadata.

## Changes
- `WorktreeMergeOutput` now includes conflict metadata: `conflicted`, `conflictFiles`, `mergeOutput`, and `recoveryHint`.
- `MergeWorktree` detects Git merge conflicts, collects unresolved file names, aborts the parent merge state, keeps the active worktree, and returns a structured `merge_conflict` result instead of a plain error.
- Conflict recovery only returns success after `git merge --abort` succeeds; abort failures are surfaced as errors.
- Worktree merge timeline output uses JSON-aware truncation so conflict metadata remains parseable.
- The workspace worktree panel shows a warning with conflict files, merge output, and recovery guidance so users can resolve in the worktree and retry.

## Verification
- `go test ./internal/service -run 'TestRuntimeWorkspaceManagerMergeWorktree'`
- `go test ./...`
- `cd frontend && npm run build`
- GitHub Codex review before merge.
