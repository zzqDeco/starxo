# Runtime Review Fixes

## Summary
- Branch: `fix/runtime-agent-review-findings`, from `feature/runtime-agent-worktree-hardening`.
- Close the six review findings around WebFetch/WebSearch network boundaries, worktree isolation, permission queue visibility, Wails listener lifecycle, and todo session isolation.
- PR target: `dev`; this branch supersedes draft PR #51.

## Backend
- Guard WebFetch/WebSearch URLs before request and before redirects.
- Allow only `http` / `https`, reject userinfo and missing hosts, and block localhost, private, link-local, multicast, unspecified, and cloud metadata addresses unless the runtime permission queue grants access.
- In worktree mode, validate file paths against the active worktree workspace instead of the parent workspace.
- Add `.starxo/` to Git `info/exclude` before creating worktrees, without editing tracked `.gitignore`.
- Scope todo state by sessionID while keeping global compatibility wrappers.

## Frontend
- Show the global permission queue so inactive-session requests are visible.
- Show session context on permission approvals and allow switching to the request's session.
- Unsubscribe Wails event listeners with the cleanup function returned by `EventsOn`.
- Guard permission settings refreshes against active-session switch races.

## Validation
- Targeted Go tests for web endpoint guards, worktree path boundaries, worktree exclude setup, and session-scoped todos.
- `go test ./...`.
- `cd frontend && npm run build`.
- `codex review --base origin/dev` must return no actionable findings before merge.
