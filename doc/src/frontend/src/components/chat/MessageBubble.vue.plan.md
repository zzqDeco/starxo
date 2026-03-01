# MessageBubble.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/MessageBubble.vue
- 文档文件: doc/src/frontend/src/components/chat/MessageBubble.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat (聊天模块)

## 2. 核心职责
- 消息气泡组件，负责渲染 user/assistant/system 三种角色的消息。
- 对 assistant 消息实现分段时间线视图：将事件按 Agent 分组为 segments，支持 transfer 分隔、子代理折叠展开。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Props (message: Message)
- 输出结果: 渲染消息气泡 UI

## 4. 关键实现细节
- **Props 定义**: `message: Message`
- **Composable 使用**: `useMarkdown` — Markdown 渲染（用于无事件的 legacy 消息回退）
- **Agent 分段逻辑** (`segments` computed):
  - 遍历 message.events，按 agent 名称分组为 EventSegment
  - transfer 事件作为独立分隔段
  - 子代理段 (isSubAgent=true): 不在 mainAgents Set ('coding_agent', 'orchestrator', '') 中的 agent
  - 后处理：为子代理段向前查找 task tool_call 的 description 作为任务描述
- **子代理折叠/展开**:
  - 默认：已完成的段折叠，活跃的段展开
  - 用户可手动切换 (`subAgentToggled`)
  - 显示统计信息: tool calls 数 + messages 数
- **Agent 颜色/标签系统**: agentColor() / agentLabel() / agentIconType() 按 agent 名称映射颜色、显示名和图标类型
- **模板结构**:
  - System: 红色提示条
  - User: 右对齐蓝色气泡，显示纯文本
  - Assistant: 头部 (avatar + time + copy) → legacy content (无事件时) → segments 时间线
    - Transfer segment: 分隔线 (agent A → agent B)
    - Sub-agent segment: 可折叠卡片 (header + task desc + events)
    - Main agent segment: 带颜色条的事件列表

## 5. 依赖关系
- 内部依赖: `@/composables/useHelpers` (useMarkdown)、`@/types/message` (Message, TurnEvent)、`./TimelineEventItem.vue`
- 外部依赖: `vue` (computed, ref)、`naive-ui` (NIcon)、`@vicons/ionicons5` (多个图标)、`vue-i18n` (useI18n)

## 6. 变更影响面
- Agent 分段逻辑修改影响 assistant 消息的展示结构
- Agent 颜色系统修改需同步 TimelineEventItem 中的 agentColor
- 子代理折叠行为修改影响用户交互体验

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- segments 计算逻辑较复杂（~50行），新增 agent 类型时需更新 mainAgents Set 和 agentLabel 映射。
- 考虑将 agentColor/agentLabel 等辅助函数提取到 composable 中，以便与 TimelineEventItem 共享。
