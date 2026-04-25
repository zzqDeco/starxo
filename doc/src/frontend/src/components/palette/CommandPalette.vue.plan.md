# CommandPalette.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/palette/CommandPalette.vue
- 文档文件: doc/src/frontend/src/components/palette/CommandPalette.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/palette

## 2. 核心职责
- 全局命令面板，提供键盘优先的会话、设置、工作区、SSH 连接和模式切换入口。

## 3. 输入与输出
- 输入来源: `Props.show`、session/chat/connection stores
- 输出结果: `update:show`、`open-settings`、`open-workspace`

## 4. 关键实现细节
- 通过 `useFocusTrap` 保持弹窗内键盘焦点。
- 支持 ArrowUp/ArrowDown/Enter/Escape 操作和鼠标 hover 更新游标。
- “切换模式”命令调用后端 `ChatService.SetMode()`，成功后再更新 `chatStore.setMode()`，避免前端状态与后端模式脱节。
- 会话命令来自最多 20 个最近会话，前 9 个显示 `Cmd/Ctrl+数字` 快捷提示。

## 5. 变更影响面
- 顶部 Header 的命令入口和 `Cmd/Ctrl+K` 均打开该组件。
- 新增工作区打开命令，需要父组件处理 `open-workspace`。
