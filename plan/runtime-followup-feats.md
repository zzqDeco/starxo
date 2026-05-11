# Runtime Follow-Up Features

## Summary
- Branch: `feature/runtime-followup-feats`.
- Scope: continue Runtime V2 after the LSP/WebSearch settings batch.
- Features:
  - Add worktree review and merge tools for active runtime worktrees.
  - Add explicit WebSearch smoke testing from settings.
  - Improve LSP status visibility by showing configured server mappings.

## Changes
- Runtime tools:
  - Added deferred read-only `WorktreeDiff` for status, diff stat, and optional patch review.
  - Added deferred writable `WorktreeMerge` to commit active worktree changes, merge back to the original workspace, optionally remove the worktree, and restore the original session workspace.
  - `WorktreeMerge` refuses to proceed when the parent workspace is dirty.
- Settings:
  - Added `SettingsService.TestWebSearch(cfg, query)` for explicit user-triggered WebSearch smoke tests.
  - WebSearch settings now show smoke-test result metadata and compact results.
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
