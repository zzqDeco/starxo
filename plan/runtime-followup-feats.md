# Runtime Follow-Up Features

## Summary
- Branch: `feature/runtime-followup-feats-review`.
- Scope: continue Runtime V2 after the LSP/WebSearch settings batch.
- Features:
  - Add worktree review and merge tools for active runtime worktrees.
  - Add explicit WebSearch smoke testing from settings.
  - Improve LSP status visibility by showing configured server mappings.

## Changes
- Runtime tools:
  - Added deferred read-only `WorktreeDiff` for status, diff stat, and optional patch review.
  - `WorktreeDiff` compares against the parent/worktree merge-base so parent-only commits are not shown as worktree reversions.
  - `WorktreeDiff` includes untracked file stat and patch content so review output matches what `git add -A` would merge.
  - Added deferred writable `WorktreeMerge` to commit active worktree changes, merge the verified worktree HEAD back to the original workspace, optionally remove the worktree, and restore the original session workspace.
  - `WorktreeMerge` refuses to proceed when the parent workspace is dirty.
  - `WorktreeMerge` verifies the active worktree branch before and after prepare, then merges the resolved worktree HEAD SHA instead of a stale branch name.
- Settings:
  - Added `SettingsService.TestWebSearch(cfg, query)` for explicit user-triggered WebSearch smoke tests.
  - WebSearch settings now show smoke-test result metadata and compact results; a successful provider response with zero parsed results is reported as a failed smoke result.
  - LSP status now shows configured server mappings in addition to running server processes.

## Verification
- Targeted Go tests for worktree diff/merge and WebSearch smoke failure behavior.
- Full `go test ./...`.
- Frontend `npm run build`.
- GitHub Codex review before merge.

## Follow-Ups
- Add a visual diff panel for worktree patches.
- Add explicit conflict recovery UX for failed worktree merges.
- Add writable LSP actions behind permission approval.
