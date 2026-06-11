# Dev App Regression Follow-ups

## Summary
- Branch: `fix/dev-app-regression-followups`.
- Fixes follow-up issues found by Computer Use app regression on `dev`.
- Scope:
  - Preserve workspace change notifications even when WorkspacePanel is not mounted.
  - Make active runtime execution session-bound; unbound sessions cannot reuse a previous session's sandbox.
  - Remove complexity-based automatic plan mode; only explicit plan mode selection/request enters plan mode.
  - Prevent `{count}` from leaking in code preview line-count metadata.

## Implementation
- Add a global frontend workspace dirty store, updated from `App.vue` on `workspace:changed`.
- Make `WorkspacePanel` consume the dirty revision on mount and while mounted, then refresh through the same guarded `refreshFiles` path.
- Add short retry after recent dirty events to avoid remote command/SFTP list races.
- Include optional `createdAt` in `WorkspaceChangedEvent`; terminal, upload and agent workspace events now send observable source/action/container metadata.
- Add `SessionService.GetSessionBoundContainerID(sessionID)` and validate `ChatService.SendMessage` against that session binding before starting a runtime turn.
- Session creation/switch callbacks now consistently notify listeners with the target session binding, including empty binding for detach-only transitions.
- Remove old complexity heuristic from default-mode user turns. Explicit plan-mode intent can still switch to plan mode.
- Render code preview line counts through a defensive computed label so i18n interpolation cannot leak `{count}`.

## Verification
- Go targeted tests cover session switch callbacks, unbound/mismatched sandbox guards and default-mode complex turns.
- Full checks:
  - `go test ./...`
  - `cd frontend && npm run build`
  - `/Users/zhaoziqian/go/bin/wails build -skipbindings -trimpath`
- Manual Computer Use regression should verify terminal-created files appear when Workspace is opened, unbound sessions cannot run agent commands, default mode does not trigger `ExitPlanMode`, and preview metadata shows real line counts.
