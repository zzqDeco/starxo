# AGENTS.md

Guidance for coding agents working in this repository.

## Project Overview

Starxo is an AI coding agent desktop app built with **Wails v2**:
- Backend: Go services
- Frontend: Vue 3 + TypeScript

The app connects to remote servers over SSH, manages Docker containers as sandbox environments, and runs LLM-powered agents that write/execute/manage code in those containers.

## Build and Development

From repository root:

```bash
wails dev
wails build
```

From `frontend/`:

```bash
npm install
npm run dev
npm run build
```

Note: there is no complete test infrastructure yet; rely on manual verification (`wails dev`) plus targeted Go tests where available.

## Tech Stack

- Go 1.24, Wails v2.11
- Vue 3.5 (`<script setup>`), TypeScript 5.7, Vite 6
- Naive UI, Pinia, xterm.js
- CloudWeGo Eino v0.8.8 (agent framework)
- LLM providers via eino-ext (OpenAI, DeepSeek, Ark, Ollama)
- MCP via `eino-ext/components/tool/mcp/officialmcp`
- i18n via `vue-i18n` (zh default, en fallback)

## Architecture

### Wails IPC and Events

- Go services are bound in `main.go`.
- TypeScript bindings are generated in `frontend/wailsjs/go/service/`.
- Frontend calls Go methods through generated bindings.
- Backend pushes state via Wails events (`wailsruntime.EventsEmit`).
- Agent events include `sessionId` and must be filtered by active session in frontend.

Key channels:
- `agent:timeline`
- `agent:interrupt`
- `agent:done`, `agent:error`
- `agent:plan`, `agent:mode_changed`
- `session:switched`
- `ssh:progress`, `ssh:connected`, `ssh:disconnected`
- `container:progress`, `container:ready`, `container:activated`, `container:deactivated`

### Deep Agent Pattern

The top-level `coding_agent` orchestrates 3 sub-agents via transfer tools:
- `code_writer`
- `code_executor`
- `file_manager`

Modes:
- `default` mode: direct `adk.Runner`
- `plan` mode: `planexecute.New()` with planner/replanner flow

Main files:
- `internal/agent/deep_agent.go`
- `internal/agent/codewriter.go`
- `internal/agent/codeexecutor.go`
- `internal/agent/filemanager.go`
- `internal/agent/prompts.go`

### Interrupt and Resume

- `ask_user` and `ask_choice` use `tool.StatefulInterrupt`.
- Frontend captures interrupt and resumes by calling runner resume APIs.
- Checkpoint state is kept in in-memory checkpoint store.

### Multi-session Model

- `ChatService` supports concurrent runs across different sessions.
- Isolation: `map[string]*SessionRun` by `sessionID`.
- Same-session concurrent runs are rejected.
- Runners are shared and expected to be concurrent-safe per run call.

### Sandbox Management

`internal/sandbox/manager.go` coordinates:
- SSH connection
- Docker lifecycle
- Remote command execution operator
- File transfer (SFTP + docker cp path)

### Data Storage

Persistent data location: `~/.starxo/`

- `config.json`
- `containers.json`
- `sessions/{id}/session.json`
- `sessions/{id}/session_data.json`
- legacy fallbacks: `messages.json`, `display.json`

### Frontend State

Pinia stores in `frontend/src/stores/`:
- `chatStore`
- `sessionStore`
- `connectionStore`
- `settingsStore`

### Context Management

`internal/context/engine.go` manages history, file context, and windowing.

## Code Conventions

- Wails-exposed services live in `internal/service/` with Wails-compatible signatures.
- Register tools via `internal/tools/registry.go`.
- Builtin tools are in `internal/tools/builtin.go`.
- Tool events are emitted by wrapper logic (`internal/agent/tool_wrapper.go`).
- Frontend alias `@` maps to `frontend/src/`.
- i18n keys live in `frontend/src/locales/zh.ts` and `frontend/src/locales/en.ts`.
- Logging: structured slog to stderr + daily rotating `logs/agent-YYYY-MM-DD.log`.

Do not manually edit generated Wails bindings unless regeneration is explicitly intended.

## Engineering Workflow

### Branching (trunk-based)

- Main branch: `master`
- Do not push directly to `master`; use PRs
- Branch naming:
  - `feature/<desc>`
  - `fix/<desc>`
  - `refactor/<desc>`
  - `docs/<desc>`
  - `test/<desc>`
  - `release/<version>`

### Commit Messages (Conventional Commits)

Format:

```text
<type>(<scope>): <subject>
```

Common types:
- `feat`
- `fix`
- `refactor`
- `docs`
- `test`
- `chore`
- `style`
- `perf`
- `ci`
- `build`

### Pull Request Expectations

1. Branch from `master`
2. Implement and commit with conventional format
3. Open PR to `master`
4. Include what changed, why, and how to verify
5. Require at least one review
6. Prefer squash merge for feature branches
7. Delete source branch after merge

## Documentation Sync Requirements

For each feature/fix/refactor:

1. Plan first in `plan/`
2. Implement selected scope
3. Sync docs:
   - update corresponding `doc/src/<file>.plan.md`
   - update relevant project-level docs under `doc/`
   - update `doc/files.index.plan.md` and `doc/files.coverage.plan.md` if file set changed and those overview files exist in the current worktree
   - update `README.md` and `README_CN.md` for user-visible behavior/config changes
4. Verify manually with `wails dev`

## Review Checklist for Agents

Before finalizing changes, verify:
- correctness and regression risk
- error handling paths
- naming clarity and maintainability
- architecture consistency with Wails event/session model
- docs updated for any behavior/interface change
- new tools/events are registered and documented
