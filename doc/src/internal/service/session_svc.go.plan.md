# session_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/session_svc.go
- 文档文件: doc/src/internal/service/session_svc.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `SessionService`，负责管理聊天会话的完整生命周期。提供会话的创建、切换、删除、重命名、保存和加载功能。每个会话拥有多个容器（通过 Containers 列表和 ActiveContainerID 管理），形成严格的父子关系。会话切换时自动保存当前会话、加载目标会话的消息历史到 per-session 上下文引擎、并通知沙箱服务进行容器重连。删除会话时先停止运行中的代理，然后级联销毁所有子容器。还提供增强版会话列表（包含容器状态信息）、统一的会话数据加载（`LoadSessionData`，返回 `model.SessionData`），以及前端显示数据的持久化。
- **核心变更**: 不再持有独立的 `ctxEngine` 字段，改为通过 `ChatService` 的 per-session 方法（`SessionCtxEngine`、`SessionTimeline`、`SessionStreamingState`）访问会话状态。会话切换时通过 `ChatService.GetOrCreateRun()` 加载数据到 per-session run，通过 `GetSessionRunSnapshot()` 获取富状态快照用于 `SessionSwitchedEvent`。删除会话时先通过 `StopSessionGeneration` + `WaitForSessionDone` 停止代理，再 `RemoveSession` 清理状态。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `CreateSession(title)`、`SwitchSession(sessionID)`、`DeleteSession(sessionID)`、`RenameSession(sessionID, title)`、`ListSessions()`、`ListSessionsEnriched()`、`GetActiveSession()`、`GetActiveSessionMessages()`、`SaveChatDisplay(data)`、`LoadChatDisplay()`、`SaveCurrentSession()`、`LoadSessionData()`
  - 依赖注入: `storage.SessionStore`、`storage.ContainerStore`、`ChatService`
- 输出结果:
  - Wails 事件发射: `session:switched`（会话切换完成，包含完整状态快照）
  - 回调通知: `onSessionSwitch`（会话切换时传递容器注册 ID）
  - 会话数据持久化到 SessionStore

## 4. 关键实现细节
- 结构体/接口定义:
  - `SessionService`: 会话服务结构体，包含 Wails 上下文、SessionStore、ContainerStore、`chatService *ChatService`（per-session 状态访问）、活动会话、会话切换回调、容器销毁回调 (`onDestroyContainer`)、互斥锁。**不再持有独立的 `ctxEngine` 字段**
  - `EnrichedSession`: 扩展会话类型，内嵌 `model.Session` 并添加 `ContainerStatus`、`ContainerName`、`ContainerSSH` 字段（基于 ActiveContainerID 查询）
- 导出函数/方法:
  - `NewSessionService(sessionStore, containerStore) *SessionService`: 构造函数
  - `SetContext(ctx)`: 设置 Wails 上下文
  - `SetChatService(cs *ChatService)`: 注入 ChatService 引用（用于 per-session 状态访问）
  - `SetOnSessionSwitch(fn)`: 注册会话切换回调
  - `SetOnDestroyContainer(fn)`: 注册容器销毁回调（级联删除时调用）
  - `BindContainer(containerRegID, workspacePath)`: 将容器绑定到当前会话
  - `GetBoundContainerID() string`: 获取当前会话绑定的容器 ID
  - `GetWorkspacePath() string`: 获取当前会话工作区路径，默认 "/workspace"
  - `ListSessions() ([]model.Session, error)`: 列出所有会话
  - `CreateSession(title) (*model.Session, error)`: 创建新会话，自动保存当前会话，调用 `chatService.SetActiveSessionID()` 切换活跃会话，调用 `tools.ClearTodos()` 清除 todo 状态
  - `SwitchSession(sessionID) error`: 切换会话，保存当前会话，加载目标会话数据到 per-session run（仅当会话未在运行时从磁盘加载），发射包含完整状态快照的 `SessionSwitchedEvent`，通知回调
  - `DeleteSession(sessionID) error`: 删除会话，先通过 `chatService.StopSessionGeneration` + `WaitForSessionDone` 停止运行中的代理，通过 `chatService.RemoveSession` 清理会话状态，然后级联销毁所有子容器
  - `RenameSession(sessionID, title) error`: 重命名会话
  - `GetActiveSession() *model.Session`: 获取当前活动会话（返回副本）
  - `GetActiveSessionMessages() ([]model.PersistedMessage, error)`: 获取活动会话消息
  - `SaveChatDisplay(data) error`: 保存前端显示数据（旧版接口）
  - `LoadChatDisplay() (string, error)`: 加载前端显示数据（旧版接口）
  - `SaveCurrentSession() error`: 持久化当前会话
  - `EnsureDefaultSession() error`: 确保存在默认会话，加载最近会话，通过 `chatService.GetOrCreateRun()` 恢复 messages 和 timeline 到 per-session 状态
  - `ListSessionsEnriched() ([]EnrichedSession, error)`: 列出包含容器信息的增强会话列表
  - `LoadSessionData() (*model.SessionData, error)`: 加载当前活跃会话的统一会话数据
- 未导出函数/方法:
  - `saveCurrentLocked() error`: 通过 `chatService.SessionCtxEngine(sessionID)` 获取上下文引擎，通过 `chatService.SessionTimeline(sessionID)` 获取时间线，通过 `chatService.SessionStreamingState(sessionID)` 获取流式状态，构建统一 `SessionData` 进行原子持久化
- Wails 绑定方法: `CreateSession`、`SwitchSession`、`DeleteSession`、`RenameSession`、`ListSessions`、`ListSessionsEnriched`、`GetActiveSession`、`GetActiveSessionMessages`、`SaveChatDisplay`、`LoadChatDisplay`、`SaveCurrentSession`、`LoadSessionData`
- 事件发射: `session:switched`（携带 `SessionSwitchedEvent`，包含 `AgentRunning`、`CurrentAgent`、`Mode`、`HasInterrupt`、`Interrupt` 等状态字段）

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/model`: Session、PersistedMessage、SessionData、DisplayTurn
  - `starxo/internal/storage`: SessionStore（含 SaveSessionData/LoadSessionData）、ContainerStore
  - `starxo/internal/tools`: ClearTodos（会话创建和切换时清除 todo 状态）
  - `starxo/internal/service`: ChatService（per-session 状态访问）
- 外部依赖:
  - `github.com/wailsapp/wails/v2/pkg/runtime` (wailsruntime): EventsEmit
- 关键配置: 无
- **不再依赖**: `starxo/internal/context` (agentctx) — 通过 ChatService 间接访问

## 6. 变更影响面
- 会话切换逻辑通过 `ChatService.SetActiveSessionID()` 切换活跃会话，通过 `ChatService.GetOrCreateRun()` 加载数据到 per-session run
- `SwitchSession` 检查 `chatService.IsSessionRunning()` — 运行中的会话不从磁盘重新加载（保留内存中的最新状态）
- `SwitchSession` 发射包含完整状态快照的 `SessionSwitchedEvent`（通过 `chatService.GetSessionRunSnapshot()`）
- `DeleteSession` 先停止运行中的代理（`StopSessionGeneration` + `WaitForSessionDone`），再清理状态（`RemoveSession`），最后级联销毁容器
- `CreateSession` 调用 `chatService.SetActiveSessionID()` 和 `tools.ClearTodos()`
- `saveCurrentLocked` 通过 `chatService.SessionCtxEngine()` / `SessionTimeline()` / `SessionStreamingState()` 访问 per-session 状态
- `EnsureDefaultSession` 通过 `chatService.GetOrCreateRun()` 加载数据到 per-session 状态
- `BindContainer` 被 SandboxService 的 `onContainerBound` 回调调用
- `GetWorkspacePath` 被 ChatService 和 FileService 使用
- `SaveCurrentSession` 被 ChatService 的 `onAgentDone` 回调触发

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `SwitchSession` 中的锁释放和重获取逻辑（调用 `onSessionSwitch` 时临时释放锁）需特别注意，防止竞态条件。
- `GetActiveSession` 返回会话副本以避免外部修改，应保持此模式。
- `saveCurrentLocked` 依赖 `chatService` 不为 nil 来获取 per-session 状态；若 `chatService` 未注入，则跳过保存。确保 `app.go` 中 `SetChatService` 在任何保存操作之前调用。
- `SwitchSession` 中 `IsSessionRunning` 检查确保不覆盖运行中会话的内存状态——修改此逻辑需谨慎。
- 前端显示数据（`SaveChatDisplay`/`LoadChatDisplay`）为旧版接口，保留向后兼容，新代码应使用 `LoadSessionData`。
- `EnrichedSession` 的容器信息查询可能在大量会话时产生 N+1 查询问题，可考虑批量查询优化。
