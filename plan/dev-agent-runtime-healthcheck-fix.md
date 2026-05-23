# Dev Agent Runtime HealthCheck Fix

## Summary
- Branch: `fix/dev-agent-runtime-healthcheck`.
- Fix the dev regression where a transient sandbox/bwrap command failure could mark the whole SSH connection disconnected while the TCP SSH session was still alive.
- Keep SSH liveness, active sandbox availability, UI state, and agent run state synchronized through one cleanup path.

## Implementation
- `SandboxService` health monitoring is generation-scoped.
- Each check first runs a raw SSH `echo ping` with a 5s timeout.
- A single SSH probe failure is logged only; two consecutive failures mark SSH disconnected.
- If raw SSH is alive, the active sandbox is inspected by runtime/workspace metadata. A missing sandbox only deactivates the sandbox and preserves SSH.
- Manual disconnect, reconnect, destroy, and health failure use shared helpers to stop the health monitor, clear service state, emit lifecycle events, and call the deactivation callback.
- `ChatService.UpdateSandbox(nil)` cancels running or starting agent runs, emits a clear `agent:error`, invalidates runners, and closes LSP servers.

## Verification
- Targeted Go tests cover transient SSH failure, consecutive SSH failure, missing sandbox deactivation, stale generation protection, and agent run cancellation on sandbox loss.
- Full verification should run:
  - `go test ./...`
  - `cd frontend && npm run build`
  - `/Users/zhaoziqian/go/bin/wails build -skipbindings -trimpath`

## Manual Regression
- Connect to `192.168.31.59`, activate a sandbox, run terminal file writes, and confirm workspace refresh.
- Run an agent file write/read through the local LLM endpoint.
- Wait for at least two health intervals and confirm SSH does not flip to disconnected during active work.
- Delete only the active sandbox workspace and confirm UI shows no active sandbox while SSH remains connected.
- Break SSH and confirm the UI moves to disconnected only after consecutive raw SSH probe failures.
