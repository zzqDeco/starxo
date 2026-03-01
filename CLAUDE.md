# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Starxo is an AI coding agent desktop app built with **Wails v2** (Go backend + Vue 3 frontend). It connects to remote servers via SSH, manages Docker containers as sandboxed coding environments, and uses LLM-powered agents to write/execute/manage code inside those containers.

## Build & Development Commands

```bash
wails dev          # Live dev mode (Vite HMR + Go hot reload)
wails build        # Production build -> build/bin/
```

Frontend only (from `frontend/` directory):
```bash
npm install        # Install frontend dependencies
npm run dev        # Vite dev server only
npm run build      # vue-tsc --noEmit && vite build
```

No test infrastructure exists yet.

## Tech Stack

- **Backend:** Go 1.24, Wails v2.11, CloudWeGo Eino v0.7 (agent framework)
- **Frontend:** Vue 3.5 (`<script setup>` + TypeScript 5.7), Vite 6, Naive UI, Pinia, xterm.js
- **LLM providers:** OpenAI, DeepSeek, Volcengine Ark, Ollama (via eino-ext)
- **MCP:** Model Context Protocol via `eino-ext/components/tool/mcp/officialmcp`
- **i18n:** vue-i18n (zh default, en fallback)

## Architecture

### IPC: Wails Bindings + Events

Go services are bound in `main.go` and auto-generate TypeScript bindings in `frontend/wailsjs/go/service/`. Frontend calls Go directly via these bindings (e.g., `ChatService.SendMessage()`).

Backend-to-frontend communication uses Wails events (`wailsruntime.EventsEmit`). Key channels:
- `agent:timeline` — unified stream for all agent activity (messages, tool calls, file transfers)
- `agent:interrupt` — pauses agent for user input (follow-up questions, choices)
- `agent:done`, `agent:error` — agent lifecycle
- `agent:plan`, `agent:mode_changed` — plan execution state
- `sandbox:progress`, `sandbox:ready`, `sandbox:disconnected` — sandbox lifecycle

### Agent Architecture (Deep Agent Pattern)

A deep agent (`coding_agent`) orchestrates 3 sub-agents via Eino ADK's `transfer_to_agent`:
- **code_writer** — reads/writes/edits code (str_replace_editor, read_file, list_files)
- **code_executor** — runs Python and shell commands (python_execute, shell_execute)
- **file_manager** — bulk file operations, non-code content

Two modes: **default** (direct `adk.Runner`) and **plan** (wrapped in `planexecute.New()` with Planner/Replanner).

Agent construction: `internal/agent/deep_agent.go` (orchestrator), individual sub-agents in `codewriter.go`, `codeexecutor.go`, `filemanager.go`. Prompts centralized in `prompts.go`.

### Interrupt/Resume Pattern

Tools like `ask_user` and `ask_choice` use `tool.StatefulInterrupt` to pause execution. Frontend shows a dialog, user responds, and `ResumeWithAnswer()`/`ResumeWithChoice()` continues via `runner.ResumeWithParams()`. State preserved in `InMemoryStore` (implements `compose.CheckPointStore`).

### Sandbox Management

`internal/sandbox/manager.go` coordinates: SSH connection → Docker container management → `RemoteOperator` (implements `commandline.Operator` over SSH+Docker exec) + `FileTransfer` (SFTP + docker cp). Small files use base64-encoded docker exec; large files (>64KB) use SFTP.

### Data Storage

All persistent data at `~/.starxo/`:
- `config.json` — app settings (SSH, Docker, LLM, MCP configs)
- `containers.json` — container registry
- `sessions/{id}/` — session.json, messages.json, display.json

### Frontend State

Pinia stores in `frontend/src/stores/`: `chatStore` (messages, streaming, interrupts, plan, timeline), `sessionStore`, `connectionStore`, `settingsStore`.

### Context Management

`internal/context/engine.go` orchestrates conversation history + file context + message windowing (default: 20 messages, 4000 chars/message truncation).

## Key Conventions

- Go services exposed to frontend live in `internal/service/` — each has Wails-compatible method signatures
- All agent tools are registered via `internal/tools/registry.go`; builtin tools in `builtin.go`
- Agent tools are wrapped with `eventEmittingTool` (`internal/agent/tool_wrapper.go`) to emit timeline events to frontend
- Frontend uses `@` path alias mapped to `frontend/src/`
- i18n keys defined in `frontend/src/locales/zh.ts` (primary) and `en.ts`
- Logging: structured slog to stderr + daily-rotated `logs/agent-YYYY-MM-DD.log`; Eino global callbacks auto-log all model/tool calls

## Engineering Conventions

### Branch Management (Trunk-based)

`master` is the main branch, always kept in a buildable state. Direct pushes to `master` are forbidden — all changes enter via pull request.

Branch naming:
- `feature/<desc>` — new features (e.g., `feature/plugin-system`)
- `fix/<desc>` — bug fixes (e.g., `fix/ssh-reconnect-timeout`)
- `refactor/<desc>` — code refactoring without behavior change
- `docs/<desc>` — documentation-only changes
- `test/<desc>` — test infrastructure or test-only changes
- `release/<version>` — release preparation (e.g., `release/0.2.0`)

Rules:
- Feature branches are created from `master` and merged back to `master`
- Delete branches after merge
- Keep branches short-lived (days, not weeks)

### Commit Message Format (Conventional Commits)

```
<type>(<scope>): <subject>

[optional body]

[optional footer(s)]
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `style`, `perf`, `ci`, `build`

Scopes:
- Backend: `agent`, `service`, `sandbox`, `tools`, `config`, `context`, `llm`, `model`, `storage`, `logger`
- Frontend: `chat`, `settings`, `session`, `terminal`, `layout`, `files`, `stores`, `types`, `i18n`
- Root: `build`, `deps`

Examples:
- `feat(agent): add retry logic to sub-agent transfers`
- `fix(sandbox): handle SSH connection timeout gracefully`
- `refactor(service): extract event emission into helper`
- `docs: update README with project structure`

### Pull Request Workflow

1. Create a feature branch from `master`
2. Make changes, commit with conventional commit messages
3. Push branch and open PR against `master`
4. PR title follows the same conventional commit format
5. PR description includes: what changed, why, how to test
6. At least one approval required before merge
7. Squash merge preferred for feature branches; merge commit for release branches
8. Delete source branch after merge

### Code Review Expectations

- All PRs require review before merge
- Reviewer checks: correctness, error handling, naming, test coverage intent
- Backend (Go): follow standard Go conventions (gofmt, effective Go)
- Frontend (Vue/TS): follow `<script setup>` pattern, use composables for shared logic
- New Wails-bound service methods must have doc comments explaining IPC usage
- New agent tools must be registered in `registry.go` and documented in `prompts.go`
- New Wails events must be documented in the Architecture section above

### Testing Conventions (To Be Established)

- Go tests: `*_test.go` files alongside source, use `testing` + `testify`
- Frontend tests: Vitest for unit tests, Playwright for E2E
- Test file naming: `<source>_test.go` (Go), `<source>.test.ts` (TS)
- No test infrastructure exists yet; see `plan/testing-strategy.md` for the roadmap
