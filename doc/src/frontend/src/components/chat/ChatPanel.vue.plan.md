# ChatPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/ChatPanel.vue
- 文档文件: doc/src/frontend/src/components/chat/ChatPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat

## 2. 核心职责
- 聊天主容器，负责消息流渲染、发送/停止、模式切换、自动滚动和中断交互。
- 作为工作台中央执行画布，在底部区域组合“任务浮层 + composer”，统一与消息区栅格对齐。

## 3. 输入与输出
- 输入来源: `chatStore`、`connectionStore`、Wails ChatService
- 输出结果: 渲染聊天界面并发起 `SendMessage/SetMode/StopGeneration`

## 4. 关键实现细节
- 模式切换:
  - 通过 `SetMode('default'|'plan')` 与后端同步
  - 流式生成中禁用切换
- 消息区:
  - `visibleMessages` 渲染 `MessageBubble`
  - `AgentStatus` 在流式时显示
- 空状态显示沙箱就绪/待连接状态、运行时/工作区/隔离能力摘要，以及快捷提示卡片（点击可直接发送）
- 底部区:
  - `InterruptDialog`
  - `TaskRailFloating`（任务摘要浮层）
  - `InputArea`（接收当前 agent 模式和切换 loading 状态，模式切换入口已下沉到 composer）
- 自动滚动:
  - 使用 `useAutoScroll`
  - 通过 `ResizeObserver` 动态计算“回到底部”按钮偏移
- 栅格对齐:
  - `--chat-content-max-width` 与 `--chat-content-padding` 统一控制消息区/底部区对齐

## 5. 依赖关系
- 内部依赖:
  - `MessageBubble.vue`, `InputArea.vue`, `InterruptDialog.vue`
  - `TaskRailFloating.vue`, `AgentStatus.vue`
  - `chatStore`, `connectionStore`, `useAutoScroll`
- 外部依赖:
  - `vue`, `naive-ui`, `@vicons/ionicons5`, `vue-i18n`
  - Wails: `SendMessage`, `SetMode`, `StopGeneration`

## 6. 变更影响面
- 旧的 `PlanPanel` 与持久 `TodoBoard` 从聊天主流中移除。
- 模式切换入口从顶部工具条移动到 composer，减少消息流上方的固定占用。
- 任务信息改为输入区上方浮层，减少消息区视觉干扰。

## 7. 维护建议
- 保持 ChatPanel 作为编排层，不在此组件堆积复杂业务状态。
- 调整底部结构时需回归验证滚动按钮定位逻辑。
