# Starxo - AI 编程智能体桌面应用

[English](README.md)

## 项目简介

Starxo 是一款基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 框架的 AI 编程智能体桌面应用。通过 SSH 连接远程服务器，在 Docker 容器沙箱中自主编写、执行和管理代码，为开发者提供安全隔离的 AI 辅助编程环境。

## 核心特性

- **Deep Agent 架构** — 主智能体协调 3 个专用子智能体（code_writer / code_executor / file_manager），通过 `transfer_to_agent` 实现任务委派
- **双模式运行** — 默认模式（直接执行）+ 计划模式（Planner/Replanner 规划-执行）
- **中断/恢复** — 支持 `ask_user` / `ask_choice` 工具暂停等待用户输入，状态通过 CheckPointStore 保持
- **沙箱隔离** — SSH + Docker 容器环境，支持容器生命周期管理（创建/重连/停止/销毁）
- **MCP 协议** — 支持 Model Context Protocol 扩展工具（stdio/SSE 传输）
- **多 LLM 支持** — OpenAI / DeepSeek / 火山引擎 Ark / Ollama
- **多语言界面** — 中文/英文（vue-i18n）
- **实时事件流** — 通过 Wails Events 实现 `agent:timeline` 统一事件流，前端实时展示 Agent 活动，所有事件携带 `sessionId` 实现多会话隔离
- **多会话并行执行** — 多个会话可同时运行 Agent；切换会话不会取消后台运行的 Agent，切换时完整恢复状态快照
- **会话持久化** — 完整的会话管理，统一存储消息历史、timeline 事件和流式状态
- **文件传输** — 支持文件上传/下载，小文件 base64 + docker exec，大文件 SFTP + docker cp
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
│   │   ├── context.go               #   AgentContext（工作空间、容器、SSH 信息）
│   │   ├── plan.go                  #   Plan/Step 类型定义
│   │   ├── plan_wrapper.go          #   计划状态持久化 + 事件发射
│   │   └── tool_wrapper.go          #   eventEmittingTool 包装器
│   │
│   ├── service/                     # Wails 绑定服务（前端 API）
│   │   ├── chat.go                  #   ChatService：Per-Session Agent 生命周期（SessionRun）、消息收发、流式输出
│   │   ├── sandbox_svc.go           #   SandboxService：连接/断开/重连、健康监控（RWMutex 并发安全）
│   │   ├── session_svc.go           #   SessionService：会话 CRUD、多会话状态协调
│   │   ├── settings_svc.go          #   SettingsService：配置管理、连接测试
│   │   ├── file_svc.go              #   FileService：文件上传/下载/预览
│   │   ├── container_svc.go         #   ContainerService：容器生命周期
│   │   └── events.go                #   事件 DTO 定义
│   │
│   ├── sandbox/                     # 远程沙箱管理
│   │   ├── manager.go               #   SandboxManager 顶层编排
│   │   ├── ssh.go                   #   SSH 连接管理
│   │   ├── docker.go                #   远程 Docker 管理
│   │   ├── operator.go              #   RemoteOperator（commandline.Operator 实现）
│   │   ├── transfer.go              #   文件传输（SFTP + docker cp）
│   │   └── setup.go                 #   环境初始化（Docker 安装、镜像拉取）
│   │
│   ├── tools/                       # Agent 工具定义
│   │   ├── registry.go              #   ToolRegistry 中央注册表
│   │   ├── builtin.go               #   内置工具注册
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
│   ├── model/                       # 数据模型（Message, Session, Container）
│   ├── storage/                     # 持久化存储（会话、容器）
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
│       │   ├── settings/            #   设置面板（SSH/Docker/LLM/MCP）
│       │   ├── files/               #   工作区抽屉、文件树、代码预览、文件传输
│       │   ├── containers/          #   容器面板 + 常驻 Dock
│       │   ├── status/              #   Agent 状态、连接状态
│       │   └── terminal/            #   终端组件（当前非主界面默认入口）
│       ├── stores/                  #   Pinia 状态管理
│       ├── types/                   #   TypeScript 类型定义
│       ├── composables/             #   Vue 组合式函数
│       └── locales/                 #   i18n 语言包（zh/en）
│
├── build/                           # 平台构建资源（Windows NSIS / macOS plist）
├── doc/                             # 项目技术文档
├── plan/                            # 未来规划文档
└── logs/                            # 运行日志（agent-YYYY-MM-DD.log）
```

## 快速开始

### 环境要求

- **Go** >= 1.24
- **Node.js** >= 18
- **Wails CLI** v2（安装：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）
- **远程服务器**：需要 SSH 访问权限 + Docker 已安装（或允许安装）

### 开发运行

```bash
wails dev
```

启动后自动开启 Vite HMR 前端热重载和 Go 后端热重载。前端开发服务器 URL 自动检测，Go 开发服务器运行在 `http://localhost:34115`。

### 生产构建

```bash
wails build
```

产物输出至 `build/bin/` 目录。

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
| `docker` | Docker 配置（镜像、容器名、资源限制） |
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

## 数据存储

所有持久化数据存储于 `~/.starxo/` 目录：

```
~/.starxo/
├── config.json                # 应用配置
├── containers.json            # 容器注册表
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
