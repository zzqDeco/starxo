# MainLayout.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/MainLayout.vue
- 文档文件: doc/src/frontend/src/components/layout/MainLayout.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout

## 2. 核心职责
- 应用主布局容器，组织为“左侧会话栏 + 中央执行画布 + 右侧运行时入口”。
- 负责工作区抽屉、命令面板、面板宽度拖拽持久化和设置面板显隐。
- 监听工作区路径桥接事件，确保时间线里的文件路径能打开工作区抽屉。

## 3. 输入与输出
- 输入来源: Header 事件、SplitHandle 拖拽事件、窗口尺寸 (`useWindowSize`)
- 输出结果: 渲染主界面骨架与全局设置面板

## 4. 关键实现细节
- 内部状态:
  - `showSettings` — 设置面板开关
  - `showWorkspaceDrawer` — 工作区抽屉开关
  - `showPalette` — 命令面板开关
  - `leftWidth` — 左侧栏宽度（默认 240）
  - `containerDockWidth` — 右侧容器区宽度（默认 360）
- 响应式计算:
  - `effectiveLeftWidth`: 窗口小于 900 时限制左侧宽度
  - `effectiveDockWidth`: 窗口小于 1280 时限制容器区宽度
- 布局结构:
  - `Sidebar` + 左分割条
  - 中央 `Header + ChatPanel`
  - `WorkspaceDrawer` 以覆盖层方式挂在聊天区内
  - 右侧 `ContainerDock` 常驻
  - SettingsPanel / CommandPalette / WorkspaceDrawer / ContainerDock 使用 async component 降低首包
  - `onWorkspaceOpenPath()` 打开 WorkspaceDrawer；路径选择由 WorkspacePanel 消费 pending path
- 拖拽持久化 key:
  - `starxo-left-panel-width`
  - `starxo-container-dock-width`

## 5. 依赖关系
- 内部依赖:
  - `Header.vue`, `Sidebar.vue`, `SplitHandle.vue`
  - `ChatPanel.vue`, async `WorkspaceDrawer.vue`, async `ContainerDock.vue`, async `SettingsPanel.vue`, async `CommandPalette.vue`
  - `@/composables/useWorkspaceBridge`
- 外部依赖: `vue`、`@vueuse/core`

## 6. 变更影响面
- 右侧旧 Tab 面板（Terminal/Files/Containers）已移除。
- 工作区从常驻区域改为按需抽屉，容器控制改为常驻 Dock。
- Header 新增 `open-command-palette` 事件；CommandPalette 新增 `open-workspace` 事件。

## 7. 维护建议
- 若新增右侧常驻区，优先通过 `SplitHandle` 接入并复用宽度持久化。
- 抽屉层级与聊天浮层（如回到底部按钮）发生冲突时，优先提高抽屉 z-index。
