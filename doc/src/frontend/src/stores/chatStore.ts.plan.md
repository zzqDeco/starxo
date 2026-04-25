# chatStore.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/stores/chatStore.ts
- 文档文件: doc/src/frontend/src/stores/chatStore.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/stores (Pinia 状态管理)

## 2. 核心职责
- 管理聊天消息列表、Agent 流式生成状态、中断交互状态和计划模式状态。
- 提供时间线事件的聚合与合并逻辑（流式消息累积、工具结果匹配）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: App.vue 中的 Wails 事件回调、ChatPanel.vue 用户操作
- 输出结果: 响应式状态供 ChatPanel、MessageBubble、InterruptDialog、PlanPanel 等组件消费

## 4. 关键实现细节
- **导出类型**: `TodoItem` 接口 — id, title, status (pending/in_progress/done/failed/blocked), depends_on?: string[]
- **State 属性**:
  - `messages: Message[]` — 消息列表
  - `isStreaming: boolean` — 是否正在流式生成
  - `currentAgent: string` — 当前活跃 Agent 名称
  - `agentDone: boolean` — Agent 是否已完成当前轮次
  - `activeTurnId: string | null` — 当前助手回复轮次的消息 ID
  - `pendingInterrupt: InterruptEvent | null` — 待处理的中断事件
  - `agentMode: 'default' | 'plan'` — 当前代理模式
  - `planSteps: PlanStepDTO[]` — 计划步骤列表
  - `latestTodos: TodoItem[]` — 最新 Todo 任务快照，从 write_todos/update_todo 工具事件中提取，供 ChatPanel 常驻 TodoBoard 消费
  - `sessionRunStates: Record<string, SessionRunState>` — 按 sessionId 保存后台运行态（running/currentAgent/mode/hasInterrupt）
- **Getters**:
  - `lastMessage` — 最后一条消息
  - `visibleMessages` — 过滤掉无内容且无事件的空助手消息
  - `hasInterrupt` — 是否有待处理中断
- **Actions**:
  - `getOrCreateTurn()` — 获取或创建当前助手回复消息，实现"一轮对话一个消息"模型
  - `addTimelineEvent(evt)` — 核心方法，处理 stream_chunk（累积到已有流式消息）、stream_end（标记流式结束）、tool_result（匹配到对应 tool_call 并调用 tryUpdateTodosFromResult）、tool_call(write_todos) 时提取 todos 到 latestTodos、thinking 事件管理（替换同一 agent 的前一个 thinking 事件，收到非 thinking 事件时清除已有 thinking）
  - `tryUpdateTodosFromResult(evt)` — 从 update_todo 工具结果中解析更新后的 todos 快照（以 `---\n` 分隔，取最后部分 JSON 解析）
  - `addUserMessage(content)` — 添加用户消息并重置轮次状态
  - `setInterrupt(evt)` / `clearInterrupt()` — 中断状态管理
  - `updatePlanSteps(steps)` / `setMode(mode)` — 计划模式管理
  - `setSessionRunState(state)` / `getSessionRunState(sessionId)` — 会话运行态管理，供 Sidebar 和 App.vue 使用
  - `setGenerating(generating, agent?)` — 更新生成状态
  - `restoreTodosFromMessages()` — 扫描所有已恢复消息的 events，提取最新的 todos 快照用于会话恢复
  - `clearMessages()` — 清空所有状态（含 latestTodos）

## 5. 依赖关系
- 内部依赖: `@/types/message` (Message, TurnEvent, InterruptEvent, PlanStepDTO)
- 外部依赖: `pinia` (defineStore)、`vue` (ref, computed)

## 6. 变更影响面
- 修改消息结构影响 MessageBubble、TimelineEventItem 的渲染
- 修改中断状态影响 InterruptDialog 组件
- 修改计划步骤影响 PlanPanel 组件
- `visibleMessages` 过滤逻辑影响 ChatPanel 消息列表显示
- `latestTodos` 状态影响 ChatPanel 中常驻 TodoBoard 的显示
- `sessionRunStates` 状态影响 Sidebar 的后台运行、中断等待和模式徽标

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `addTimelineEvent` 方法较复杂，新增事件类型时需仔细处理事件合并逻辑。
- 确保 `activeTurnId` 生命周期管理正确，避免消息丢失或重复。
