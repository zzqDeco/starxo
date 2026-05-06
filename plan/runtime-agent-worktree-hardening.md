# Runtime Agent Worktree Hardening

## Summary
- Branch: `feature/runtime-agent-worktree-hardening`, from latest `dev`.
- Harden Runtime V2 `Agent` tool worktree isolation and task output behavior.
- Keep explicit `EnterWorktree` / `ExitWorktree` as session-scoped tools.
- Change `Agent(isolation=worktree)` to use a context-scoped workspace override so background subagents do not hijack the parent session workspace.

## Backend
- Add isolated worktree creation to `runtimeWorkspaceManager`.
- Add a context-scoped runtime workspace override that takes precedence over session active worktree state.
- Normalize `Agent` input:
  - `subagent_type`: `general | code_writer | code_executor | file_manager`
  - `isolation`: `none | worktree`
- Include worktree path/branch in synchronous Agent tool output and background task output files.
- Update subagent instructions with the actual isolated workspace path and isolation semantics.

## Validation
- Targeted Go tests for isolated worktree routing, Agent input normalization, result formatting, and instruction metadata.
- Full `go test ./...`.
- Frontend build to ensure Wails/API changes do not regress the app surface.
