# Starxo - AI Coding Agent Desktop App

[中文文档](README_CN.md)

## About

Starxo is an AI coding agent desktop application built on the [CloudWeGo Eino](https://github.com/cloudwego/eino) framework. It connects to remote servers via SSH, manages lightweight bwrap/Seatbelt sandboxes as coding environments, and uses LLM-powered agents to autonomously write, execute, and manage code.

## Features

- **Claude Code-style Agent Runtime** — Eino v0.9 runtime with direct tools, `ToolSearch`, dynamic `Agent` subagents, task management, worktree isolation, Skill, and AGENTS.md context
- **Dual Execution Modes** — Default mode runs direct ReAct tool use without complexity-based auto-plan; Plan mode is explicit and narrows the same runtime loop to read/search/planning until `ExitPlanMode` approval
- **Interrupt/Resume** — `ask_user` / `ask_choice` tools pause agent execution for user input, state preserved via CheckPointStore
- **Sandbox Isolation** — SSH + lightweight OS sandbox runtime: Linux `bubblewrap` (`bwrap`) or macOS Seatbelt (`sandbox-exec`)
- **Sandbox Diagnostics** — Settings panel checks bwrap/Seatbelt, Python, venv, user namespaces, AppArmor restrictions, and returns copyable remote fix commands
- **Runtime V2 Tool Surface** — Always-visible `ToolSearch`, direct file/search/edit/shell tools, dynamic `Agent` delegation, worktree isolation, deferred LSP/Skill/Web/Notebook tools, and managed background task output
- **Runtime Tool Configuration** — Settings for WebSearch/TinyFish diagnostics and persistent LSP server mappings, including custom language server commands
- **Tool Permissions** — Risky runtime and MCP tool calls enter a queued approval UI with deny, allow once, session grants, and grant management in Settings
- **MCP Protocol** — Model Context Protocol tool extension support (stdio/SSE transports)
- **Multi-LLM Support** — OpenAI / DeepSeek / Volcengine Ark / Ollama
- **Bilingual UI** — Chinese/English (vue-i18n)
- **Real-time Event Stream** — Unified `agent:timeline` event stream via Wails Events for live agent activity display, all events tagged with `sessionId` for multi-session isolation
- **Multi-Session Parallel Execution** — Multiple sessions can run agents concurrently; switching sessions does not cancel background agents, with full state restore on switch. Agent execution is bound to the active session's sandbox and cannot reuse another session's active sandbox.
- **Session Persistence** — Full session management with unified session data (messages + timeline + streaming state)
- **File Transfer** — Upload/download support via SFTP directly into each persistent sandbox workspace, with workspace metadata, path copy, and tmp cleanup
- **Developer Workbench UI** — Dense dark workbench with command palette, session rail, centered execution canvas, lifecycle-aware workspace drawer, persistent runtime dock, sandbox command terminal, and composer-level mode controls

## Tech Stack

### Backend

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.24 | Primary language |
| Wails | v2.11 | Desktop framework (Go + WebView) |
| CloudWeGo Eino | v0.9.0-beta.1 | Agent framework (ADK, ChatModelAgent, Runner, ToolSearch/Skill/Reduction/Summarization middleware) |
| eino-ext | - | LLM Providers (OpenAI/Ark/Ollama) + MCP + Commandline |
| golang.org/x/crypto | - | SSH connections |
| pkg/sftp | v1.13 | SFTP file transfer |
| MCP Go SDK | v1.4 | Model Context Protocol |

### Frontend

| Technology | Version | Purpose |
|------------|---------|---------|
| Vue | 3.5 | UI framework (`<script setup>` + TypeScript) |
| TypeScript | 5.7 | Type system |
| Vite | 6.2 | Build tool |
| Naive UI | 2.41 | Component library (dark theme) |
| Pinia | 2.3 | State management |
| xterm.js | 5.5 | Terminal emulator |
| markdown-it + highlight.js | 14.1 / 11.11 | Markdown rendering + syntax highlighting |
| vue-i18n | 12 | Internationalization (zh/en) |

## Project Structure

```
starxo/
├── main.go                          # Entry point, Wails init, service bindings
├── app.go                           # App struct, service assembly, lifecycle management
├── wails.json                       # Wails project configuration
├── go.mod / go.sum                  # Go dependency management
│
├── internal/
│   ├── agent/                       # AI Agent construction & configuration
│   │   ├── runtime_agent.go         #   Eino v0.9 ChatModelAgent runtime builder
│   │   ├── runtime_behavior.go      #   Current-objective behavior middleware
│   │   ├── deep_agent.go            #   Legacy deep-transfer fallback agent builder
│   │   ├── subagents.go             #   Dynamic runtime subagent registry
│   │   ├── eino_v09_context.go      #   Skill, AGENTS.md, reduction, summarization middleware
│   │   ├── runner.go                #   Runner builders (default + plan mode)
│   │   ├── prompts.go               #   System prompts for all agents
│   │   ├── codewriter.go            #   Legacy transfer fallback code_writer sub-agent
│   │   ├── codeexecutor.go          #   Legacy transfer fallback code_executor sub-agent
│   │   ├── filemanager.go           #   Legacy transfer fallback file_manager sub-agent
│   │   ├── context.go               #   AgentContext (workspace, sandbox, SSH info)
│   │   ├── plan.go                  #   Plan/Step type definitions
│   │   ├── plan_wrapper.go          #   Plan state persistence + event emission
│   │   └── tool_wrapper.go          #   eventEmittingTool wrapper
│   │
│   ├── service/                     # Wails-bound services (frontend API)
│   │   ├── chat.go                  #   ChatService: per-session agent lifecycle (SessionRun), messaging, streaming
│   │   ├── runtime_agents_build.go  #   Runtime agent builder selection
│   │   ├── runtime_objective.go     #   Current-objective prompt/history sidecar
│   │   ├── runtime_context_compact.go # Runtime V2 context compact state
│   │   ├── runtime_agent_tool.go    #   Runtime V2 dynamic Agent tool
│   │   ├── runtime_lsp_manager.go   #   Runtime V2 persistent language server manager
│   │   ├── runtime_workspaces.go    #   Runtime V2 session-scoped worktree workspace manager
│   │   ├── runtime_web_tools.go     #   Runtime V2 WebFetch/WebSearch tool implementations
│   │   ├── websearch_diagnostics.go #   WebSearch/TinyFish configuration diagnostics
│   │   ├── sandbox_svc.go           #   SandboxService: connect/disconnect/reconnect, health monitor (RWMutex)
│   │   ├── session_svc.go           #   SessionService: session CRUD, multi-session state coordination
│   │   ├── settings_svc.go         #   SettingsService: config management, connection testing
│   │   ├── file_svc.go              #   FileService: upload/download/preview
│   │   ├── container_svc.go         #   ContainerService: sandbox registry lifecycle (compat name)
│   │   └── events.go                #   Event DTO definitions
│   │
│   ├── sandbox/                     # Remote sandbox management
│   │   ├── manager.go               #   SandboxManager top-level orchestrator
│   │   ├── ssh.go                   #   SSH connection management
│   │   ├── runtime.go               #   bwrap/Seatbelt runtime management
│   │   ├── operator.go              #   RemoteOperator (commandline.Operator impl)
│   │   └── transfer.go              #   File transfer (SFTP into sandbox workspace)
│   │
│   ├── tools/                       # Agent tool definitions
│   │   ├── registry.go              #   ToolRegistry central registry
│   │   ├── builtin.go               #   Built-in tool registration
│   │   ├── runtime_tools.go         #   Runtime V2 tools: Bash/Read/Write/Edit/Glob/Grep/tasks
│   │   ├── runtime_deferred_tools.go #  Runtime V2 deferred tools: LSP/Skill/Notebook/Web metadata
│   │   ├── tool_search.go           #   Runtime-wide deferred tool discovery
│   │   ├── mcp.go                   #   MCP server connection + tool loading
│   │   ├── followup.go              #   ask_user interrupt tool
│   │   ├── choice.go                #   ask_choice interrupt tool
│   │   ├── runtime_objective.go     #   Current-objective context guard for ask tools
│   │   ├── todos.go                 #   write_todos / update_todo task tools
│   │   ├── notify.go                #   notify_user notification tool
│   │   └── custom.go                #   Custom tool helper
│   │
│   ├── config/                      # Configuration management
│   ├── context/                     # Context engine (history, file context, windowing)
│   ├── llm/                         # LLM provider factory + optional agentic beta adapters
│   ├── model/                       # Data models (Message, Session, sandbox registry)
│   ├── storage/                     # Persistence (sessions, sandboxes)
│   ├── store/                       # CheckPointStore (interrupt/resume state)
│   └── logger/                      # Structured logging + Eino callbacks
│
├── frontend/
│   ├── package.json                 # Frontend dependencies
│   ├── vite.config.ts               # Vite config (@ path alias)
│   ├── tsconfig.json                # TypeScript configuration
│   └── src/
│       ├── main.ts                  # Vue app entry
│       ├── App.vue                  # Root component: dark theme, Wails event listeners
│       ├── style.css                # Global styles
│       ├── components/
│       │   ├── chat/                #   Chat panel, message bubbles, interrupt dialog, floating task rail, input area
│       │   ├── layout/              #   Main layout, header, sidebar, task rail components
│       │   ├── settings/            #   Settings panel (SSH/Sandbox/LLM/MCP)
│       │   ├── files/               #   Workspace drawer, file tree, code preview, file transfer
│       │   ├── containers/          #   Sandbox panel + persistent dock
│       │   ├── status/              #   Agent status, connection status
│       │   └── terminal/            #   Terminal component (not a default main-view entry)
│       ├── stores/                  #   Pinia state management
│       ├── types/                   #   TypeScript type definitions
│       ├── composables/             #   Vue composables
│       └── locales/                 #   i18n language packs (zh/en)
│
├── build/                           # Platform build assets (Windows NSIS / macOS plist)
├── .github/workflows/               # GitHub Actions CI/CD workflows
├── doc/                             # Technical documentation
├── plan/                            # Future roadmap documents
└── logs/                            # Runtime logs (agent-YYYY-MM-DD.log)
```

## Getting Started

### Prerequisites

- **Go** >= 1.24
- **Node.js** >= 18
- **Wails CLI** v2 — Install: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Remote Server** with SSH access plus either Linux `bubblewrap`/`python3` or macOS `sandbox-exec`/`python3`

### Development

```bash
wails dev
```

Launches with Vite HMR for frontend hot reload and Go backend hot reload. Frontend dev server URL is auto-detected; Go dev server runs at `http://localhost:34115`.

### Agent Runtime V2

The top-level agent now runs on Eino `v0.9.0-beta.1` and receives a smaller always-loaded runtime tool surface. Eino's dynamic `tool_search` middleware exposes deferred tools on demand while Starxo keeps catalog metadata, plan-mode filtering, permission checks, and discovered-tool persistence. Core tools include `Read`, `Edit`, `Write`, `Bash`, `Glob`, `Grep`, `TaskOutput`, `TaskStop`, `ExitPlanMode`, and `Agent`; legacy names such as `read_file` and `shell_execute` remain aliases.

The fixed `transfer_to_agent` subagent path is no longer the default runtime. The top-level agent is an Eino `ChatModelAgent` ReAct loop with a pinned current-objective sidecar, so new standalone requests do not resume unrelated old tasks. `Agent` is the delegation entry point and resolves `subagent_type` through `agent.runtime.subagents`. Built-ins include `general`, `code_writer`, `code_executor`, `file_manager`, and `reviewer`; each definition can constrain allowed tools, default isolation, instructions, and whether background execution is allowed. Omitting `subagent_type` forks the current agent context; setting it creates a fresh worker constrained by that definition. The previous deep-transfer implementation remains behind `agent.runtime.enableBuiltinDeepTransferFallback` for debugging only.

Deferred runtime tools currently include `EnterWorktree`, `ExitWorktree`, `WorktreeDiff`, `WorktreeMerge`, `LSP`, `LSPEdit`, `Skill`, `NotebookEdit`, `WebFetch`, and `WebSearch`. `LSP` uses a persistent language server per session/workspace/language when the remote sandbox has one installed (`gopls`, `typescript-language-server`, `pyright-langserver`, or `rust-analyzer`), and falls back to `rg`/`sed` when it cannot use a server. `LSPEdit` exposes writable language-server rename/format operations through the permission queue. `Agent` can run focused subagents synchronously or in the background, and can request context-scoped worktree isolation for bounded tasks without switching the parent session workspace.

Runtime worktrees now have a review/merge loop: `WorktreeDiff` reports active worktree status, diff stat, and optional patch content; `WorktreeMerge` commits active worktree changes, merges them back into the original sandbox workspace, optionally removes the worktree, and restores the parent workspace. Merge refuses to run while the parent workspace has uncommitted changes, and merge conflicts return structured conflict files/recovery guidance after aborting the parent merge state. The workspace drawer also shows the active session worktree, can review/copy/merge it with confirmation, and the file browser follows the active worktree path.

File-changing runtime tools now emit structured diff metadata. `Write` and `Edit` timeline events show created/updated state, replacement counts, `+/-` line counts, bytes, and bounded patch previews instead of requiring users to inspect raw JSON.

`WebSearch` defaults to DuckDuckGo HTML search and can be redirected through `agent.webSearch.providers`. `type: "tinyfish"` is a dedicated TinyFish Search API adapter for `GET https://api.search.tinyfish.ai` with `X-API-Key` read from `TINYFISH_API_KEY` by default; it supports TinyFish `query`, `location`, `language`, and `page` parameters and parses `results[].title/url/snippet`. `type: "http"` remains available for custom GET/POST providers with headers, body templates, and JSON path extraction.

Settings include static WebSearch diagnostics and an explicit WebSearch smoke test. The smoke test runs only when clicked, then reports provider, URL, duration, and compact results.

`WebFetch` and `WebSearch` only execute `http`/`https` requests. Empty hosts, URL userinfo, unsafe local/private/link-local/multicast/unspecified targets, IPv6 ULA/link-local targets, and `169.254.169.254` are blocked by default; non-public endpoints require an explicit runtime permission grant and redirects are validated before they are followed.

Plan mode keeps only read-only trusted tools visible. Writable tools and shell execution are hidden until the plan is approved.

Risky tool calls are routed through the Runtime V2 permission queue. The desktop UI can deny, allow once, or allow the tool for the current session; session grants are persisted with the session data and can be reviewed or revoked from Settings / Permissions.

Background Bash and Agent jobs can be inspected from the Runtime Tasks panel. The panel lists tasks for the active session, refreshes task status/output, supports copying output, and can stop running tasks through the Runtime V2 task APIs.

Long sessions use token-aware context compaction. Starxo keeps recent turns in full and injects a compact runtime summary that preserves discovered tools, session permission grants, task output pointers, file read ranges, recent edit summaries, todos, plan state, and active worktree routing. Eino v0.9 summarization/reduction middleware stores large tool results under `.starxo/tool-results`, while Starxo's sidecar compact state keeps the runtime bookkeeping that must survive summarization. Full message history remains persisted in `session_data.json`.

Skill middleware loads workspace-local `.starxo/skills/<name>/SKILL.md` and `.claude/skills/<name>/SKILL.md`. AGENTS.md middleware reads `AGENTS.md` and `.starxo/AGENTS.md` as transient runtime context without writing those instructions into persisted chat history.

The experimental Eino agentic provider path is available through `agent.runtime.agenticProtocol=agentic_openai|agentic_ark|auto`, but the default remains the existing `*schema.Message` path (`agenticProtocol=off`). If agentic provider setup fails, Starxo logs the failure and falls back to the Message runtime. `agent.runtime.toolSearchMode=model_native` currently falls back to client-side search until Starxo can pre-grant model-native deferred tool discovery safely.

### Production Build

```bash
wails build
```

Output goes to `build/bin/`.

Starxo uses Wails platform-native shell settings at build time. macOS uses a unified hidden titlebar and system appearance, Windows follows the system theme with Mica where available, and Linux uses a conservative GTK/WebKit fallback. The Vue UI follows system light/dark mode and applies platform-specific design tokens. macOS builds use bundle id `com.starxo.app` and declare Local Network access because Starxo connects to LAN sandbox hosts over SSH; allow the system prompt if you use `192.168.x.x` or `.local` remotes. If a signed macOS app reports `no route to host` for a LAN address while Terminal SSH still works, use the in-app SSH diagnostics card, enable Starxo in System Settings > Privacy & Security > Local Network, then quit and reopen the app. For stale development permissions, copy and run `tccutil reset LocalNetwork com.starxo.app` manually, reopen Starxo, and allow the Local Network prompt again. If you previously tested an older `com.wails.starxo` build, reset the old Local Network permission or allow the prompt again for the new bundle identity.

### Tagged Release

GitHub Actions publishes desktop packages when a `vX.Y.Z` tag is pushed to a commit reachable from `master`.

```bash
git checkout master
git pull origin master
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds ad-hoc signed macOS packages plus unsigned Windows and Linux packages, checks basic platform bundle resources, uploads them to the GitHub Release, and generates `SHA256SUMS.txt`.

After the workflow finishes, verify:

- The Release page is public and contains the macOS zip, Windows exe, Windows installer, Linux tarball, and `SHA256SUMS.txt`.
- `SHA256SUMS.txt` hashes match the downloaded assets.
- Each platform package launches at least once; macOS is ad-hoc signed but not notarized, and Windows is unsigned, so system security prompts are expected for v1.
- Settings open correctly, SSH connection testing works, and sandbox runtime detection reports the expected remote runtime state.
- On a Linux remote, creating a sandbox can write to the workspace and `network=false` blocks outbound network access.

### Frontend-Only Development

```bash
cd frontend
npm install
npm run dev
```

## Configuration

App configuration is stored at `~/.starxo/config.json`:

| Block | Description |
|-------|-------------|
| `ssh` | SSH connection (host, port, username, auth method) |
| `sandbox` | Sandbox runtime settings (runtime, workspace root, network, process limits, Python bootstrap) |
| `llm` | LLM settings (provider, model, API key, base URL) |
| `mcp` | MCP server settings (command, args, env vars, transport) |
| `agent` | Agent settings (mode, system prompt, workspace directory) |

### Dev-Only Deferred Surface Flags

These environment variables are for development and debugging only:

- `STARXO_ENABLE_DEFERRED_SURFACE_DEBUG_API=1`
  - Enables the Wails deferred-surface debug API
  - Startup-latched: restart the app after changing it
- `STARXO_ENABLE_DEV_DEFERRED_BUILTIN_SAMPLE=1`
  - Registers the `dev_deferred_builtin_sample` top-level deferred builtin sample
  - Startup-latched: restart the app after changing it

Both flags are disabled by default and are not intended as production-facing controls.

### Sandbox Diagnostics And Fix Guide

The Sandbox settings tab can run a full remote diagnostics pass before saving settings. Linux checks cover `bwrap`, `python3`, Python venv creation, user namespace sysctls, AppArmor's unprivileged user namespace restriction, and a bwrap smoke command. macOS checks cover `sandbox-exec`, `python3`, and a minimal Seatbelt smoke command.

Normal Linux package dependencies can be installed with the Install runtime button. Host security changes such as `sysctl` or AppArmor adjustments are never executed automatically; Starxo only displays copyable commands so the operator can review and run them manually.

Sandbox creation reports each setup step, including Python venv creation, pip upgrade, and package installation. Python bootstrap commands obey `commandTimeoutSec`; pip failures include remote network, index, and proxy guidance, and incomplete sandbox directories are cleaned up best-effort. The runtime terminal executes one command at a time in the active sandbox workspace and is disabled when no sandbox is active.

Runtime health checks separate SSH liveness from active sandbox availability. A transient sandbox command failure no longer drops the SSH connection; if the active sandbox disappears, Starxo deactivates it and stops any affected agent run with a clear error.

## Data Storage

All persistent data is stored under `~/.starxo/`:

```
~/.starxo/
├── config.json                # App configuration
├── sandboxes.json             # Sandbox registry
└── sessions/
    └── {session-id}/
        ├── session.json       # Session metadata
        ├── session_data.json  # Unified session data (messages + display + streaming)
        ├── messages.json      # Conversation history - legacy fallback
        └── display.json       # Rich display data - legacy fallback
```

## Documentation

- **Docs Guide**: `doc/README.md` — Documentation structure and sync rules
- **Technical Docs**: `doc/src/` — Per-file technical notes for the current codebase
- **Project Docs**: `doc/` — Overview, research, and non-file-specific technical documents
- **Implementation Plans**: `plan/` — Active change plans
- **Contributor Workflow**: `AGENTS.md` — Primary branch / PR / doc sync rules
- **Additional Agent Notes**: `CLAUDE.md` — Supplemental architecture and coding guidance

## Development Workflow

- `master` is the trunk branch.
- `dev` is the development buffer branch.
- Topic branches should be created from `dev` and merged into `dev` first.
- `dev` is merged into `master` after integration is verified.
