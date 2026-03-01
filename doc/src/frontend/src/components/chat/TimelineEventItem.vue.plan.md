# TimelineEventItem.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/TimelineEventItem.vue
- 文档文件: doc/src/frontend/src/components/chat/TimelineEventItem.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat (聊天模块)

## 2. 核心职责
- 时间线事件项组件，负责渲染单个 TurnEvent 的可视化表示。
- 按事件类型和工具类别提供差异化的渲染样式（消息、工具调用、transfer、中断、信息）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Props (event: TurnEvent, showAgentBadge?: boolean)
- 输出结果: 渲染事件 UI

## 4. 关键实现细节
- **Props 定义**:
  - `event: TurnEvent` — 时间线事件
  - `showAgentBadge?: boolean` — 是否显示 Agent 标签（默认 true）
- **Composable 使用**: `useMarkdown` — 消息内容的 Markdown 渲染
- **工具分类系统** (`toolInfo` computed): 按 toolName 分为 6 类:
  - `file`: read_file, write_file, list_files — 绿色 (#34d399)
  - `edit`: str_replace_editor — 蓝色 (#38bdf8)
  - `shell`: shell_execute, python_execute — 紫色 (#a78bfa)
  - `agent`: task（子代理委派） — 青色 (#22d3ee)
  - `todo`: write_todos, update_todo — 黄色 (#f59e0b)
  - `notify`: notify_user — 青色 (#22d3ee)
  - `other`: 未分类工具 — 黄色 (#f59e0b)
- **特殊渲染**:
  - `notify_user`: 内联状态横幅（不使用折叠面板）
  - `task` (agent): 委派卡片（显示子代理名称 + 描述 + 状态）
  - `write_todos` / `update_todo`: 渲染 TodoBoard 组件
  - 标准 tool_call: NCollapse 折叠面板（活跃时展开，完成时折叠），显示参数和结果
- **结果截断**: 超过 500 字符的 toolResult 截断并提供"展开全部"按钮
- **Todo 解析**: write_todos 从 toolArgs 解析，update_todo 从 toolResult 中 `---\n` 分隔符后解析

## 5. 依赖关系
- 内部依赖: `@/composables/useHelpers` (useMarkdown)、`@/types/message` (TurnEvent)、`./TodoBoard.vue` (TodoItem 类型和组件)
- 外部依赖: `vue` (computed, ref)、`naive-ui` (NIcon, NCollapse, NCollapseItem, NButton)、`@vicons/ionicons5` (多个图标)、`vue-i18n` (useI18n)

## 6. 变更影响面
- 工具分类修改影响所有 tool_call 事件的渲染
- 新增工具类型需在 toolInfo computed 中添加分支
- 结果截断阈值修改影响长输出的展示

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- toolInfo computed 使用 if-else 链匹配工具名，新增工具时在对应类别添加条件。
- agentColor/agentLabel 函数与 MessageBubble 中重复，可考虑提取共享。
- parsedTodos 的 JSON 解析使用 try-catch 容错，确保格式变更不会导致崩溃。
