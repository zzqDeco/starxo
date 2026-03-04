# chat.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/chat.go
- 文档文件: doc/src/internal/service/chat.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `ChatService`，是前端与 AI 代理之间的核心桥梁。它管理聊天交互的完整生命周期：构建代理和 runner、发送用户消息、流式处理代理事件、处理中断（follow-up 问题和选择题）、恢复执行、停止生成、清除历史。支持 "default" 和 "plan" 两种运行模式。所有代理事件通过统一的 `agent:timeline` 通道转发到前端（已移除冗余的 `agent:action`、`agent:message`、`agent:tool_result` 通道）。内置 `TimelineCollector` 在后端收集所有时间线事件，使后端成为唯一的持久化生产者。流式输出使用 50ms 批量窗口合并 IPC 调用，降低通信频率。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `SendMessage(userMessage)`、`ResumeWithAnswer(answer)`、`ResumeWithChoice(selectedIndex)`、`StopGeneration()`、`ClearHistory()`、`SetMode(mode)`、`GetMode()`、`BuildRunners()`
  - 依赖注入: `config.Store`、`sandbox.SandboxManager`、`agentctx.Engine`、`SessionService`
- 输出结果:
  - Wails 事件发射: `agent:timeline`、`agent:error`、`agent:done`、`agent:interrupt`、`agent:mode_changed`（已移除冗余的 `agent:message`、`agent:action`、`agent:tool_result`）
  - 状态变更: 代理执行、会话历史更新、TimelineCollector 累积时间线数据、StreamingState 跟踪流式中途状态

## 4. 关键实现细节
- 结构体/接口定义:
  - `PendingInterrupt`: 中断恢复状态，包含 `CheckpointID`、`InterruptID`、`Info`
  - `ChatService`: 核心聊天服务结构体，包含 Wails 上下文、代理实例、runner 实例、上下文引擎、沙箱管理器、配置存储、会话服务、取消函数、模式标识、检查点存储、待处理中断、互斥锁、`TimelineCollector`（后端时间线收集器）、`streamingState`（流式中途状态指针）
- 导出函数/方法:
  - `NewChatService(store) *ChatService`: 构造函数，默认 "default" 模式，初始化 TimelineCollector
  - `SetContext(ctx)`: 设置 Wails 应用上下文
  - `SetDependencies(sbx, ctxEngine)`: 注入沙箱和上下文引擎
  - `UpdateSandbox(sbx)`: 更新沙箱引用并使 runner 失效
  - `InvalidateRunner()`: 强制下次消息时重建 runner
  - `SetOnAgentDone(fn)`: 注册代理完成回调
  - `SetSessionService(ss)`: 注入会话服务
  - `SetMode(mode) error`: 切换运行模式（"default"/"plan"），发射 `agent:mode_changed` 事件
  - `GetMode() string`: 获取当前模式
  - `SendMessage(userMessage) error`: 发送用户消息，异步执行代理并流式处理事件。同时调用 `timeline.AddUserTurn()` 记录用户轮次
  - `ResumeWithAnswer(answer) error`: 以文本回答恢复中断执行
  - `ResumeWithChoice(selectedIndex) error`: 以选择恢复中断执行
  - `StopGeneration() error`: 取消当前代理执行
  - `ClearHistory() error`: 清除对话历史、重置 runner、清空 TimelineCollector 和 streamingState
  - `BuildRunners() error`: 构建 deep agent 和两种 runner，连接 MCP 服务器并加载工具
  - `Timeline() *agentctx.TimelineCollector`: 获取时间线收集器（供 SessionService 导出 display 数据）
  - `StreamingState() *model.StreamingState`: 获取当前流式中途状态（供 SessionService 保存中途快照）
- 未导出函数/方法:
  - `emitTimeline(evt map[string]interface{}, agentName string)`: 统一的时间线事件发射辅助方法，同时向前端发射 `agent:timeline` 事件并写入 `TimelineCollector`
  - `drainStream(stream, agentName)`: 流式输出消费方法，使用 50ms ticker 批量窗口合并 stream_chunk 事件，同时跟踪 `streamingState`
- Wails 绑定方法: `SendMessage`、`ResumeWithAnswer`、`ResumeWithChoice`、`StopGeneration`、`ClearHistory`、`SetMode`、`GetMode`、`BuildRunners`
- 事件发射:
  - `agent:timeline`: 统一时间线事件（消息、工具调用、工具结果、转移、中断、流式块、流式结束、信息、reasoning、thinking）
  - `agent:error`: 错误信息
  - `agent:done`: 代理执行完成
  - `agent:interrupt`: 中断事件（followup/choice）
  - `agent:mode_changed`: 模式切换

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/agent`: BuildDeepAgent、BuildDefaultRunner、BuildPlanRunner、AgentContext、DefaultAgentContext
  - `starxo/internal/config`: Store、AppConfig
  - `starxo/internal/context` (agentctx): Engine（上下文引擎，管理消息历史）、TimelineCollector（后端时间线收集器）
  - `starxo/internal/llm`: NewChatModel
  - `starxo/internal/logger`: 日志记录
  - `starxo/internal/model`: SessionData、StreamingState（流式中途状态）
  - `starxo/internal/sandbox`: SandboxManager
  - `starxo/internal/store` (checkpoint): NewInMemoryStore
  - `starxo/internal/tools`: ToolRegistry、RegisterBuiltinTools、ConnectMCPServer、LoadMCPTools、FollowUpInfo、ChoiceInfo
- 外部依赖:
  - `github.com/cloudwego/eino/adk`: Runner、Agent、AsyncIterator、AgentEvent、CheckPointID、ResumeParams
  - `github.com/cloudwego/eino/compose`: CheckPointStore
  - `github.com/cloudwego/eino/schema`: Message、ToolCall、ConcatMessages
  - `github.com/wailsapp/wails/v2/pkg/runtime` (wailsruntime): EventsEmit
  - `strings`: 流式 chunk 拼接
  - `time`: 50ms 批量窗口 ticker
- 关键配置: 通过 `config.Store` 获取 LLM 配置、SSH 配置、MCP 服务器配置

## 6. 变更影响面
- 事件格式变更直接影响前端 Vue 组件的事件监听和渲染
- runner 构建逻辑变更影响代理的初始化和工具可用性
- 中断处理逻辑变更影响用户与代理的交互流程
- 消息处理变更影响会话历史的完整性
- `processEvents` 将工具调用请求（assistant + ToolCalls）和工具执行结果（tool role）同步加入上下文历史，确保持久化后可完整恢复
- `processEvents` 中 ToolCalls 事件处理后使用 `continue` 跳过后续的 `allContents` 累积，避免将工具调用内容重复添加为孤立的 assistant 消息（否则会导致 LLM API 报 "No tool output found" 错误）
- `processEvents` 在工具调用事件前发射 reasoning 事件（Type: "reasoning"，当 msg.Content 非空时），在 transfer 事件后和子代理 tool_result 事件后发射 thinking 事件（Type: "thinking"）；transfer 事件附带 agentDescs 中文描述映射（code_writer/code_executor/file_manager），存入 ToolArgs 字段
- `processEvents` 包含 reasoning 文本存在/缺失的 debug 日志
- `processEvents` 通过 `pendingToolCalls` map 跟踪已存入的 tool_call_id，在事件循环结束后为未收到结果的孤立 tool_call 注入合成错误响应（"Error: tool execution failed or was interrupted"），防止 LLM API 因缺失 tool result 返回 400 错误
- Agent 执行过程中通过防抖机制（每 10 秒）中间保存会话，降低崩溃丢失风险
- `emitTimeline` 辅助方法确保每次 `agent:timeline` 事件同步写入 `TimelineCollector`，使后端成为唯一的 display 数据持久化生产者
- `drainStream` 使用 50ms ticker 窗口合并流式 chunk，将 IPC 频率从 10-50 次/秒降至 ~20 次/秒
- `drainStream` 同时维护 `streamingState`，使 `maybeSave` 能在流式中途保存包含部分内容的快照
- `Timeline()` 和 `StreamingState()` getter 被 `SessionService` 调用，用于构建统一的 `SessionData` 进行持久化
- 新增的 reasoning/thinking 事件类型影响前端 TimelineEventItem 的渲染逻辑
- 流式输出逻辑影响前端实时内容显示
- 是整个后端最核心的服务文件，变更需格外谨慎

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `processEvents` 方法较长（~150 行），如需扩展事件类型建议拆分为独立的事件处理函数。
- 并发安全通过 `sync.Mutex` 保证，修改锁的获取/释放逻辑时需特别注意死锁风险。
- `BuildRunners` 在锁内执行耗时操作（MCP 连接），可考虑优化为锁外执行。
- 流式输出的 `drainStream` 方法在 goroutine 中运行，使用 50ms ticker 批量窗口合并 chunk；修改批量间隔需权衡延迟与 IPC 频率。
- `emitTimeline` 是所有时间线事件的唯一出口，修改时需同步检查 `TimelineCollector.AddEvent()` 的事件分类逻辑。
- `streamingState` 在 `drainStream` 中更新、在 agent 完成后清除；`SessionService.saveCurrentLocked()` 读取该状态时需确保线程安全。
- 中断恢复（ResumeWithAnswer/ResumeWithChoice）共享大量重复代码，可考虑抽取公共方法。
