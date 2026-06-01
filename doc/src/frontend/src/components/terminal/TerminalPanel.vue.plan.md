# TerminalPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/terminal/TerminalPanel.vue
- 文档文件: doc/src/frontend/src/components/terminal/TerminalPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/terminal (终端模块)

## 2. 核心职责
- 终端面板组件，使用 xterm.js 显示 Agent/用户命令输出和容器状态信息。
- 提供当前 active sandbox workspace 的非交互式命令输入，不是持久 PTY。
- 底部状态栏显示 SSH 连接状态、活跃容器名称和输出行数统计。
- 提供 xterm.js 不可用时的纯 HTML 回退渲染。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Wails 事件 (`terminal:output`, `container:ready`, `container:progress`)、SandboxService `RunTerminalCommand`、connectionStore (sshConnected)、containerStore (activeContainerID)
- 输出结果: 渲染终端输出 UI、命令输入条、底部状态栏

## 4. 关键实现细节
- **Wails 事件监听** (通过 useWailsEvent composable):
  - `terminal:output` — 接收 stdout/stderr/exitCode，写入终端，递增行数计数
  - `container:ready` — 显示带时间戳的容器连接成功消息
  - `container:progress` — 显示带时间戳的容器创建进度 `[HH:MM:SS] [%] step`
- **xterm.js 初始化**: 动态导入、深色主题配置、FitAddon 自适应
- 初始 banner 使用短句 “Starxo terminal / Run commands in the active sandbox workspace.”，避免把终端面板表现成营销式或版本号式启动页。
- **ResizeObserver**: 自动适配容器尺寸
- **回退模式**: xterm 不可用时使用 `SxTerminalRow` 列表渲染；stderr 使用 `--sx-danger-*` 语义 token。
- **状态栏** (.terminal-status-bar):
  - 左侧: SSH 连接状态点 (绿色 connected / 灰色 disconnected) + 活跃容器名称 (Cube 图标)
  - 右侧: 输出行数统计
- **命令输入条**:
  - SSH 未连接或没有 active sandbox 时禁用
  - 提交后调用 `RunTerminalCommand`，命令输出仍通过 `terminal:output` 写入面板
  - 使用 `useNativeTextInputSync` 主动聚焦并同步 Naive UI 内层 input，避免 xterm 输出区抢焦点或辅助技术设置 value 后按钮仍 disabled
- **行数计数**: `lineCount` ref 跟踪终端输出总行数，clearTerminal 时重置

## 5. 依赖关系
- 内部依赖: `@/composables/useWailsEvent`、`@/stores/connectionStore`、`@/stores/containerStore`
- 外部依赖: vue、naive-ui、@vicons/ionicons5 (TrashOutline, Cube, PaperPlaneOutline)、@xterm/xterm (动态导入)、@xterm/addon-fit (动态导入)、vue-i18n
- UI 依赖: `@/components/ui/SxTerminalRow.vue`

## 6. 变更影响面
- 新增 connectionStore 和 containerStore 依赖用于状态栏显示
- 作为 MainLayout `terminal` inspector mode 的一等面板，不再隐藏在 runtime tab 内。

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 事件名与后端 SandboxService 发射的事件保持一致（`container:*` 命名空间）。
- xterm.js 使用动态导入，加载失败时自动回退。
- 状态栏依赖 connectionStore/containerStore，store 接口变更时需同步更新。
- 当前命令输入是一次性 shell command runner；如果后续做完整 PTY，需要新增专门的 process/session 生命周期和 resize/stdin/stdout 通道。
- 点击命令栏区域必须聚焦命令 input；xterm 输出区域不能成为提交命令的输入路径。
- 终端面板在 native UI pass 中按 inspector 工具面板处理：白色/中性输出面、轻 toolbar、底部状态栏使用系统字体，减少彩色终端装饰。
