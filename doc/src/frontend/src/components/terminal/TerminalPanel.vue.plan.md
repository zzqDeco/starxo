# TerminalPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/terminal/TerminalPanel.vue
- 文档文件: doc/src/frontend/src/components/terminal/TerminalPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/terminal (终端模块)

## 2. 核心职责
- 终端输出面板组件，使用 xterm.js 显示 Agent 的命令执行输出和沙盒状态信息。
- 提供 xterm.js 不可用时的纯 HTML 回退渲染。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Wails 事件 (terminal:output, sandbox:ready, sandbox:progress)
- 输出结果: 渲染终端输出 UI

## 4. 关键实现细节
- **Wails 事件监听** (通过 useWailsEvent composable):
  - `terminal:output` — 接收 stdout/stderr/exitCode，写入终端
  - `sandbox:ready` — 显示连接成功消息
  - `sandbox:progress` — 显示初始化进度
- **xterm.js 初始化** (`initXterm`):
  - 动态导入 `@xterm/xterm` 和 `@xterm/addon-fit`
  - 配置深色主题色板（与全局 CSS 变量一致）
  - 配置字体 (JetBrains Mono)、字号 (12px)、回滚行数 (5000)
  - 使用 FitAddon 自适应容器大小
- **ResizeObserver**: 监听容器尺寸变化，自动调用 fitAddon.fit()
- **回退模式**: xterm 不可用时使用 div 列表渲染 stdout/stderr/info 行
- **清除功能**: xterm.clear() 或清空 lines 数组

## 5. 依赖关系
- 内部依赖: `@/composables/useWailsEvent`
- 外部依赖: `vue` (ref, onMounted, onUnmounted, nextTick)、`naive-ui` (NButton, NIcon)、`@vicons/ionicons5` (TrashOutline)、`@xterm/xterm` (动态导入)、`@xterm/addon-fit` (动态导入)、`vue-i18n` (useI18n)

## 6. 变更影响面
- xterm.js 版本升级可能影响 API 兼容性
- 主题色修改需同步 Terminal 构造参数的 theme 对象
- 被 MainLayout 右侧面板包含

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- xterm.js 使用动态导入 (await import)，加载失败时会自动回退到简单终端。
- 需注意 onUnmounted 中正确 dispose termInstance 以避免内存泄漏。
- ResizeObserver 在 onUnmounted 中 disconnect，确保清理。
