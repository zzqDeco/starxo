# 实现映射

> 所属项目: Starxo | 文档类型: 实现映射

---

## 一、模块职责表

| 包路径 | 职责 |
|--------|------|
| `main.go` / `app.go` | 应用入口与 Wails 生命周期管理，创建并连接所有 Service |
| `internal/agent` | AI 智能体构建：Deep Agent 及三个子智能体定义、Runner 构建、提示词管理 |
| `internal/service` | Wails 绑定服务层：封装业务逻辑为前端可调用的方法，管理事件发射，per-session 代理执行管理 |
| `internal/sandbox` | 远程沙箱管理：SSH 连接、Docker 容器操作、命令执行、文件传输 |
| `internal/tools` | 工具注册与实现：内置工具、中断工具、MCP 工具加载、工具包装器 |
| `internal/config` | 配置管理：配置结构体定义、JSON 文件读写、默认值 |
| `internal/context` | 上下文引擎：消息历史管理、上下文窗口化、文件上下文注入 |
| `internal/llm` | LLM 提供商适配：根据配置创建 OpenAI/Ark/Ollama 模型实例 |
| `internal/model` | 数据模型定义：Session、Message、Container 等领域对象 |
| `internal/storage` | 持久化存储：Session 文件存储、Container 注册表 JSON 存储 |
| `internal/store` | 检查点存储：内存中的 CheckPointStore 实现，用于中断/恢复 |
| `internal/logger` | 日志系统：文件日志、Eino 回调日志、Agent 事件日志 |

### 各包文件清单

| 包 | 文件 | 一句话说明 |
|----|------|-----------|
| `(root)` | `main.go` | 程序入口，嵌入前端资源，配置 Wails 窗口参数 |
| `(root)` | `app.go` | App 结构体，持有所有 Service，管理 startup/shutdown 生命周期；不再持有 ctxEngine |
| `agent` | `deep_agent.go` | 构建 Deep Agent (coding_agent)，注册子智能体和直接工具 |
| `agent` | `codewriter.go` | Code Writer 子智能体定义，专注代码编写 |
| `agent` | `codeexecutor.go` | Code Executor 子智能体定义，专注代码执行 |
| `agent` | `filemanager.go` | File Manager 子智能体定义，专注文件操作 |
| `agent` | `runner.go` | BuildDefaultRunner 和 BuildPlanRunner，构建两种运行模式 |
| `agent` | `plan.go` | 计划模式相关逻辑，PlanExecute 配置 |
| `agent` | `plan_wrapper.go` | 计划步骤包装，将 plan 状态转换为前端事件 |
| `agent` | `prompts.go` | 所有智能体的系统提示词定义 |
| `agent` | `context.go` | AgentContext 结构体，传递运行时上下文给智能体；OnToolEvent 回调新增 ctx 参数用于 session 身份传播 |
| `agent` | `tool_wrapper.go` | 工具包装器，为子智能体的工具调用注入事件发射回调；通过 ctx 传播 sessionID |
| `service` | `chat.go` | ChatService，per-session 消息处理、Agent 运行管理、中断恢复、SessionRun 状态管理 |
| `service` | `sandbox_svc.go` | SandboxService，沙箱连接/断开/重连/健康监控；使用 RWMutex 并发安全 |
| `service` | `session_svc.go` | SessionService，会话 CRUD、切换、持久化；通过 ChatService 访问 per-session 状态 |
| `service` | `settings_svc.go` | SettingsService，配置读写、SSH/LLM 连接测试 |
| `service` | `file_svc.go` | FileService，文件上传/下载/列表/预览 |
| `service` | `container_svc.go` | ContainerService，容器列表/状态刷新/启停/销毁 |
| `service` | `events.go` | 所有 Wails 事件的载荷结构体定义；TimelineEvent/InterruptEvent/ModeChangedEvent/SessionSwitchedEvent 含 sessionId |
| `sandbox` | `ssh.go` | SSHClient，SSH 连接管理和命令执行 |
| `sandbox` | `docker.go` | RemoteDockerManager，远程 Docker 容器操作 |
| `sandbox` | `manager.go` | SandboxManager，组合 SSH + Docker + Operator + Transfer |
| `sandbox` | `operator.go` | Operator，封装容器内命令执行和文件读写 |
| `sandbox` | `transfer.go` | FileTransfer，SFTP 文件传输，本地-远程-容器三级传输 |
| `sandbox` | `setup.go` | Setup，容器初始化 (安装工具、创建工作目录等) |
| `tools` | `builtin.go` | 内置工具注册 (shell 命令、文件读写) |
| `tools` | `followup.go` | FollowUp 工具 (ask_user)，触发 StatefulInterrupt |
| `tools` | `choice.go` | Choice 工具 (ask_choice)，触发 StatefulInterrupt |
| `tools` | `todos.go` | Todo 工具 (write_todos, update_todo)，任务追踪；新增 ClearTodos() 用于会话切换 |
| `tools` | `notify.go` | NotifyUser 工具，向用户发送通知 |
| `tools` | `registry.go` | ToolRegistry，工具注册表管理 |
| `tools` | `mcp.go` | MCP 服务器连接和工具加载 |
| `tools` | `custom.go` | 自定义工具扩展点 |
| `config` | `config.go` | 配置结构体定义 (AppConfig, SSHConfig, DockerConfig 等) |
| `config` | `store.go` | 配置文件读写 (~/.starxo/config.json) |
| `context` | `engine.go` | Engine，消息历史管理 (Add/Prepare/Clear/Import/Export) |
| `context` | `windowing.go` | WindowMessages，上下文窗口化与截断策略 |
| `context` | `timeline.go` | TimelineCollector，后端时间线收集器 (Add/Export/Import/Clear) |
| `context` | `history.go` | 消息历史持久化辅助 |
| `context` | `filecontext.go` | 文件上下文注入，向消息中添加文件信息 |
| `llm` | `provider.go` | NewChatModel，根据 LLMConfig.Type 创建模型实例 |
| `model` | `session.go` | Session 数据模型 |
| `model` | `message.go` | Message 数据模型 |
| `model` | `container.go` | Container 数据模型及状态常量 |
| `model` | `session_data.go` | SessionData 统一会话数据模型 |
| `storage` | `session_store.go` | SessionStore，会话文件系统持久化 |
| `storage` | `container_store.go` | ContainerStore，容器注册表 JSON 持久化 |
| `store` | `checkpoint.go` | InMemoryStore，Eino CheckPointStore 内存实现 |
| `logger` | `logger.go` | 日志初始化、文件输出、Info/Warn/Error 方法 |
| `logger` | `callbacks.go` | Eino 全局回调注册，记录 Agent/Tool 事件 |

---

## 二、规则 -> 代码追溯矩阵

| 规则 | 主要实现文件 | 关键函数/方法 |
|------|-------------|--------------|
| BR-001 (沙箱前置检查) | `internal/service/chat.go` | `buildRunnersLocked()` 开头的 `s.sandbox == nil` 检查 |
| BR-002 (Deep Agent 编排) | `internal/agent/deep_agent.go` | `BuildDeepAgent()` 中 `deep.New()` 的 `SubAgents` 参数 |
| BR-003 (中断/恢复) | `internal/tools/followup.go`, `internal/tools/choice.go`, `internal/service/chat.go` | `StatefulInterrupt`, `handleInterruptForRun()`, `ResumeWithAnswer()`, `ResumeWithChoice()` |
| BR-004 (配置失效) | `internal/service/chat.go`, `app.go` | `InvalidateRunner()`, `invalidateRunners()`, `settingsService.SetOnSettingsSave()` |
| BR-005 (会话持久化) | `internal/service/session_svc.go`, `internal/storage/session_store.go` | `SaveCurrentSession()`, `saveCurrentLocked()`, `SwitchSession()` |
| BR-006 (SFTP 传输) | `internal/sandbox/transfer.go` | `UploadToContainer()`, `DownloadFromContainer()` |
| BR-007 (上下文窗口化) | `internal/context/windowing.go`, `internal/context/engine.go` | `WindowMessages()`, `TruncateContent()`, `PrepareMessages()` |
| BR-008 (线程安全) | `internal/service/chat.go`, `internal/service/sandbox_svc.go` | `chat.go`: `sync.Mutex` + per-session `SessionRun`; `sandbox_svc.go`: `sync.RWMutex` + 锁外回调 |
| BR-009 (容器独立) | `app.go`, `internal/service/sandbox_svc.go` | `shutdown()` 仅断开 SSH, `ActivateContainer()`/`DeactivateContainer()` |
| BR-010 (MCP 非致命) | `internal/service/chat.go` | `buildRunnersLocked()` 中的 MCP 循环 `continue` 分支 |
| BR-011 (Per-Session 执行) | `internal/service/chat.go`, `internal/service/session_svc.go` | `SessionRun`, `getOrCreateRun()`, `activeRun()`, per-session 运行守卫, `contextWithSessionID()` |
| BR-012 (事件路由) | `internal/service/chat.go`, `frontend/src/App.vue` | `emitTimelineForRun/ForSession()` + 前端 `isActiveSession()` 过滤 |

---

## 三、运行时调用链

### 核心调用链: 用户消息 -> Agent 响应 (Per-Session)

```
[前端] 用户输入 -> Wails IPC -> ChatService.SendMessage(userMessage)
  │
  ├─ activeRun() -> getOrCreateRun(activeSessionID)
  │    └─ SessionRun {ctxEngine, timeline, running, mode, ...}
  │
  ├─ run.ctxEngine.AddUserMessage(userMessage)
  │    └─ engine.go: 消息添加到 per-session 历史列表
  │
  ├─ run.timeline.AddUserTurn()
  │    └─ timeline.go: 记录用户轮次到 per-session 时间线
  │
  ├─ [首次] buildRunnersLocked() (在锁内执行，无间隙)
  │    ├─ llm.NewChatModel(ctx, cfg.LLM)
  │    │    └─ provider.go: 根据 type 创建 openai/ark/ollama 模型
  │    ├─ tools.RegisterBuiltinTools(registry, op, wsPath)
  │    │    └─ builtin.go: 注册 shell/file 工具
  │    ├─ tools.ConnectMCPServer() + LoadMCPTools()
  │    │    └─ mcp.go: 连接 MCP 服务器，加载工具
  │    ├─ buildAgentContext()
  │    │    └─ OnToolEvent = func(ctx, ...) { emitTimelineForSession(evt, SessionIDFromContext(ctx)) }
  │    ├─ agent.BuildDeepAgent(ctx, mdl, op, extraTools, ac)
  │    │    ├─ NewCodeWriterAgent()    -- codewriter.go + WrapToolsWithEvents
  │    │    ├─ NewCodeExecutorAgent()  -- codeexecutor.go + WrapToolsWithEvents
  │    │    ├─ NewFileManagerAgent()   -- filemanager.go + WrapToolsWithEvents
  │    │    └─ deep.New() -- Eino ADK Deep Agent
  │    ├─ agent.BuildDefaultRunner()   -- runner.go
  │    └─ agent.BuildPlanRunner()      -- runner.go
  │
  ├─ contextWithSessionID(runCtx, sessionID)
  │
  ├─ run.ctxEngine.PrepareMessages()
  │    └─ windowing.go: WindowMessages() 截断旧消息
  │
  └─ goroutine: runner.Run(runCtx, messages, checkpointID)
       └─ Eino ADK Runner
            └─ DeepAgent.Generate()
                 ├─ LLM 推理
                 ├─ [工具调用] tool_wrapper.go 包装 -> onEvent(ctx, ...) -> emitTimelineForSession
                 │    └─ operator.go: RunCommand() / ReadFile() / WriteFile()
                 │         └─ docker.go: ExecCommand() (docker exec)
                 ├─ [子智能体] transfer_to_agent -> 子智能体.Generate()
                 │    └─ 同上工具调用流程 (ctx 携带 sessionID)
                 └─ [中断] followup.go / choice.go -> StatefulInterrupt
                      └─ chat.go: handleInterruptForRun() -> emit agent:interrupt {sessionId}
```

### 事件发射链: Agent -> 前端 (Per-Session)

```
Agent 事件产生
  └─ chat.go: processEventsForRun(events, checkpointID, run)
       ├─ event.Action.TransferToAgent
       │    └─ emitTimelineForRun {type:"transfer", sessionId} + {type:"thinking", sessionId}
       ├─ msg.Content (reasoning)
       │    └─ emitTimelineForRun {type:"reasoning", sessionId}
       ├─ msg.ToolCalls
       │    └─ emitTimelineForRun {type:"tool_call", sessionId}
       ├─ msg.Role == Tool
       │    └─ emitTimelineForRun {type:"tool_result", sessionId} + {type:"thinking", sessionId}
       ├─ mv.IsStreaming
       │    └─ drainStreamForRun() -> emitTimelineForRun {type:"stream_chunk"} x N
       │                           -> emitTimelineForRun {type:"stream_end"}
       ├─ msg.Content (非流式)
       │    └─ emitTimelineForRun {type:"message", sessionId}
       ├─ event.Action.Interrupted
       │    └─ handleInterruptForRun() -> emit "agent:interrupt" {sessionId}
       └─ 孤立 tool_call 检测
            └─ run.ctxEngine.AddToolResult(id, "Error: ...")

前端事件接收 (App.vue)
  └─ EventsOn('agent:timeline', (data) => {
       if (!isActiveSession(data)) return  -- 按 sessionId 过滤
       chatStore.addTimelineEvent(data)
     })
```

### 会话切换调用链: SessionService -> ChatService -> 前端

```
SessionService.SwitchSession(sessionID)
  ├─ saveCurrentLocked()
  │    └─ chatService.SessionCtxEngine(currentID).ExportMessages()
  │    └─ chatService.SessionTimeline(currentID).Export()
  │    └─ chatService.SessionStreamingState(currentID)
  │    └─ sessionStore.SaveSessionData(currentID, sd)
  ├─ chatService.SetActiveSessionID(sessionID)
  ├─ [若未运行] chatService.GetOrCreateRun(sessionID)
  │    ├─ run.ctxEngine.ImportMessages(sessionData.Messages)
  │    └─ run.timeline.Import(sessionData.Display)
  ├─ chatService.GetSessionRunSnapshot(sessionID)
  │    └─ return (running, currentAgent, mode, interrupt)
  ├─ emit session:switched {session, containerID, agentRunning, currentAgent, mode, hasInterrupt, interrupt}
  └─ onSessionSwitch(containerRegID)
       └─ go sandboxService.ActivateContainer(containerRegID)
```

---

## 四、共享/复用关系

### 跨模块共享组件

| 共享组件 | 提供者 | 消费者 |
|----------|--------|--------|
| `agent.AgentContext` | `service/chat.go` 构建 | `agent/deep_agent.go`, `agent/codewriter.go`, `agent/codeexecutor.go`, `agent/filemanager.go`, `agent/prompts.go`, `agent/tool_wrapper.go` |
| `agent/tool_wrapper.go` | `agent` 包 | 所有子智能体通过它包装工具以发射事件 (通过 ctx 传播 sessionID) |
| `service/events.go` | `service` 包 | 所有 Service 引用事件结构体; 前端 TypeScript 类型对应 |
| `service.SessionRun` | `service/chat.go` 定义 | `service/session_svc.go` 通过 ChatService 方法间接访问 |
| `sandbox.SandboxManager` | `service/sandbox_svc.go` 创建 | `service/chat.go` (获取 Operator), `service/file_svc.go` (获取 Transfer + Docker) |
| `sandbox.Operator` | `sandbox/operator.go` | `service/chat.go` (传递给 Agent), `service/file_svc.go` (文件列表), `tools/builtin.go` (注册工具) |
| `config.Store` | `config/store.go` | `service/chat.go`, `service/sandbox_svc.go`, `service/settings_svc.go` |
| `config.AppConfig` | `config/config.go` 定义 | 几乎所有 Service 读取配置 |
| `context.Engine` | `service/chat.go` 在 SessionRun 中创建 | `service/chat.go` (per-session 消息管理), `service/session_svc.go` (通过 ChatService 间接访问) |
| `context.TimelineCollector` | `service/chat.go` 在 SessionRun 中创建 | `service/chat.go` (per-session 事件收集), `service/session_svc.go` (通过 ChatService 间接访问) |
| `storage.SessionStore` | `app.go` 创建 | `service/session_svc.go` |
| `storage.ContainerStore` | `app.go` 创建 | `service/sandbox_svc.go`, `service/container_svc.go`, `service/session_svc.go` |
| `tools.ClearTodos` | `tools/todos.go` | `service/chat.go` (ClearHistory), `service/session_svc.go` (CreateSession, SwitchSession) |
| `model.*` | `model` 包定义 | `service/*`, `storage/*` |

### 依赖方向

```
main.go / app.go
  └── service/*        (Wails 绑定层 + per-session 管理)
        ├── agent/*    (智能体构建)
        │     ├── tools/*    (工具注册)
        │     └── llm/*      (模型适配)
        ├── sandbox/*  (远程执行环境)
        ├── config/*   (配置管理)
        ├── context/*  (上下文引擎 — per-session 实例)
        ├── storage/*  (持久化)
        ├── store/*    (检查点)
        ├── model/*    (数据模型)
        └── logger/*   (日志)
```

依赖规则: 上层可依赖下层，下层不依赖上层。`model` 和 `config` 作为基础包被广泛依赖。`service` 层是组装层，依赖所有其他包。`context.Engine` 和 `context.TimelineCollector` 现在作为 per-session 实例在 `ChatService.SessionRun` 中创建，不再由 `app.go` 全局创建。
