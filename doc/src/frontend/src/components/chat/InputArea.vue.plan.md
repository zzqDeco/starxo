# InputArea.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/InputArea.vue
- 文档文件: doc/src/frontend/src/components/chat/InputArea.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat (聊天模块)

## 2. 核心职责
- 聊天输入区域组件，提供文本输入、文件附加和发送/停止操作。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Props (isStreaming)
- 输出结果: Emits (send, stop)

## 4. 关键实现细节
- **Props 定义**:
  - `isStreaming: boolean` — 是否正在流式生成
- **Emits 定义**:
  - `send(content: string, filePath?: string)` — 发送消息
  - `stop()` — 停止生成
- **内部状态**:
  - `inputText` — 输入文本
  - `attachedFile` — 附加文件路径
- **关键逻辑**:
  - `canSend` — 输入非空且未在生成时可发送
  - Enter 键发送，Shift+Enter 换行
  - 文件附加通过 Wails 的 `window.runtime.OpenFileDialog` 实现
  - 发送后清空输入和附件
- **模板结构**: 附件指示条 → 输入行（附件按钮 + NInput textarea + 发送/停止按钮） → Shift+Enter 提示

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `vue` (ref, computed)、`naive-ui` (NInput, NButton, NIcon, NTooltip)、`@vicons/ionicons5` (Send, Attach, StopCircle)、`vue-i18n` (useI18n)

## 6. 变更影响面
- Emits 签名修改需同步 ChatPanel 的事件处理
- 文件附加功能依赖 Wails runtime dialog

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- textarea autosize 范围 (1-6行) 可根据需求调整。
- 文件附加使用 `@ts-ignore` 注释，后续可考虑声明 Wails runtime 类型。
