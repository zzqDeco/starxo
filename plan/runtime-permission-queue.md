# Runtime Permission Queue

## Summary
- Branch: `feature/runtime-permission-queue`, from latest `dev`.
- Complete Runtime V2 permission queue for risky runtime/MCP tools.
- Keep read-only trusted tools auto-allowed; route write/exec/task/destructive tools through one queue.
- Preserve `allow_once`, `allow_session`, and `deny`; persist `allow_session` grants with session data.

## Backend
- Add `ChatService.ListToolPermissionRequests(sessionID)` for frontend queue recovery after reload or session switch.
- Add `ChatService.ListToolPermissionGrants(sessionID)`, `RevokeToolPermissionGrant`, and `ClearToolPermissionGrants`.
- Resolve requests by deleting them from the pending map before notifying the waiter, so frontend refreshes do not see stale requests.
- Emit `runtime:permission_resolved`, `runtime:permission_canceled`, and `runtime:permission_grants_changed`.
- Keep permission checks centralized in `WrapMCPToolWithPermissionCheck` for runtime and MCP catalog entries.

## Frontend
- Replace the single pending permission modal state with a queue.
- Queue all permission requests, including inactive-session requests, and show the first request for the active session.
- Refresh pending requests on startup and session switch.
- Add a Settings / Permissions panel for pending approvals and persisted session grants.
- Allow grant revocation and clearing from the settings panel.

## Validation
- `go test ./internal/service -run 'Test.*Permission'`
- `go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -s -nopackage`
- `cd frontend && npm run build`
- Full `go test ./...` before PR.
