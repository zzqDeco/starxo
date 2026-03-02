# App.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/App.vue
- 文档文件: doc/src/frontend/src/App.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src (根组件)

## 2. 核心职责
- 应用根组件，负责全局主题配置、Wails 事件监听注册、以及初始数据恢复。
- 配置 Naive UI 深色主题和自定义主题覆盖（颜色、圆角、字体等）。
- 在 `onMounted` 中初始化所有 Wails 后端事件监听器，建立前后端通信桥梁。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Wails 事件（session:switched, sandbox:progress, sandbox:ready, sandbox:disconnected, agent:timeline, agent:done, agent:error, agent:interrupt, agent:plan, agent:mode_changed）、settingsStore/connectionStore/chatStore/sessionStore 状态
- 输出结果: 渲染 NConfigProvider 包裹的 MainLayout 组件；将 Wails 事件数据分发到对应 Store

## 4. 关键实现细节
- **主题配置**: `themeOverrides` 对象定义了完整的深色主题色板，包括 primaryColor (#22d3ee cyan)、bodyColor (#0c0e1a)、cardColor (#141726) 等，以及 Button/Input/Card/Modal/Tag/Dropdown/Collapse 组件级别的圆角覆盖
- **消息恢复**: `restoreActiveMessages()` 函数实现两级回退策略：
  1. 优先加载富显示数据（含时间线事件）`loadChatDisplay()`
  2. 回退到基础持久化消息 `loadActiveMessages()`
- **Wails 事件监听**: 使用 `EventsOn` 注册 9 类事件处理器：
  - `session:switched` → 切换活跃会话并恢复消息；无容器会话时清空 connectionStore 的 SSH/Docker 状态
  - `sandbox:progress/ready/disconnected` → 更新 connectionStore 连接状态
  - `agent:timeline` → 添加时间线事件到 chatStore
  - `agent:done` → 标记生成完成并持久化消息
  - `agent:error` → 显示错误消息
  - `agent:interrupt` → 设置中断状态（需用户交互）
  - `agent:plan` → 更新计划步骤
  - `agent:mode_changed` → 切换代理模式
- **Naive UI Provider 嵌套**: NConfigProvider > NMessageProvider > NDialogProvider > MainLayout

## 5. 依赖关系
- 内部依赖: `@/components/layout/MainLayout.vue`、`@/stores/settingsStore`、`@/stores/connectionStore`、`@/stores/chatStore`、`@/stores/sessionStore`、`@/types/session`、`@/types/message`
- 外部依赖: `naive-ui` (NConfigProvider, NMessageProvider, NDialogProvider, darkTheme, GlobalThemeOverrides)、`vue` (onMounted)
- Wails 绑定: `wailsjs/runtime/runtime` (EventsOn)

## 6. 变更影响面
- 修改主题覆盖会影响所有 Naive UI 组件的视觉表现
- 修改事件监听器会影响前后端数据同步
- 修改消息恢复逻辑会影响会话切换后的消息展示

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增 Wails 事件时需同步更新 Go 后端的事件发射代码和 `@/types/message` 类型定义。
- 主题色值修改需同步 `style.css` 中的 CSS 变量以保持一致性。
- 事件监听器数量较多，如需进一步拆分可考虑抽取到 composable 中。
