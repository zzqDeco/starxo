# Agent Runtime V2 + ToolSearch

## Summary
- Branch: `feature/agent-runtime-v2`, based on latest `dev`.
- Goal: make ToolSearch a runtime-wide foundation instead of an MCP-only helper.
- Scope in this implementation: catalog metadata, always-loaded Runtime V2 core tools, permission-aware deferred search/load, permission queue, background task APIs, dynamic Agent tool, worktree isolation, persistent language-server-backed LSP with fallback, Skill/Web/Notebook deferred tools, and documentation/test coverage.
- Out of scope for this slice: full Claude Code parity, dedicated frontend runtime task panel, structured diff UI, writable LSP actions, and token-aware compaction.

## Runtime Surface
- `tool_search` is always visible and callable.
- Always-loaded core runtime tools:
  - `Bash`
  - `Read`
  - `Write`
  - `Edit`
  - `Glob`
  - `Grep`
  - `TaskOutput`
  - `TaskStop`
  - `ExitPlanMode`
  - `Agent`
- Runtime deferred tools:
  - `EnterWorktree`
  - `ExitWorktree`
  - `LSP`
  - `Skill`
  - `NotebookEdit`
  - `WebFetch`
  - `WebSearch`
- Legacy aliases remain searchable where applicable:
  - `shell_execute` -> `Bash`
  - `read_file` -> `Read`
  - `write_file` -> `Write`
  - `list_files` -> `Glob`
  - `str_replace_editor` -> `Edit`
- Deferred tools continue to use `DiscoveredToolRecord`; ToolSearch only writes records for newly discovered deferred tools.
- `ToolSearchOutput` now includes `loaded` and `pendingSources` in addition to the legacy `pending_mcp_servers`.

## Permission Model
- Catalog entries now carry runtime-wide metadata: source, class, kind, aliases, search hints, read-only hints, defer flags, and permission spec.
- Search and load checks apply to both MCP and non-MCP runtime entries.
- Plan mode only exposes read-only trusted tools through the visible surface. Writable tools such as `Bash`, `Write`, `Edit`, and `TaskStop` are not loaded in plan mode.
- ToolSearch visibility is not used as execution permission. Discovery only makes a tool eligible for schema injection; actual execution still passes through load/permission checks.

## Runtime Tasks
- Background `Bash` calls create managed runtime tasks.
- Background `Agent` calls use the same runtime task manager and output files.
- `ChatService` exposes:
  - `ListRuntimeTasks(sessionID)`
  - `ReadRuntimeTaskOutput(taskID, offset, limit)`
  - `StopRuntimeTask(taskID)`
  - `ApproveToolPermission(requestID, decision)`
  - `DenyToolPermission(requestID)`
- Task output is persisted under `~/.starxo/sessions/<session>/runtime-tasks/`.
- Large foreground tool results may be persisted under `~/.starxo/sessions/<session>/tool-results/`.

## Path And Execution Boundaries
- File/search/edit tools operate through the active remote sandbox operator.
- Workspace paths are normalized before execution and reject `..` traversal or workspace-external absolute paths.
- `Bash` runs in the active workspace and supports foreground or background execution.
- `Read` supports offset/limit by line range.
- `Grep` and `Glob` prefer remote `rg`/`find` behavior with stable sorted output.
- `EnterWorktree` creates a session-scoped git worktree under `.starxo/worktrees`; Runtime V2 file/search/edit/shell tools follow the active worktree until `ExitWorktree`.
- `ExitWorktree(action=remove)` refuses dirty worktrees unless `discard_changes=true`.

## Dynamic Agent And Deferred Tools
- `Agent` spawns a focused subagent for bounded tasks. It can run synchronously or in the background and can request `isolation=worktree`.
- `LSP` uses a persistent language server per session/workspace/language when available, with `rg`/`sed` fallback when the remote server is missing or the request lacks position data.
- Supported server commands are `gopls serve`, `typescript-language-server --stdio`, `pyright-langserver --stdio`, and `rust-analyzer`.
- `Skill` lists/reads `.starxo/skills` and `.claude/skills` prompts inside the workspace.
- `NotebookEdit` edits `.ipynb` cells through parsed JSON and the normal workspace guard.
- `WebFetch` and `WebSearch` are deferred runtime tools executed by the local app process.
- `WebSearch` supports `agent.webSearch` provider configuration, including DuckDuckGo fallback, custom HTTP providers, and a dedicated TinyFish Search API adapter.
- TinyFish provider alignment: `GET https://api.search.tinyfish.ai`, `X-API-Key` from `TINYFISH_API_KEY` by default, request parameters `query/location/language/page`, and response parsing from `results[].title/url/snippet`.

## Permission Queue
- Runtime V2 now emits `runtime:permission_request` for non-read-only trusted tools.
- Frontend approval supports deny, allow once, and allow session.
- `allow_session` grants are persisted in `SessionData.PermissionGrants`.
- Read-only trusted tools bypass the queue; no UI context fails closed for risky tools.

## Follow-Up Work
- Add frontend Runtime Tasks panel.
- Add diagnostics/install guidance for missing language servers.
- Add writable LSP operations such as rename, format, and code action apply behind the permission queue.
- Add token-aware compaction that preserves discovered tools, active tasks, permissions, todos, file read state, and diff summaries.
- Add structured diff UI for `Edit`/`Write`.
- Expand integration tests against the remote sandbox host.
