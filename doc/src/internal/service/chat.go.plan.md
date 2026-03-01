# chat.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/chat.go
- 文档文件: doc/src/internal/service/chat.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `ChatService`，是前端与 AI 代理之间的核心桥梁。它管理聊天交互的完整生命周期：构建代理和 runner、发送用户消息、流式处理代理事件、处理中断（follow-up 问题和选择题）、恢复执行、停止生成、清除历史。支持 "default" 和 "plan" 两种运行模式。所有代理事件（消息、工具调用、转移、中断、流式输出）都通过 Wails 事件系统转发到前端。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `SendMessage(userMessage)`、`ResumeWithAnswer(answer)`、`ResumeWithChoice(selectedIndex)`、`StopGeneration()`、`ClearHistory()`、`SetMode(mode)`、`GetMode()`
  - 依赖注入: `config.Store`、`sandbox.SandboxManager`、`agentctx.Engine`、`SessionService`
- 输出结果:
  - Wails 事件发射: `agent:message`、`agent:timeline`、`agent:action`、`agent:tool_result`、`agent:error`、`agent:done`、`agent:interrupt`、`agent:mode_changed`
  - 状态变更: 代理执行、会话历史更新

## 4. 关键实现细节
- 结构体/接口定义:
  - `PendingInterrupt`: 中断恢复状态，包含 `CheckpointID`、`InterruptID`、`Info`
  - `ChatService`: 核心聊天服务结构体，包含 Wails 上下文、代理实例、runner 实例、上下文引擎、沙箱管理器、配置存储、会话服务、取消函数、模式标识、检查点存储、待处理中断、互斥锁
- 导出函数/方法:
  - `NewChatService(store) *ChatService`: 构造函数，默认 "default" 模式
  - `SetContext(ctx)`: 设置 Wails 应用上下文
  - `SetDependencies(sbx, ctxEngine)`: 注入沙箱和上下文引擎
  - `UpdateSandbox(sbx)`: 更新沙箱引用并使 runner 失效
  - `InvalidateRunner()`: 强制下次消息时重建 runner
  - `SetOnAgentDone(fn)`: 注册代理完成回调
  - `SetSessionService(ss)`: 注入会话服务
  - `SetMode(mode) error`: 切换运行模式（"default"/"plan"），发射 `agent:mode_changed` 事件
  - `GetMode() string`: 获取当前模式
  - `SendMessage(userMessage) error`: 发送用户消息，异步执行代理并流式处理事件
  - `ResumeWithAnswer(answer) error`: 以文本回答恢复中断执行
  - `ResumeWithChoice(selectedIndex) error`: 以选择恢复中断执行
  - `StopGeneration() error`: 取消当前代理执行
  - `ClearHistory() error`: 清除对话历史并重置 runner
  - `BuildRunners() error`: 构建 deep agent 和两种 runner，连接 MCP 服务器并加载工具
- Wails 绑定方法: `SendMessage`、`ResumeWithAnswer`、`ResumeWithChoice`、`StopGeneration`、`ClearHistory`、`SetMode`、`GetMode`、`BuildRunners`
- 事件发射:
  - `agent:message`: 完整消息（非流式）
  - `agent:timeline`: 统一时间线事件（消息、工具调用、工具结果、转移、中断、流式块、流式结束、信息）
  - `agent:action`: 代理动作（tool_call、transfer、info）
  - `agent:tool_result`: 工具执行结果
  - `agent:error`: 错误信息
  - `agent:done`: 代理执行完成
  - `agent:interrupt`: 中断事件（followup/choice）
  - `agent:mode_changed`: 模式切换

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/agent`: BuildDeepAgent、BuildDefaultRunner、BuildPlanRunner、AgentContext、DefaultAgentContext
  - `starxo/internal/config`: Store、AppConfig
  - `starxo/internal/context` (agentctx): Engine（上下文引擎，管理消息历史）
  - `starxo/internal/llm`: NewChatModel
  - `starxo/internal/logger`: 日志记录
  - `starxo/internal/sandbox`: SandboxManager
  - `starxo/internal/store` (checkpoint): NewInMemoryStore
  - `starxo/internal/tools`: ToolRegistry、RegisterBuiltinTools、ConnectMCPServer、LoadMCPTools、FollowUpInfo、ChoiceInfo
- 外部依赖:
  - `github.com/cloudwego/eino/adk`: Runner、Agent、AsyncIterator、AgentEvent、CheckPointID、ResumeParams
  - `github.com/cloudwego/eino/compose`: CheckPointStore
  - `github.com/cloudwego/eino/schema`: Message、ToolCall、ConcatMessages
  - `github.com/wailsapp/wails/v2/pkg/runtime` (wailsruntime): EventsEmit
- 关键配置: 通过 `config.Store` 获取 LLM 配置、SSH 配置、MCP 服务器配置

## 6. 变更影响面
- 事件格式变更直接影响前端 Vue 组件的事件监听和渲染
- runner 构建逻辑变更影响代理的初始化和工具可用性
- 中断处理逻辑变更影响用户与代理的交互流程
- 消息处理变更影响会话历史的完整性
- 流式输出逻辑影响前端实时内容显示
- 是整个后端最核心的服务文件，变更需格外谨慎

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `processEvents` 方法较长（~150 行），如需扩展事件类型建议拆分为独立的事件处理函数。
- 并发安全通过 `sync.Mutex` 保证，修改锁的获取/释放逻辑时需特别注意死锁风险。
- `BuildRunners` 在锁内执行耗时操作（MCP 连接），可考虑优化为锁外执行。
- 流式输出的 `drainStream` 方法在 goroutine 中运行，需确保 stream 的正确关闭。
- 中断恢复（ResumeWithAnswer/ResumeWithChoice）共享大量重复代码，可考虑抽取公共方法。
