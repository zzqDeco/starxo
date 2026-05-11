# Runtime Worktree UX

## Summary
- Branch: `feature/runtime-worktree-ux`.
- Goal: turn Runtime V2 worktree review/merge tools into a visible desktop workflow.
- Scope:
  - Workspace drawer shows active worktree metadata for the current session.
  - Users can review status/stat/patch, copy review output, merge with confirmation, remove after merge, or exit and keep the worktree.
  - Workspace file browser follows the active runtime worktree instead of always showing the original sandbox workspace.
  - Timeline renders `EnterWorktree`, `ExitWorktree`, `WorktreeDiff`, and `WorktreeMerge` results as structured worktree events.

## Backend
- Added ChatService Wails methods:
  - `GetRuntimeWorktreeState(sessionID)`
  - `ReviewRuntimeWorktree(sessionID, includePatch, maxBytes)`
  - `MergeRuntimeWorktree(sessionID, commitMessage, removeWorktree)`
  - `ExitRuntimeWorktree(sessionID, action, discardChanges)`
- `FileService.workspacePath()` now resolves the active session's runtime worktree route through `ChatService.runtimeWorkspaces`.
- Workspace listing and metadata commands explicitly `cd` into the resolved workspace path.
- Timeline tool-result truncation keeps larger `WorktreeDiff` payloads so status/stat/patch can be inspected inline.

## Frontend
- Added `WorktreeReviewPanel.vue` under the workspace drawer.
- `WorkspacePanel.vue` mounts the review panel above the file browser and refreshes files after worktree changes.
- `TimelineEventItem.vue` classifies worktree tools separately and renders structured metadata, status, diff stat, and patch sections.
- Locales include `workspace.worktree.*` and worktree tool labels.

## Verification
- Targeted Go tests cover worktree state DTOs, FileService worktree path routing, and timeline result limits.
- Full `go test ./...`.
- Frontend `npm run build`.
- GitHub Codex review before merge.
