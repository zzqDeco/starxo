# Sandbox Diagnostics And Workspace Tools

## Summary
- Add a sandbox diagnostics panel for remote bwrap/Seatbelt runtime checks.
- Add a remote fix guide that copies privileged/security commands instead of running them automatically.
- Improve the workspace browser with active sandbox metadata, path copy, stable file sorting, and tmp cleanup.

## Backend
- `SettingsService.DiagnoseSandboxRuntime` returns structured runtime checks and fix suggestions.
- Linux diagnostics check bwrap, python3, venv creation, user namespace sysctls, AppArmor userns restriction, and a bwrap smoke command.
- macOS diagnostics check `sandbox-exec`, python3, and a minimal Seatbelt smoke command.
- `FileService.GetWorkspaceInfo` returns active sandbox/runtime/SSH/workspace size metadata.
- `FileService.CleanupSandboxTmp` only clears the active sandbox `tmp` directory; workspace files are never removed.

## Frontend
- `SandboxDiagnosticsPanel.vue` renders check status, command output details, and copyable fix commands.
- `SandboxConfig.vue` owns only sandbox configuration inputs and delegates runtime checks to the diagnostics panel.
- `WorkspacePanel.vue` shows active sandbox metadata, workspace path copy, tmp cleanup, and relative file-tree labels.

## Safety Rules
- `InstallSandboxRuntime` remains limited to normal Linux package installation.
- `sysctl`, AppArmor, and other host security changes are copy-only guide commands.
- Tmp cleanup is path-guarded on the backend and cannot target workspace files.

## Verification
- `go test ./...`
- `cd frontend && npm run build`
- Manual `wails dev`: run diagnostics against a Linux host, copy fix commands, create/activate a sandbox, browse files, and clean tmp.
