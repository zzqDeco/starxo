# chat.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/chat.go
- 文档文件: doc/src/internal/service/chat.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `ChatService`，是前端与 AI 代理之间的核心桥梁。它管理聊天交互的完整生命周期：构建代理和 runner、发送用户消息、流式处理代理事件、处理中断（follow-up 问题和选择题）、恢复执行、停止生成、清除历史。支持 "default" 和 "plan" 两种运行模式。所有代理事件通过统一的 `agent:timeline` 通道转发到前端。内置 `TimelineCollector` 在后端收集所有时间线事件，使后端成为唯一的持久化生产者。流式输出使用 50ms 批量窗口合并 IPC 调用，降低通信频率。
- **核心架构变更**: 采用 per-session agent 执行模型。引入 `SessionRun` 结构体，每个会话拥有独立的 ctxEngine、timeline、运行生命周期和中断状态。`ChatService` 通过 `sessions map[string]*SessionRun` 管理多个会话的并发执行，支持后台会话独立运行。通过 `context.Context` 传播 sessionID，确保事件路由到正确的会话。
- **模式控制增强**: 支持手动模式切换（`SetMode/GetMode`）与复杂任务自动升级到 plan 模式（`shouldAutoPlanMode`），并发射 `agent:mode_changed`。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `SendMessage(userMessage)`、`ResumeWithAnswer(answer)`、`ResumeWithChoice(selectedIndex)`、`StopGeneration()`、`ClearHistory()`、`SetMode(mode)`、`GetMode()`、`BuildRunners()`
  - 依赖注入: `config.Store`、`sandbox.SandboxManager`（ctxEngine 参数保留但忽略，向后兼容）、`SessionService`
- 输出结果:
  - Wails 事件发射: `agent:timeline`、`agent:error`、`agent:done`、`agent:interrupt`、`agent:mode_changed`（所有事件均携带 `sessionId` 字段）
  - 状态变更: 代理执行、会话历史更新、TimelineCollector 累积时间线数据、StreamingState 跟踪流式中途状态

## 4. 关键实现细节
- 结构体/接口定义:
  - `contextKey`: context key 类型，用于 sessionID 传播
  - `PendingInterrupt`: 中断恢复状态，包含 `CheckpointID`、`InterruptID`、`Info`
  - `SessionRun`: **per-session 执行状态结构体**，包含 `sessionID`、`ctxEngine`（独立上下文引擎）、`timeline`（独立时间线收集器）、`running`（运行标志）、`cancelFn`（取消函数）、`runDone`（完成信号 channel）、`pendingInterrupt`（挂起中断）、`streamingState`（流式中途状态）、`mode`（"default"/"plan"）、`currentAgent`（当前代理名称）
  - `ChatService`: 核心聊天服务结构体，包含共享资源（deepAgent、defaultRunner、planRunner、sandbox、store、checkpointStore）和 per-session 状态管理（`sessions map[string]*SessionRun`、`activeSessionID string`）
- 导出函数/方法:
  - `NewChatService(store) *ChatService`: 构造函数，初始化 sessions map 和 checkpointStore
  - `SetContext(ctx)`: 设置 Wails 应用上下文
  - `SetDependencies(sbx, ctxEngine)`: 注入沙箱管理器（ctxEngine 参数忽略，向后兼容）
  - `UpdateSandbox(sbx)`: 更新沙箱引用并使 runner 失效
  - `InvalidateRunner()`: 强制下次消息时重建 runner
  - `SetOnAgentDone(fn)`: 注册代理完成回调，回调签名 `func(sessionID string)`
  - `SetSessionService(ss)`: 注入会话服务
  - `SetActiveSessionID(id)`: 设置当前活跃会话 ID
  - `GetActiveSessionID() string`: 获取当前活跃会话 ID
  - `GetOrCreateRun(sessionID) *SessionRun`: 获取或创建 per-session 运行状态（供 SessionService 使用）
  - `RemoveSession(sessionID)`: 移除会话运行状态
  - `SetMode(mode) error`: 切换活跃会话的运行模式，发射 `agent:mode_changed` 事件（含 sessionID）
  - `GetMode() string`: 获取活跃会话当前模式
  - `IsRunning() bool`: 活跃会话是否有运行中的代理
  - `IsSessionRunning(sessionID) bool`: 指定会话是否有运行中的代理
  - `WaitForSessionDone(sessionID, timeout) error`: 等待指定会话代理运行完成
  - `SendMessage(userMessage) error`: 发送用户消息，per-session 运行守卫，在锁内构建 runners（无间隙），异步执行代理并流式处理事件
  - `ResumeWithAnswer(answer) error`: 以文本回答恢复中断执行
  - `ResumeWithChoice(selectedIndex) error`: 以选择恢复中断执行
  - `StopGeneration() error`: 取消活跃会话当前代理执行
  - `StopSessionGeneration(sessionID)`: 取消指定会话的代理执行
  - `ClearHistory() error`: 清除活跃会话对话历史、重置 timeline 和 streamingState、调用 `tools.ClearTodos()`
  - `BuildRunners() error`: 公开入口，内部委托 `buildRunnersLocked()`
  - `Timeline() *agentctx.TimelineCollector`: 获取活跃会话时间线收集器
  - `StreamingState() *model.StreamingState`: 获取活跃会话当前流式中途状态（返回副本）
  - `CtxEngine() *agentctx.Engine`: 获取活跃会话上下文引擎
  - `SessionCtxEngine(sessionID) *agentctx.Engine`: 获取指定会话上下文引擎
  - `SessionTimeline(sessionID) *agentctx.TimelineCollector`: 获取指定会话时间线收集器
  - `SessionStreamingState(sessionID) *model.StreamingState`: 获取指定会话流式状态（返回副本）
  - `GetSessionRunSnapshot(sessionID) (running, currentAgent, mode, interrupt)`: 获取会话运行状态快照（供 SessionSwitchedEvent 使用）
- 未导出函数/方法:
  - `contextWithSessionID(ctx, sessionID) context.Context`: 将 sessionID 注入 context
  - `SessionIDFromContext(ctx) string`: 从 context 提取 sessionID
  - `getOrCreateRun(sessionID) *SessionRun`: 获取或创建 per-session 运行状态（caller 持锁）
  - `activeRun() *SessionRun`: 获取活跃会话的 SessionRun（caller 持锁）
  - `invalidateRunners()`: 清空 deepAgent/defaultRunner/planRunner
  - `emitTimelineForRun(evt, run)`: 为指定 run 发射时间线事件（设置 sessionID，写入 timeline collector）
  - `emitTimelineForSession(evt, sessionID)`: 通过 sessionID 查找 run 发射时间线事件（用于 OnToolEvent 回调）
  - `processEventsForRun(events, checkpointID, run)`: per-session 事件消费，发射前端事件、检测中断
  - `handleInterruptForRun(interruptCtx, checkpointID, run)`: per-session 中断处理
  - `drainStreamForRun(stream, agentName, run)`: per-session 流式输出消费，50ms 批量窗口
  - `buildRunnersLocked()`: 在锁内构建 mode-aware deep agent（default/plan 各一份）和两种 runner
  - `shouldAutoPlanMode(userMessage string) bool`: 基于关键词启发式判断复杂任务并自动切换到 plan 模式
  - `buildAgentContext()`: 构建 AgentContext，OnToolEvent 回调通过 context 传播 sessionID
  - `buildInterruptEvent(pi, sessionID) *InterruptEvent`: 将 PendingInterrupt 转换为 InterruptEvent
- Wails 绑定方法: `SendMessage`、`ResumeWithAnswer`、`ResumeWithChoice`、`StopGeneration`、`ClearHistory`、`SetMode`、`GetMode`、`BuildRunners`
- 事件发射:
  - `agent:timeline`: 统一时间线事件（所有事件均含 `sessionId` 字段）
  - `agent:error`: 错误信息（含 `sessionId`）
  - `agent:done`: 代理执行完成（含 `sessionId`）
  - `agent:interrupt`: 中断事件（含 `sessionId`）
  - `agent:mode_changed`: 模式切换（含 `sessionId`）

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/agent`: BuildDeepAgent、BuildDefaultRunner、BuildPlanRunner、AgentContext、DefaultAgentContext
  - `starxo/internal/config`: Store、AppConfig
  - `starxo/internal/context` (agentctx): Engine（上下文引擎）、TimelineCollector（时间线收集器）、NewEngine、NewTimelineCollector
  - `starxo/internal/llm`: NewChatModel
  - `starxo/internal/logger`: 日志记录
  - `starxo/internal/model`: SessionData、StreamingState（流式中途状态）
  - `starxo/internal/sandbox`: SandboxManager
  - `starxo/internal/store` (checkpoint): NewInMemoryStore
  - `starxo/internal/tools`: ToolRegistry、RegisterBuiltinTools、ConnectMCPServer、LoadMCPTools、FollowUpInfo、ChoiceInfo、ClearTodos
- 外部依赖:
  - `github.com/cloudwego/eino/adk`: Runner、Agent、AsyncIterator、AgentEvent、CheckPointID、ResumeParams
  - `github.com/cloudwego/eino/compose`: CheckPointStore
  - `github.com/cloudwego/eino/schema`: Message、ToolCall、ConcatMessages
  - `github.com/wailsapp/wails/v2/pkg/runtime` (wailsruntime): EventsEmit
  - `context`、`strings`、`time`、`io`、`sync`、`fmt`（标准库）
- 关键配置: 通过 `config.Store` 获取 LLM 配置、SSH 配置、MCP 服务器配置

## 6. 变更影响面
- 事件格式变更直接影响前端 Vue 组件的事件监听和渲染（所有事件现在携带 `sessionId`）
- runner 构建逻辑变更影响代理的初始化和工具可用性
- 中断处理逻辑变更影响用户与代理的交互流程
- 消息处理变更影响会话历史的完整性
- `processEventsForRun` 将工具调用请求和工具执行结果同步加入 per-session 上下文历史
- `processEventsForRun` 中 ToolCalls 事件处理后使用 `continue` 跳过后续的 `allContents` 累积，避免重复
- `processEventsForRun` 在工具调用事件前发射 reasoning 事件，在 transfer 和子代理 tool_result 后发射 thinking 事件
- `processEventsForRun` 通过 `pendingToolCalls` map 跟踪已存入的 tool_call_id，为未收到结果的孤立 tool_call 注入合成错误响应
- Agent 执行过程中通过防抖机制（每 10 秒）中间保存会话
- `emitTimelineForRun` / `emitTimelineForSession` 确保每次 `agent:timeline` 事件同步写入对应会话的 TimelineCollector
- `drainStreamForRun` 使用 50ms ticker 窗口合并流式 chunk，同时维护 per-session `streamingState`
- `SessionCtxEngine`、`SessionTimeline`、`SessionStreamingState` 被 `SessionService` 调用，用于 per-session 持久化
- `GetSessionRunSnapshot` 被 `SessionService.SwitchSession` 调用，构建富状态的 `SessionSwitchedEvent`
- `buildRunnersLocked` 在锁内执行，消除了之前 `BuildRunners` 解锁后到赋值之间的竞态间隙
- `contextWithSessionID` / `SessionIDFromContext` 实现 context-based sessionID 传播，使 OnToolEvent 回调能正确路由事件
- `contextWithSessionID` 额外写入 plain key `\"sessionID\"`，供底层包读取会话作用域（避免 import cycle）
- `SendMessage` 中自动 plan 切换逻辑会改变后续 runner 选择与主代理工具权限边界
- `buildRunnersLocked` 现在分别构建 default/plan 两套 deep agent，plan 模式下主代理不直接持有 `extraTools`
- 是整个后端最核心的服务文件，变更需格外谨慎

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `processEventsForRun` 方法较长（~150 行），如需扩展事件类型建议拆分为独立的事件处理函数。
- 并发安全通过 `sync.Mutex` 保证，`SessionRun` 的字段在锁内访问。`buildRunnersLocked` 在锁内执行消除竞态。
- `emitTimelineForSession` 中的短暂锁获取/释放需注意——用于 OnToolEvent 回调路径，避免长锁持有。
- 流式输出的 `drainStreamForRun` 在 goroutine 中运行，使用 50ms ticker 批量窗口合并 chunk；修改批量间隔需权衡延迟与 IPC 频率。
- `streamingState` 在 `drainStreamForRun` 中更新、在 agent 完成后清除；`SessionService.saveCurrentLocked()` 通过 `SessionStreamingState()` 读取（返回副本，线程安全）。
- 中断恢复（ResumeWithAnswer/ResumeWithChoice）共享大量重复代码，可考虑抽取公共方法。
- `todoStore` 是全局的，`ClearTodos()` 在 `ClearHistory` 中调用；如需 per-session todos 需进一步重构。
- `checkpointStore` 是所有会话共享的单个 InMemoryStore；如需会话隔离需按 session 分隔 checkpoint namespace。
