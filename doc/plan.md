# Starxo 项目技术文档 — 主索引

> 仓库: `starxo` | 产品名: **Starxo** | 版本: 0.1.0
> 生成时间: 2026-03-01 | 文档体系: aigw-core 模式

---

## 一、项目定位

| 属性 | 值 |
|------|------|
| 仓库路径 | `starxo` |
| 项目类型 | 桌面应用 (Wails v2) |
| 产品名 | Starxo |
| 代码文件数 | ~48 Go + ~35 Vue/TS/CSS = ~83 文件 |
| 业务定位 | AI 编程智能体桌面应用，通过 SSH 连接远程服务器并在 Docker 容器中提供隔离的代码编写、执行和文件管理能力 |

Starxo 是一款基于 CloudWeGo Eino 框架构建的 AI 编程助手桌面应用。它采用 Deep Agent 模式，由一个主控智能体(coding_agent)编排三个专用子智能体(code_writer、code_executor、file_manager)，在远程 Docker 沙箱中完成代码编写、执行和文件操作。应用通过 Wails v2 将 Go 后端与 Vue 3 前端集成为原生桌面体验。

---

## 二、实现事实摘要

| 构建维度 | 事实 |
|----------|------|
| Go 构建 | Wails v2.11, `go.mod` 声明 `go 1.24.0` |
| 前端构建 | Vite 6 + vue-tsc, `package.json` scripts: `dev` / `build` / `preview` |
| 容器构建 | 无 (桌面应用，通过 `//go:embed all:frontend/dist` 嵌入前端资源) |
| 测试覆盖 | 无 (当前无测试文件) |
| 入口 | Go: `main.go` -> `app.go`; 前端: `frontend/src/main.ts` -> `App.vue` |
| 配置 | `~/.starxo/config.json` (SSH/Docker/LLM/MCP/Agent) |
| 会话存储 | `~/.starxo/sessions/{id}/` |
| 容器注册表 | `~/.starxo/containers.json` |

---

## 三、关键入口与运行方式

### Go 入口

- **`main.go`**: 程序入口，嵌入前端资源，配置 Wails 窗口(1400x900, 最小 1000x600)，注册 6 个绑定服务
- **`app.go`**: `App` 结构体持有所有服务实例，`startup()` 分发 Wails context 并连接服务间依赖，`shutdown()` 保存会话并断开 SSH

### 前端入口

- **`frontend/src/main.ts`**: 创建 Vue 应用，注册 Pinia、Router、i18n、Naive UI
- **`frontend/src/App.vue`**: 根组件，挂载 `MainLayout`

### Wails 绑定服务列表

| 服务 | 绑定字段 | 源文件 |
|------|----------|--------|
| ChatService | `app.chatService` | `internal/service/chat.go` |
| SandboxService | `app.sandboxService` | `internal/service/sandbox_svc.go` |
| FileService | `app.fileService` | `internal/service/file_svc.go` |
| SettingsService | `app.settingsService` | `internal/service/settings_svc.go` |
| SessionService | `app.sessionService` | `internal/service/session_svc.go` |
| ContainerService | `app.containerService` | `internal/service/container_svc.go` |

### 构建命令

```bash
# 开发模式 (热重载)
wails dev

# 生产构建
wails build

# 前端单独开发
cd frontend && npm run dev

# 前端构建
cd frontend && npm run build
```

---

## 四、IPC 接口 (Wails Events)

### Agent 事件通道

| 事件名 | 载荷类型 | 说明 |
|--------|----------|------|
| `agent:timeline` | `TimelineEvent` | 统一时间线事件 (message/tool_call/tool_result/transfer/info/interrupt/plan/stream_chunk/stream_end) |
| `agent:done` | `nil` | Agent 处理完成 |
| `agent:error` | `string` | 错误信息 |
| `agent:interrupt` | `InterruptEvent` | 中断等待用户输入 (followup/choice) |
| `agent:plan` | `PlanEvent` | 计划状态变更 |
| `agent:mode_changed` | `ModeChangedEvent` | 模式切换 (default/plan) |
| `agent:action` | `AgentActionEvent` | Agent 动作 (tool_call/transfer/info) |
| `agent:message` | `MessageEvent` | 完整消息 (非流式) |
| `agent:tool_result` | `ToolResultEvent` | 工具调用结果 |

### Sandbox 事件通道

| 事件名 | 载荷类型 | 说明 |
|--------|----------|------|
| `sandbox:progress` | `SandboxProgressEvent` | 连接进度 (step + percent) |
| `sandbox:ready` | `SandboxStatusDTO` | 沙箱连接就绪 |
| `sandbox:disconnected` | `nil` | 沙箱断开连接 |

### Session 事件通道

| 事件名 | 载荷类型 | 说明 |
|--------|----------|------|
| `session:switched` | `SessionSwitchedEvent` | 活跃会话切换 |

---

## 五、关键配置

### 配置文件

- 路径: `~/.starxo/config.json`
- 加载: `internal/config/store.go` -> `config.NewStore()`
- 默认值: `internal/config/config.go` -> `DefaultConfig()`

### 配置块列表

| 配置块 | 结构体 | 关键字段 |
|--------|--------|----------|
| `ssh` | `SSHConfig` | host, port(默认22), user(默认root), password, privateKey |
| `docker` | `DockerConfig` | image(默认python:3.11-slim), memoryLimit(2048MB), cpuLimit(1.0), workDir(/workspace), network(true) |
| `llm` | `LLMConfig` | type(默认openai), baseURL, apiKey, model(默认gpt-4o), headers |
| `mcp` | `MCPConfig` | servers[]: name, transport, command, args, url, env, enabled |
| `agent` | `AgentConfig` | maxIterations(默认30) |

---

## 六、依赖与技术栈

### Go 直接依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| `github.com/cloudwego/eino` | v0.7.36 | AI Agent 框架核心 (ADK, Deep Agent, PlanExecute) |
| `github.com/cloudwego/eino-ext/.../ark` | v0.1.64 | 火山引擎 Ark 模型适配 |
| `github.com/cloudwego/eino-ext/.../ollama` | v0.1.8 | Ollama 本地模型适配 |
| `github.com/cloudwego/eino-ext/.../openai` | v0.1.8 | OpenAI/DeepSeek 模型适配 |
| `github.com/cloudwego/eino-ext/.../commandline` | latest | 命令行工具组件 |
| `github.com/cloudwego/eino-ext/.../officialmcp` | v0.1.0 | MCP 官方协议工具 |
| `github.com/modelcontextprotocol/go-sdk` | v1.4.0 | MCP Go SDK |
| `github.com/google/uuid` | v1.6.0 | UUID 生成 |
| `github.com/pkg/sftp` | v1.13.10 | SFTP 文件传输 |
| `github.com/wailsapp/wails/v2` | v2.11.0 | 桌面应用框架 |
| `golang.org/x/crypto` | v0.41.0 | SSH 加密 |

### 前端依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| `vue` | ^3.5.13 | UI 框架 |
| `naive-ui` | ^2.41.0 | 组件库 |
| `pinia` | ^2.3.1 | 状态管理 |
| `vue-router` | ^4.5.0 | 路由 |
| `vue-i18n` | ^12.0.0-alpha.3 | 国际化 |
| `@xterm/xterm` | ^5.5.0 | 终端模拟 |
| `@xterm/addon-fit` | ^0.10.0 | 终端自适应 |
| `markdown-it` | ^14.1.0 | Markdown 渲染 |
| `highlight.js` | ^11.11.1 | 代码高亮 |
| `@vueuse/core` | ^12.5.0 | 组合式工具库 |
| `@vicons/ionicons5` | ^0.12.0 | 图标库 |
| `typescript` | ~5.7.0 | 类型系统 |
| `vite` | ^6.2.0 | 构建工具 |

---

## 七、专题文档索引

| 文档文件 | 内容 |
|----------|------|
| [`business-domain.plan.md`](business-domain.plan.md) | 业务域定义：边界、核心对象、角色、约束 |
| [`business-flows.plan.md`](business-flows.plan.md) | 业务流程：主流程、分支流程、异常流程 |
| [`business-rules.plan.md`](business-rules.plan.md) | 业务规则目录：BR-001 ~ BR-010，含优先级矩阵 |
| [`implementation.plan.md`](implementation.plan.md) | 实现映射：模块职责、规则追溯、调用链、复用关系 |
| [`interfaces.plan.md`](interfaces.plan.md) | 接口说明：Wails 绑定方法、事件通道、配置契约 |
| [`milestones.plan.md`](milestones.plan.md) | 迭代计划：M1 ~ M5 |
| [`open-questions.plan.md`](open-questions.plan.md) | 待确认项：业务层、实现层问题及当前假设 |
| [`files.index.plan.md`](files.index.plan.md) | 文件级文档索引：全部源文件到 .plan.md 的映射表 |
| [`files.coverage.plan.md`](files.coverage.plan.md) | 覆盖统计：文件数、覆盖率、排除目录 |

---

## 八、文件级技术文档

### 文档目录结构

```
doc/
├── plan.md                          # 本文件 — 主索引
├── business-domain.plan.md          # 业务域定义
├── business-flows.plan.md           # 业务流程
├── business-rules.plan.md           # 业务规则目录
├── implementation.plan.md           # 实现映射
├── interfaces.plan.md               # 接口说明
├── milestones.plan.md               # 迭代计划
├── open-questions.plan.md           # 待确认项
├── files.index.plan.md              # 文件级文档索引
├── files.coverage.plan.md           # 覆盖统计
└── src/                             # 文件级技术文档 (待创建)
    ├── main.plan.md
    ├── app.plan.md
    ├── internal/
    │   ├── agent/
    │   ├── service/
    │   ├── sandbox/
    │   ├── tools/
    │   ├── config/
    │   ├── context/
    │   ├── llm/
    │   ├── model/
    │   ├── storage/
    │   ├── store/
    │   └── logger/
    └── frontend/
        └── src/
```

### 映射规则

1. 每个源文件对应一个同名 `.plan.md` 文档文件
2. 文档文件保持与源文件相同的目录层级，统一放在 `doc/src/` 下
3. 例: `internal/agent/deep_agent.go` -> `doc/src/internal/agent/deep_agent.plan.md`
4. 例: `frontend/src/stores/chatStore.ts` -> `doc/src/frontend/src/stores/chatStore.plan.md`
5. 完整映射表见 [`files.index.plan.md`](files.index.plan.md)
6. 覆盖统计见 [`files.coverage.plan.md`](files.coverage.plan.md)
