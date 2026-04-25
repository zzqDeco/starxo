# message.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/types/message.ts
- 文档文件: doc/src/frontend/src/types/message.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/types (类型定义)

## 2. 核心职责
- 定义聊天系统核心数据类型，包括消息、时间线事件、中断交互和计划模式的类型接口。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 无（纯类型定义）
- 输出结果: 导出类型接口供 stores 和 components 使用

## 4. 关键实现细节
- **导出类型/接口**:
  - `TurnEvent` — 时间线事件，type 字段区分 10 种类型: message / tool_call / tool_result / transfer / info / interrupt / plan / stream_chunk / stream_end / reasoning / thinking。包含可选字段 toolName/toolArgs/toolId/toolResult/isStreaming。**新增 `sessionId?: string`** 字段，用于标识事件所属会话，前端据此过滤非活跃会话的事件
  - `Message` — 聊天消息，role 区分 user/assistant/system，包含 events 数组（TurnEvent[]）和可选 agent/isStreaming
  - `TerminalOutputEvent` — 终端输出事件（stdout/stderr/exitCode）
  - `PersistedMessage` — 持久化消息（简化格式，用于后端存储回退）
  - `InterruptEvent` — 中断事件，type 区分 followup（追问）和 choice（选择），包含 interruptId/checkpointId 和问题/选项数据。**新增 `sessionId?: string`** 字段
  - `InterruptOption` — 中断选项（label + description）
  - `PlanEvent` — 计划事件，包含步骤列表
  - `PlanStepDTO` — 计划步骤，status 区分 todo/doing/done/failed/skipped，包含 taskId/desc/execResult
  - `ModeChangedEvent` — 模式切换事件，mode 为 default 或 plan。**新增 `sessionId?: string`** 字段
  - `SessionRunState` — 会话运行态，包含 `sessionId/running/currentAgent/mode/hasInterrupt`，对应后端 `agent:run_state`

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: 无（纯类型定义）

## 6. 变更影响面
- `TurnEvent` 的 `sessionId` 字段被 `App.vue` 的 `isActiveSession()` 过滤函数使用，决定是否将事件路由到 chatStore
- `InterruptEvent` 的 `sessionId` 字段被 `App.vue` 的 `agent:interrupt` 事件处理器用于过滤
- `ModeChangedEvent` 的 `sessionId` 字段被 `App.vue` 的 `agent:mode_changed` 事件处理器用于过滤
- `SessionRunState` 被 `App.vue` 的 `agent:run_state` 事件处理器写入 chatStore，供 Sidebar 显示后台运行/中断状态
- `TurnEvent` 修改影响 chatStore.addTimelineEvent、TimelineEventItem、MessageBubble
- `Message` 修改影响 chatStore、MessageBubble、ChatPanel
- `InterruptEvent` 修改影响 chatStore.setInterrupt、InterruptDialog
- `PlanStepDTO` 修改影响 chatStore.updatePlanSteps、PlanPanel
- 类型需与 Go 后端发送的事件数据结构保持一致（`internal/service/events.go`）

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增 TurnEvent type 值时需同步更新 chatStore.addTimelineEvent 的处理逻辑。
- 类型变更需确保与 Go 后端 Wails 事件的 JSON 结构保持一致。
- `sessionId` 字段为可选（`?`），向后兼容旧版不携带 sessionId 的事件。
