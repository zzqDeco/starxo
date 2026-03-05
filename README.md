# Starxo - AI Coding Agent Desktop App

[中文文档](README_CN.md)

## About

Starxo is an AI coding agent desktop application built on the [CloudWeGo Eino](https://github.com/cloudwego/eino) framework. It connects to remote servers via SSH, manages Docker containers as sandboxed coding environments, and uses LLM-powered agents to autonomously write, execute, and manage code.

## Features

- **Deep Agent Architecture** — Orchestrator agent delegates to 3 specialized sub-agents (code_writer / code_executor / file_manager) via `transfer_to_agent`
- **Dual Execution Modes** — Default mode (direct execution) + Plan mode (Planner/Replanner structured execution)
- **Interrupt/Resume** — `ask_user` / `ask_choice` tools pause agent execution for user input, state preserved via CheckPointStore
- **Sandbox Isolation** — SSH + Docker container environment with full container lifecycle management (create/reconnect/stop/destroy)
- **MCP Protocol** — Model Context Protocol tool extension support (stdio/SSE transports)
- **Multi-LLM Support** — OpenAI / DeepSeek / Volcengine Ark / Ollama
- **Bilingual UI** — Chinese/English (vue-i18n)
- **Real-time Event Stream** — Unified `agent:timeline` event stream via Wails Events for live agent activity display, all events tagged with `sessionId` for multi-session isolation
- **Multi-Session Parallel Execution** — Multiple sessions can run agents concurrently; switching sessions does not cancel background agents, with full state restore on switch
- **Session Persistence** — Full session management with unified session data (messages + timeline + streaming state)
- **File Transfer** — Upload/download support; small files via base64 + docker exec, large files via SFTP + docker cp
- **Refined Workbench UI** — Maximized window by default; long workspace drawer from top-right trigger; persistent container dock; floating task rail above input

## Tech Stack

### Backend

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.24 | Primary language |
| Wails | v2.11 | Desktop framework (Go + WebView) |
| CloudWeGo Eino | v0.7 | Agent framework (ADK, Runner, Deep Agent, PlanExecute) |
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
│   │   ├── deep_agent.go            #   Deep Agent orchestrator (3 sub-agents)
│   │   ├── runner.go                #   Runner builders (default + plan mode)
│   │   ├── prompts.go               #   System prompts for all agents
│   │   ├── codewriter.go            #   code_writer sub-agent
│   │   ├── codeexecutor.go          #   code_executor sub-agent
│   │   ├── filemanager.go           #   file_manager sub-agent
│   │   ├── context.go               #   AgentContext (workspace, container, SSH info)
│   │   ├── plan.go                  #   Plan/Step type definitions
│   │   ├── plan_wrapper.go          #   Plan state persistence + event emission
│   │   └── tool_wrapper.go          #   eventEmittingTool wrapper
│   │
│   ├── service/                     # Wails-bound services (frontend API)
│   │   ├── chat.go                  #   ChatService: per-session agent lifecycle (SessionRun), messaging, streaming
│   │   ├── sandbox_svc.go           #   SandboxService: connect/disconnect/reconnect, health monitor (RWMutex)
│   │   ├── session_svc.go           #   SessionService: session CRUD, multi-session state coordination
│   │   ├── settings_svc.go         #   SettingsService: config management, connection testing
│   │   ├── file_svc.go              #   FileService: upload/download/preview
│   │   ├── container_svc.go         #   ContainerService: container lifecycle
│   │   └── events.go                #   Event DTO definitions
│   │
│   ├── sandbox/                     # Remote sandbox management
│   │   ├── manager.go               #   SandboxManager top-level orchestrator
│   │   ├── ssh.go                   #   SSH connection management
│   │   ├── docker.go                #   Remote Docker management
│   │   ├── operator.go              #   RemoteOperator (commandline.Operator impl)
│   │   ├── transfer.go              #   File transfer (SFTP + docker cp)
│   │   └── setup.go                 #   Environment setup (Docker install, image pull)
│   │
│   ├── tools/                       # Agent tool definitions
│   │   ├── registry.go              #   ToolRegistry central registry
│   │   ├── builtin.go               #   Built-in tool registration
│   │   ├── mcp.go                   #   MCP server connection + tool loading
│   │   ├── followup.go              #   ask_user interrupt tool
│   │   ├── choice.go                #   ask_choice interrupt tool
│   │   ├── todos.go                 #   write_todos / update_todo task tools
│   │   ├── notify.go                #   notify_user notification tool
│   │   └── custom.go                #   Custom tool helper
│   │
│   ├── config/                      # Configuration management
│   ├── context/                     # Context engine (history, file context, windowing)
│   ├── llm/                         # LLM provider factory
│   ├── model/                       # Data models (Message, Session, Container)
│   ├── storage/                     # Persistence (sessions, containers)
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
│       │   ├── settings/            #   Settings panel (SSH/Docker/LLM/MCP)
│       │   ├── files/               #   Workspace drawer, file tree, code preview, file transfer
│       │   ├── containers/          #   Container panel + persistent dock
│       │   ├── status/              #   Agent status, connection status
│       │   └── terminal/            #   Terminal component (not a default main-view entry)
│       ├── stores/                  #   Pinia state management
│       ├── types/                   #   TypeScript type definitions
│       ├── composables/             #   Vue composables
│       └── locales/                 #   i18n language packs (zh/en)
│
├── build/                           # Platform build assets (Windows NSIS / macOS plist)
├── doc/                             # Technical documentation
├── plan/                            # Future roadmap documents
└── logs/                            # Runtime logs (agent-YYYY-MM-DD.log)
```

## Getting Started

### Prerequisites

- **Go** >= 1.24
- **Node.js** >= 18
- **Wails CLI** v2 — Install: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Remote Server** with SSH access + Docker installed (or permission to install)

### Development

```bash
wails dev
```

Launches with Vite HMR for frontend hot reload and Go backend hot reload. Frontend dev server URL is auto-detected; Go dev server runs at `http://localhost:34115`.

### Production Build

```bash
wails build
```

Output goes to `build/bin/`.

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
| `docker` | Docker settings (image, container name, resource limits) |
| `llm` | LLM settings (provider, model, API key, base URL) |
| `mcp` | MCP server settings (command, args, env vars, transport) |
| `agent` | Agent settings (mode, system prompt, workspace directory) |

## Data Storage

All persistent data is stored under `~/.starxo/`:

```
~/.starxo/
├── config.json                # App configuration
├── containers.json            # Container registry
└── sessions/
    └── {session-id}/
        ├── session.json       # Session metadata
        ├── session_data.json  # Unified session data (messages + display + streaming)
        ├── messages.json      # Conversation history - legacy fallback
        └── display.json       # Rich display data - legacy fallback
```

## Documentation

- **Technical Docs**: `doc/` — Project-level docs + per-file technical specs
- **Roadmap**: `plan/` — Future update directions
- **Dev Guide**: `CLAUDE.md` — Architecture overview + engineering conventions
