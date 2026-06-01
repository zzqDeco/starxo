# InputArea.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/InputArea.vue
- 文档文件: doc/src/frontend/src/components/chat/InputArea.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat

## 2. 核心职责
- 工作台 composer 组件，提供模式切换、附件、文本输入、发送与停止操作。

## 3. 输入与输出
- 输入来源: `Props.isStreaming`, `Props.agentMode`, `Props.modeSwitching`
- 输出结果: `send(content, filePath?)`、`stop()`、`switch-mode(mode)`

## 4. 关键实现细节
- Props:
  - `isStreaming: boolean`
  - `agentMode: 'default' | 'plan'`
  - `modeSwitching: boolean`
- Emits:
  - `send(content: string, filePath?: string)`
  - `stop()`
  - `switch-mode(mode: 'default'|'plan')`
- 交互逻辑:
  - Enter 发送，Shift+Enter 换行
  - 发送后清空 `inputText` 与 `attachedFile`
  - 附件由 `window.runtime.OpenFileDialog()` 选择
  - 通过 `useNativeTextInputSync` 同步原生 textarea，保证键盘、粘贴和辅助技术 value 更新都会进入 Vue state
- 样式结构:
  - `composer-meta`: 模式 segmented control + Shift+Enter 提示
  - 附件条（可移除）
  - `input-shell` 单层容器：附件按钮 + textarea + 发送/停止按钮
  - textarea autosize 保持 1~4 行，超出后内部滚动
  - 移除底部 `Shift+Enter` 提示行，整体更紧凑

## 5. 依赖关系
- 外部依赖:
  - `vue` (`ref`, `computed`)
  - `naive-ui` (`NButton`, `NIcon`, `NTooltip`)
  - `@vicons/ionicons5` (`Send`, `Attach`, `StopCircle`)
  - `vue-i18n`

## 6. 变更影响面
- 模式切换从 ChatPanel 顶部工具条下沉到 composer，输入区成为主要操作面板。
- 输入区视觉密度降低，和 ChatPanel 底部栅格统一。
- macOS 下模式切换按原生 segmented control 处理，active 状态使用中性 raised segment，不使用大面积蓝色 primary button。

## 7. 维护建议
- 若后续引入多模态附件，优先扩展 `attachedFile` 结构为对象数组。
- Wails runtime 类型建议后续补充声明，减少 `@ts-ignore` 依赖。
- 在 scoped CSS 内覆盖 Naive UI 子节点时，macOS 平台选择器应使用 `:global(:root[data-platform="macos"] .mode-buttons .n-button)`，不要在 `:global(...)` 内嵌 `:deep(...)`。
- composer 发送按钮的 enabled 状态必须来源于已同步的 `inputText`，不能直接读取 DOM 或依赖未触发 input event 的值。
