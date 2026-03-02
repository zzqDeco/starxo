# ChatPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/ChatPanel.vue
- 文档文件: doc/src/frontend/src/components/chat/ChatPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat (聊天模块)

## 2. 核心职责
- 聊天面板主容器，组合消息列表、输入区域、中断对话框、计划面板和 Agent 状态指示器。
- 处理消息发送、停止生成、自动滚动和空状态展示。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: chatStore 状态（visibleMessages, isStreaming, currentAgent）、connectionStore 状态（isReady）
- 输出结果: 渲染完整聊天界面；通过 Wails 绑定调用 SendMessage/StopGeneration

## 4. 关键实现细节
- **Props/Emits**: 无（作为页面级容器组件）
- **Pinia Store 交互**:
  - chatStore: visibleMessages, isStreaming, currentAgent, addUserMessage, setGenerating, addMessage
  - connectionStore: isReady
- **Composable 使用**: `useAutoScroll` — 管理自动滚动和底部检测
- **Wails 绑定调用**: `SendMessage(content)` 发送消息、`StopGeneration()` 停止生成
- **关键模板结构**:
  - 空状态: 显示图标、标题、连接状态提示文案和 3 个提示卡片（可点击直接发送）
  - 消息列表: v-for 渲染 MessageBubble，使用 visibleMessages（过滤空消息）
  - PlanPanel: 始终渲染（内部根据 hasSteps 决定是否显示）
  - AgentStatus: isStreaming 时显示
  - 滚动到底部按钮: 非底部时显示，带闪烁动画提示新消息
  - InterruptDialog: 始终渲染（内部根据 interrupt 状态决定是否显示）
  - TodoBoard (persistent): 当 `chatStore.latestTodos.length > 0` 时在输入区域上方常驻显示，使用 compact 模式，带 todo-panel 过渡动画
  - InputArea: 底部输入框，接收 send/stop 事件
- **自动滚动逻辑**: 监听 messages.length 和最后消息的 events.length 变化，autoScroll 模式下自动滚动到底部
- **提示卡片**: 点击 hint-card 会将国际化文本作为消息发送

## 5. 依赖关系
- 内部依赖: `@/stores/chatStore`、`@/stores/connectionStore`、`@/composables/useHelpers` (useAutoScroll)、`./MessageBubble.vue`、`./InputArea.vue`、`./InterruptDialog.vue`、`./PlanPanel.vue`、`./TodoBoard.vue`、`@/components/status/AgentStatus.vue`
- 外部依赖: `vue` (ref, watch, nextTick, computed)、`naive-ui` (NIcon)、`@vicons/ionicons5` (ArrowDown)、`vue-i18n` (useI18n)
- Wails 绑定: `wailsjs/go/service/ChatService` (SendMessage, StopGeneration)

## 6. 变更影响面
- 修改消息发送流程影响整个聊天交互
- 修改空状态提示需同步国际化文件
- 自动滚动逻辑修改影响用户阅读体验

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增子组件时注意保持组合模式，ChatPanel 作为容器不应包含过多业务逻辑。
- SendMessage 调用失败时需确保 UI 状态正确恢复。
