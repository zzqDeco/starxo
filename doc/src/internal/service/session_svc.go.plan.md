# session_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/session_svc.go
- 文档文件: doc/src/internal/service/session_svc.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `SessionService`，负责管理聊天会话的完整生命周期。提供会话的创建、切换、删除、重命名、保存和加载功能。每个会话关联一个容器（通过 ContainerID 绑定）和工作区路径。会话切换时自动保存当前会话、加载目标会话的消息历史到上下文引擎、并通知沙箱服务进行容器重连。还提供增强版会话列表（包含容器状态信息）和前端显示数据的持久化。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `CreateSession(title)`、`SwitchSession(sessionID)`、`DeleteSession(sessionID)`、`RenameSession(sessionID, title)`、`ListSessions()`、`ListSessionsEnriched()`、`GetActiveSession()`、`GetActiveSessionMessages()`、`SaveChatDisplay(data)`、`LoadChatDisplay()`、`SaveCurrentSession()`
  - 依赖注入: `storage.SessionStore`、`storage.ContainerStore`、`agentctx.Engine`
- 输出结果:
  - Wails 事件发射: `session:switched`（会话切换完成）
  - 回调通知: `onSessionSwitch`（会话切换时传递容器注册 ID）
  - 会话数据持久化到 SessionStore

## 4. 关键实现细节
- 结构体/接口定义:
  - `SessionService`: 会话服务结构体，包含 Wails 上下文、SessionStore、ContainerStore、上下文引擎、活动会话、会话切换回调、互斥锁
  - `EnrichedSession`: 扩展会话类型，内嵌 `model.Session` 并添加 `ContainerStatus`、`ContainerName`、`ContainerSSH` 字段
- 导出函数/方法:
  - `NewSessionService(sessionStore, containerStore) *SessionService`: 构造函数
  - `SetContext(ctx)`: 设置 Wails 上下文
  - `SetCtxEngine(engine)`: 设置上下文引擎
  - `SetOnSessionSwitch(fn)`: 注册会话切换回调
  - `BindContainer(containerRegID, workspacePath)`: 将容器绑定到当前会话
  - `GetBoundContainerID() string`: 获取当前会话绑定的容器 ID
  - `GetWorkspacePath() string`: 获取当前会话工作区路径，默认 "/workspace"
  - `ListSessions() ([]model.Session, error)`: 列出所有会话
  - `CreateSession(title) (*model.Session, error)`: 创建新会话，自动保存当前会话并清空历史
  - `SwitchSession(sessionID) error`: 切换会话，保存当前、加载目标、恢复消息、发射事件、通知回调
  - `DeleteSession(sessionID) error`: 删除会话（不能删除活动会话）
  - `RenameSession(sessionID, title) error`: 重命名会话
  - `GetActiveSession() *model.Session`: 获取当前活动会话（返回副本）
  - `GetActiveSessionMessages() ([]model.PersistedMessage, error)`: 获取活动会话消息
  - `SaveChatDisplay(data) error`: 保存前端显示数据
  - `LoadChatDisplay() (string, error)`: 加载前端显示数据
  - `SaveCurrentSession() error`: 持久化当前会话
  - `EnsureDefaultSession() error`: 确保存在默认会话，加载最近会话
  - `ListSessionsEnriched() ([]EnrichedSession, error)`: 列出包含容器信息的增强会话列表
- Wails 绑定方法: `CreateSession`、`SwitchSession`、`DeleteSession`、`RenameSession`、`ListSessions`、`ListSessionsEnriched`、`GetActiveSession`、`GetActiveSessionMessages`、`SaveChatDisplay`、`LoadChatDisplay`、`SaveCurrentSession`
- 事件发射: `session:switched`

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/context` (agentctx): Engine（消息历史管理）
  - `starxo/internal/model`: Session、PersistedMessage
  - `starxo/internal/storage`: SessionStore、ContainerStore
- 外部依赖:
  - `github.com/wailsapp/wails/v2/pkg/runtime` (wailsruntime): EventsEmit
- 关键配置: 无

## 6. 变更影响面
- 会话切换逻辑影响 ChatService（通过 `onSessionSwitch` 触发沙箱重连）
- `BindContainer` 被 SandboxService 的 `onContainerBound` 回调调用
- `GetWorkspacePath` 被 ChatService 和 FileService 使用
- `SaveCurrentSession` 被 ChatService 的 `onAgentDone` 回调触发
- `EnsureDefaultSession` 在应用启动时由 `app.go` 调用
- 会话存储格式变更影响已有会话数据的加载

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `SwitchSession` 中的锁释放和重获取逻辑（调用 `onSessionSwitch` 时临时释放锁）需特别注意，防止竞态条件。
- `GetActiveSession` 返回会话副本以避免外部修改，应保持此模式。
- 前端显示数据（`SaveChatDisplay`/`LoadChatDisplay`）以 raw string 存储，格式由前端定义。
- `EnrichedSession` 的容器信息查询可能在大量会话时产生 N+1 查询问题，可考虑批量查询优化。
