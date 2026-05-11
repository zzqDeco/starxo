# Starxo - AI 编程智能体桌面应用

[English](README.md)

## 项目简介

Starxo 是一款基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 框架的 AI 编程智能体桌面应用。通过 SSH 连接远程服务器，在轻量 bwrap/Seatbelt 沙箱中自主编写、执行和管理代码，为开发者提供安全隔离的 AI 辅助编程环境。

## 核心特性

- **Deep Agent 架构** — 主智能体协调 3 个专用子智能体（code_writer / code_executor / file_manager），通过 `transfer_to_agent` 实现任务委派
- **双模式运行** — 默认模式（直接执行）+ 计划模式（Planner/Replanner 规划-执行）
- **中断/恢复** — 支持 `ask_user` / `ask_choice` 工具暂停等待用户输入，状态通过 CheckPointStore 保持
- **沙箱隔离** — SSH + 轻量系统沙箱运行时：Linux `bubblewrap` (`bwrap`) 或 macOS Seatbelt (`sandbox-exec`)
- **沙箱诊断** — 设置页检测 bwrap/Seatbelt、Python、venv、user namespace、AppArmor 限制，并返回可复制的远端修复命令
- **Runtime V2 工具面** — 始终可用的 `ToolSearch`、直接文件/搜索/编辑/shell 工具、动态 `Agent` 委派、worktree 隔离、deferred LSP/Skill/Web/Notebook 工具，以及长任务后台输出管理
- **Runtime 工具配置** — 设置页支持 WebSearch/TinyFish 诊断和常驻 LSP server 映射配置，可自定义 language server 命令
- **工具权限审批** — 高风险 runtime 和 MCP 工具调用进入排队审批 UI，可拒绝、允许一次、本会话允许，并可在设置页管理授权
- **MCP 协议** — 支持 Model Context Protocol 扩展工具（stdio/SSE 传输）
- **多 LLM 支持** — OpenAI / DeepSeek / 火山引擎 Ark / Ollama
- **多语言界面** — 中文/英文（vue-i18n）
- **实时事件流** — 通过 Wails Events 实现 `agent:timeline` 统一事件流，前端实时展示 Agent 活动，所有事件携带 `sessionId` 实现多会话隔离
- **多会话并行执行** — 多个会话可同时运行 Agent；切换会话不会取消后台运行的 Agent，切换时完整恢复状态快照
- **会话持久化** — 完整的会话管理，统一存储消息历史、timeline 事件和流式状态
- **文件传输** — 通过 SFTP 直接上传/下载到每个持久沙箱工作区，并提供工作区元信息、路径复制和 tmp 清理
- **开发工作台 UI** — 高信息密度深色工作台，包含命令面板、会话栏、中央执行画布、工作区抽屉、右侧运行时 Dock 和 composer 内模式控制

## 技术栈

### 后端

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.24 | 主语言 |
| Wails | v2.11 | 桌面框架（Go + WebView） |
| CloudWeGo Eino | v0.7 | Agent 框架（ADK, Runner, Deep Agent, PlanExecute） |
| eino-ext | - | LLM Provider (OpenAI/Ark/Ollama) + MCP + Commandline |
| golang.org/x/crypto | - | SSH 连接 |
| pkg/sftp | v1.13 | SFTP 文件传输 |
| MCP Go SDK | v1.4 | Model Context Protocol |

### 前端

| 技术 | 版本 | 用途 |
|------|------|------|
| Vue | 3.5 | UI 框架（`<script setup>` + TypeScript） |
| TypeScript | 5.7 | 类型系统 |
| Vite | 6.2 | 构建工具 |
| Naive UI | 2.41 | 组件库（暗色主题） |
| Pinia | 2.3 | 状态管理 |
| xterm.js | 5.5 | 终端模拟器 |
| markdown-it + highlight.js | 14.1 / 11.11 | Markdown 渲染 + 代码高亮 |
| vue-i18n | 12 | 国际化（zh/en） |

## 项目结构

```
starxo/
├── main.go                          # 应用入口，Wails 初始化，绑定服务
├── app.go                           # App 结构体，服务初始化，生命周期管理
├── wails.json                       # Wails 项目配置
├── go.mod / go.sum                  # Go 依赖管理
│
├── internal/
│   ├── agent/                       # AI Agent 构建与配置
│   │   ├── deep_agent.go            #   Deep Agent 主编排器（3 子 Agent）
│   │   ├── runner.go                #   Runner 构建（默认模式 + 计划模式）
│   │   ├── prompts.go               #   所有 Agent 系统提示词
│   │   ├── codewriter.go            #   code_writer 子 Agent
│   │   ├── codeexecutor.go          #   code_executor 子 Agent
│   │   ├── filemanager.go           #   file_manager 子 Agent
│   │   ├── context.go               #   AgentContext（工作空间、沙箱、SSH 信息）
│   │   ├── plan.go                  #   Plan/Step 类型定义
│   │   ├── plan_wrapper.go          #   计划状态持久化 + 事件发射
│   │   └── tool_wrapper.go          #   eventEmittingTool 包装器
│   │
│   ├── service/                     # Wails 绑定服务（前端 API）
│   │   ├── chat.go                  #   ChatService：Per-Session Agent 生命周期（SessionRun）、消息收发、流式输出
│   │   ├── runtime_context_compact.go # Runtime V2 上下文压缩状态
│   │   ├── runtime_agent_tool.go    #   Runtime V2 动态 Agent 工具
│   │   ├── runtime_lsp_manager.go   #   Runtime V2 常驻 language server 管理
│   │   ├── runtime_workspaces.go    #   Runtime V2 会话级 worktree workspace 管理
│   │   ├── runtime_web_tools.go     #   Runtime V2 WebFetch/WebSearch 工具实现
│   │   ├── websearch_diagnostics.go #   WebSearch/TinyFish 配置诊断
│   │   ├── sandbox_svc.go           #   SandboxService：连接/断开/重连、健康监控（RWMutex 并发安全）
│   │   ├── session_svc.go           #   SessionService：会话 CRUD、多会话状态协调
│   │   ├── settings_svc.go          #   SettingsService：配置管理、连接测试
│   │   ├── file_svc.go              #   FileService：文件上传/下载/预览
│   │   ├── container_svc.go         #   ContainerService：沙箱注册生命周期（兼容命名）
│   │   └── events.go                #   事件 DTO 定义
│   │
│   ├── sandbox/                     # 远程沙箱管理
│   │   ├── manager.go               #   SandboxManager 顶层编排
│   │   ├── ssh.go                   #   SSH 连接管理
│   │   ├── runtime.go               #   bwrap/Seatbelt 运行时管理
│   │   ├── operator.go              #   RemoteOperator（commandline.Operator 实现）
│   │   └── transfer.go              #   文件传输（SFTP 到沙箱工作区）
│   │
│   ├── tools/                       # Agent 工具定义
│   │   ├── registry.go              #   ToolRegistry 中央注册表
│   │   ├── builtin.go               #   内置工具注册
│   │   ├── runtime_tools.go         #   Runtime V2 工具：Bash/Read/Write/Edit/Glob/Grep/tasks
│   │   ├── runtime_deferred_tools.go #  Runtime V2 deferred 工具：LSP/Skill/Notebook/Web metadata
│   │   ├── tool_search.go           #   Runtime-wide deferred 工具发现
│   │   ├── mcp.go                   #   MCP 服务器连接 + 工具加载
│   │   ├── followup.go              #   ask_user 中断工具
│   │   ├── choice.go                #   ask_choice 中断工具
│   │   ├── todos.go                 #   write_todos / update_todo 任务工具
│   │   ├── notify.go                #   notify_user 通知工具
│   │   └── custom.go                #   自定义工具助手
│   │
│   ├── config/                      # 配置管理
│   ├── context/                     # 上下文引擎（历史、文件上下文、窗口化）
│   ├── llm/                         # LLM Provider 工厂
│   ├── model/                       # 数据模型（Message、Session、沙箱注册表）
│   ├── storage/                     # 持久化存储（会话、沙箱）
│   ├── store/                       # CheckPointStore（中断恢复状态）
│   └── logger/                      # 结构化日志 + Eino 回调
│
├── frontend/
│   ├── package.json                 # 前端依赖
│   ├── vite.config.ts               # Vite 配置（@ 路径别名）
│   ├── tsconfig.json                # TypeScript 配置
│   └── src/
│       ├── main.ts                  # Vue 应用入口
│       ├── App.vue                  # 根组件：暗色主题、Wails 事件监听
│       ├── style.css                # 全局样式
│       ├── components/
│       │   ├── chat/                #   聊天面板、消息气泡、中断对话框、任务浮层、输入区
│       │   ├── layout/              #   主布局、头部、侧边栏、任务轨组件
│       │   ├── settings/            #   设置面板（SSH/沙箱/LLM/MCP）
│       │   ├── files/               #   工作区抽屉、文件树、代码预览、文件传输
│       │   ├── containers/          #   沙箱面板 + 常驻 Dock
│       │   ├── status/              #   Agent 状态、连接状态
│       │   └── terminal/            #   终端组件（当前非主界面默认入口）
│       ├── stores/                  #   Pinia 状态管理
│       ├── types/                   #   TypeScript 类型定义
│       ├── composables/             #   Vue 组合式函数
│       └── locales/                 #   i18n 语言包（zh/en）
│
├── build/                           # 平台构建资源（Windows NSIS / macOS plist）
├── .github/workflows/               # GitHub Actions CI/CD 工作流
├── doc/                             # 项目技术文档
├── plan/                            # 未来规划文档
└── logs/                            # 运行日志（agent-YYYY-MM-DD.log）
```

## 快速开始

### 环境要求

- **Go** >= 1.24
- **Node.js** >= 18
- **Wails CLI** v2（安装：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）
- **远程服务器**：需要 SSH 访问权限，并安装 Linux `bubblewrap`/`python3` 或 macOS `sandbox-exec`/`python3`

### 开发运行

```bash
wails dev
```

启动后自动开启 Vite HMR 前端热重载和 Go 后端热重载。前端开发服务器 URL 自动检测，Go 开发服务器运行在 `http://localhost:34115`。

### Agent Runtime V2

顶层 Agent 现在使用更小的 always-loaded runtime 工具面，并通过 `ToolSearch` 按需发现 deferred tools。核心工具包括 `Read`、`Edit`、`Write`、`Bash`、`Glob`、`Grep`、`TaskOutput`、`TaskStop`、`ExitPlanMode`、`Agent`；`read_file`、`shell_execute` 等旧工具名继续作为别名保留。

当前 deferred runtime tools 包括 `EnterWorktree`、`ExitWorktree`、`WorktreeDiff`、`WorktreeMerge`、`LSP`、`LSPEdit`、`Skill`、`NotebookEdit`、`WebFetch`、`WebSearch`。`LSP` 会在远端沙箱安装了对应服务时按 session/workspace/language 复用常驻 language server（`gopls`、`typescript-language-server`、`pyright-langserver`、`rust-analyzer`），不可用时降级到 `rg`/`sed`。`LSPEdit` 通过 permission queue 暴露可写的 language-server rename/format 操作。`Agent` 可同步或后台运行聚焦子任务，也可以为边界清晰的任务请求 context-scoped worktree 隔离，不会切换父会话 workspace。

Runtime worktree 现在具备审阅/合并闭环：`WorktreeDiff` 返回 active worktree 状态、diff stat 和可选 patch；`WorktreeMerge` 会提交 active worktree 修改，合并回原沙箱 workspace，可选移除 worktree，并恢复父会话 workspace。父 workspace 有未提交修改时会拒绝 merge。工作区抽屉也会显示当前会话的 active worktree，可直接审阅、复制、确认合并，并且文件浏览会跟随 active worktree 路径。

会修改文件的 runtime tools 现在会输出结构化 diff 元数据。`Write` 和 `Edit` 的 timeline 事件会展示创建/更新状态、替换数、`+/-` 行数、字节数和受限 patch 预览，不再要求用户阅读原始 JSON。

`WebSearch` 默认使用 DuckDuckGo HTML 搜索，也可以通过 `agent.webSearch.providers` 切换 provider。`type: "tinyfish"` 是专用 TinyFish Search API 适配器，对齐 `GET https://api.search.tinyfish.ai`，默认从 `TINYFISH_API_KEY` 读取 API key 并写入 `X-API-Key`，支持 TinyFish `query`、`location`、`language`、`page` 参数，并解析 `results[].title/url/snippet`。`type: "http"` 继续用于自定义 GET/POST provider，支持 headers、body template 和 JSON path 提取。

设置页提供 WebSearch 静态诊断和显式 smoke test。smoke test 只在用户点击时运行，并返回 provider、URL、耗时和紧凑结果。

`WebFetch` 和 `WebSearch` 只执行 `http`/`https` 请求。空 host、URL userinfo、本地/私有/link-local/组播/未指定地址、IPv6 ULA/link-local 地址以及 `169.254.169.254` 默认阻止；访问非公网 endpoint 需要显式 runtime permission 授权，redirect 跟随后也会先重新校验目标。

计划模式只暴露 read-only trusted 工具；写入、编辑和 shell 执行会在计划批准后才进入可见工具面。

高风险工具调用会进入 Runtime V2 permission queue。桌面端可选择拒绝、允许一次或本会话允许；本会话授权会随 session data 持久化，并可在设置页 Permissions 分区查看或撤销。

后台 Bash 和 Agent 任务可在运行任务面板中查看。面板按当前会话列出任务，支持刷新状态/输出、复制输出，并可通过 Runtime V2 task APIs 停止运行中的任务。

长会话使用 token-aware context compaction。Starxo 会完整保留最近轮次，并注入一段 compact runtime summary，用于保留已发现工具、本会话权限、后台任务 output pointer、文件读取范围、最近编辑摘要、todos、plan 状态和 active worktree routing。完整消息历史仍保存在 `session_data.json`。

### 生产构建

```bash
wails build
```

产物输出至 `build/bin/` 目录。

### Tag 发布

当 `vX.Y.Z` tag 推送到可从 `master` 访问的 commit 时，GitHub Actions 会自动发布桌面安装包。

```bash
git checkout master
git pull origin master
git tag v0.1.0
git push origin v0.1.0
```

发布工作流会构建未签名的 macOS、Windows、Linux 包，上传到 GitHub Release，并生成 `SHA256SUMS.txt`。

工作流结束后需要检查：

- Release 页面已公开，并包含 macOS zip、Windows exe、Windows installer、Linux tarball 和 `SHA256SUMS.txt`。
- `SHA256SUMS.txt` 中的哈希能匹配下载后的产物。
- 三个平台的包至少能启动一次；v1 未签名，macOS/Windows 的系统安全提示属于预期现象。
- 设置页可正常打开，SSH 连接测试可用，沙箱运行时检测能返回符合远端环境的状态。
- Linux 远端可以创建 sandbox 并写入 workspace，且 `network=false` 时外联网络被阻断。

### 前端独立开发

```bash
cd frontend
npm install
npm run dev
```

## 配置说明

应用配置存储于 `~/.starxo/config.json`，包含以下配置块：

| 配置块 | 说明 |
|--------|------|
| `ssh` | SSH 连接配置（主机、端口、用户名、认证方式） |
| `sandbox` | 沙箱运行时配置（运行时、工作区根目录、网络、进程级限制、Python 初始化） |
| `llm` | LLM 配置（Provider、模型、API Key、Base URL） |
| `mcp` | MCP 服务器配置（命令、参数、环境变量、传输方式） |
| `agent` | Agent 配置（模式选择、系统提示词、工作目录） |

### Deferred Surface 开发态开关

以下环境变量仅用于开发和排障：

- `STARXO_ENABLE_DEFERRED_SURFACE_DEBUG_API=1`
  - 打开 Wails deferred surface 调试接口
  - 启动时锁存；修改后需要重启应用
- `STARXO_ENABLE_DEV_DEFERRED_BUILTIN_SAMPLE=1`
  - 注册 `dev_deferred_builtin_sample` 顶层 deferred builtin 实验样本
  - 启动时锁存；修改后需要重启应用

这两个开关默认关闭，不作为生产环境的用户配置入口。

### 沙箱诊断与修复指南

沙箱设置页可以在保存设置前执行完整远端诊断。Linux 检查包括 `bwrap`、`python3`、Python venv 创建、user namespace sysctl、AppArmor unprivileged user namespace 限制，以及 bwrap smoke 命令。macOS 检查包括 `sandbox-exec`、`python3` 和最小 Seatbelt smoke 命令。

普通 Linux 包依赖可以通过“安装运行时”按钮安装。`sysctl`、AppArmor 等主机安全策略变更不会自动执行；Starxo 只展示可复制命令，由操作者审阅后手动运行。

## 数据存储

所有持久化数据存储于 `~/.starxo/` 目录：

```
~/.starxo/
├── config.json                # 应用配置
├── sandboxes.json             # 沙箱注册表
└── sessions/
    └── {session-id}/
        ├── session.json       # 会话元数据
        ├── session_data.json  # 统一会话数据（消息 + 展示 + 流式状态）
        ├── messages.json      # 对话消息历史 - 旧版兼容
        └── display.json       # 富文本展示数据 - 旧版兼容
```

## 文档

- **文档总览**：`doc/README.md` — 文档结构与同步规则
- **文件级技术文档**：`doc/src/` — 当前代码文件对应说明
- **项目级文档**：`doc/` — 总览、研究和非文件级技术资料
- **实施计划**：`plan/` — 当前有效的变更方案
- **协作规范**：`AGENTS.md` — 主分支 / PR / 文档同步规则
- **补充说明**：`CLAUDE.md` — 额外的架构与代理工作说明

## 开发流程

- `master` 是主干分支。
- `dev` 是开发缓冲分支。
- 日常 topic 分支从 `dev` 切出，并先合入 `dev`。
- `dev` 集成验证通过后，再统一合入 `master`。
