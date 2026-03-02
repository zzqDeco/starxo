# TerminalPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/terminal/TerminalPanel.vue
- 文档文件: doc/src/frontend/src/components/terminal/TerminalPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/terminal (终端模块)

## 2. 核心职责
- 终端输出面板组件，使用 xterm.js 显示 Agent 的命令执行输出和容器状态信息。
- 底部状态栏显示 SSH 连接状态、活跃容器名称和输出行数统计。
- 提供 xterm.js 不可用时的纯 HTML 回退渲染。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Wails 事件 (`terminal:output`, `container:ready`, `container:progress`)、connectionStore (sshConnected)、containerStore (activeContainerID)
- 输出结果: 渲染终端输出 UI + 底部状态栏

## 4. 关键实现细节
- **Wails 事件监听** (通过 useWailsEvent composable):
  - `terminal:output` — 接收 stdout/stderr/exitCode，写入终端，递增行数计数
  - `container:ready` — 显示带时间戳的容器连接成功消息
  - `container:progress` — 显示带时间戳的容器创建进度 `[HH:MM:SS] [%] step`
- **xterm.js 初始化**: 动态导入、深色主题配置、FitAddon 自适应
- **ResizeObserver**: 自动适配容器尺寸
- **回退模式**: xterm 不可用时使用 div 列表渲染；stderr 样式增加红色左边框 + 浅红背景
- **状态栏** (.terminal-status-bar):
  - 左侧: SSH 连接状态点 (绿色 connected / 灰色 disconnected) + 活跃容器名称 (Cube 图标)
  - 右侧: 输出行数统计
- **行数计数**: `lineCount` ref 跟踪终端输出总行数，clearTerminal 时重置

## 5. 依赖关系
- 内部依赖: `@/composables/useWailsEvent`、`@/stores/connectionStore`、`@/stores/containerStore`
- 外部依赖: vue、naive-ui、@vicons/ionicons5 (TrashOutline, Cube)、@xterm/xterm (动态导入)、@xterm/addon-fit (动态导入)、vue-i18n

## 6. 变更影响面
- 新增 connectionStore 和 containerStore 依赖用于状态栏显示
- 被 MainLayout 右侧面板包含

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 事件名与后端 SandboxService 发射的事件保持一致（`container:*` 命名空间）。
- xterm.js 使用动态导入，加载失败时自动回退。
- 状态栏依赖 connectionStore/containerStore，store 接口变更时需同步更新。
