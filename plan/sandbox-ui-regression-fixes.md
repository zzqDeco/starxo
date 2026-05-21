# Sandbox UI Regression Fixes

## Summary
- Fix sandbox creation hanging on Python bootstrap by splitting setup into observable steps with per-command timeout handling.
- Keep the workspace drawer synchronized with sandbox activation, deactivation, destruction, SSH disconnect, and session switches.
- Make the runtime terminal a non-interactive command runner for the active sandbox workspace.

## Backend
- `RemoteRuntimeManager.CreateSandbox` reports setup progress for runtime check, directory creation, venv creation, pip upgrade, package install, and Seatbelt profile writing.
- Setup commands are wrapped with remote `timeout --kill-after=5s` when available and also use a local context deadline derived from `commandTimeoutSec`.
- Python package install failures include a network/pip/proxy hint, and incomplete sandbox roots are removed best-effort before returning the error.
- `SandboxService.RunTerminalCommand` executes a user-submitted shell command through the active sandbox operator and returns stdout/stderr/exit code.
- `ContainerService.DestroyContainer` emits `container:destroyed` so UI panels can clear stale active sandbox state.

## Frontend
- `WorkspacePanel` listens to sandbox lifecycle events and clears file lists, selected paths, preview content, and search state when the active sandbox goes away.
- Workspace refresh and preview requests use request IDs so stale async responses cannot repopulate a destroyed sandbox.
- `TerminalPanel` adds an explicit command input bar that is disabled until SSH and an active sandbox are available.

## Verification
- Added Go tests for sandbox bootstrap progress/timeout cleanup and terminal command error boundaries.
- Manual regression should cover SSH connect, sandbox create, workspace drawer refresh/clear, terminal `pwd`/`echo`/`cat`, agent file IO, and sandbox destroy on `192.168.31.59`.
