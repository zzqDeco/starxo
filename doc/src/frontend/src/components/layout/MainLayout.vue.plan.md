# MainLayout.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/MainLayout.vue
- 文档文件: doc/src/frontend/src/components/layout/MainLayout.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout

## 2. 核心职责
- 应用主布局容器，组织为“左侧会话栏 + 中央执行画布 + 右侧 inspector”。
- 负责 workspace/runtime inspector 互斥、工作区抽屉、运行任务面板、命令面板、面板宽度拖拽持久化和设置面板显隐。
- 监听工作区路径桥接事件，确保时间线里的文件路径能打开工作区抽屉。

## 3. 输入与输出
- 输入来源: Header 事件、SplitHandle 拖拽事件、窗口尺寸 (`useWindowSize`)
- 输出结果: 渲染主界面骨架与全局设置面板

## 4. 关键实现细节
- 内部状态:
  - `showSettings` — 设置面板开关
  - `showWorkspaceDrawer` — 工作区抽屉开关
  - `showRuntimeTasks` — Runtime Tasks 面板开关
  - `showPalette` — 命令面板开关
  - `leftWidth` — 左侧栏宽度（默认 240）
  - `runtimeDockWidth` — runtime inspector 宽度（默认 360）
  - `workspaceInspectorWidth` — workspace inspector 宽度（默认 620）
  - `inspectorMode` — 桌面端右侧 inspector 模式，`runtime` / `workspace` / `null`
- 响应式计算:
  - `effectiveLeftWidth`: 窗口小于 900 时限制左侧宽度
  - `effectiveDockWidth`: 按当前 inspector 模式和窗口宽度约束右侧宽度
  - `isSidebarCompact`: workspace inspector 打开或宽度不足时把左侧栏压缩为 compact rail
- 布局结构:
  - `Sidebar` + 左分割条
  - 中央 `Header + ChatPanel`
  - 桌面端右侧 inspector 在 `WorkspacePanel` 与 `ContainerDock` 间互斥切换
  - 窄屏下 `WorkspaceDrawer` 与 responsive `ContainerDock` overlay 互斥
  - SettingsPanel / CommandPalette / WorkspaceDrawer / ContainerDock / RuntimeTasksPanel 使用 async component 降低首包
  - `onWorkspaceOpenPath()` 打开 WorkspaceDrawer；路径选择由 WorkspacePanel 消费 pending path
- 拖拽持久化 key:
  - `starxo-left-panel-width`
  - `starxo-runtime-inspector-width`
  - `starxo-workspace-inspector-width`

## 5. 依赖关系
- 内部依赖:
  - `Header.vue`, `Sidebar.vue`, `SplitHandle.vue`
  - `ChatPanel.vue`, async `WorkspaceDrawer.vue`, async `ContainerDock.vue`, async `SettingsPanel.vue`, async `CommandPalette.vue`, async `RuntimeTasksPanel.vue`
  - `@/composables/useWorkspaceBridge`
- 外部依赖: `vue`、`@vueuse/core`

## 6. 变更影响面
- 右侧旧 Tab 面板（Terminal/Files/Containers）已移除。
- 工作区从桌面覆盖抽屉改为 inspector，容器控制和 workspace 不再同时挤占右侧区域。
- Header 新增 `toggle-runtime-tasks` 事件；CommandPalette 新增 `open-runtime-tasks` 事件。

## 7. 维护建议
- 若新增右侧常驻区，优先通过 `SplitHandle` 接入并复用宽度持久化。
- 抽屉层级与聊天浮层（如回到底部按钮）发生冲突时，优先提高抽屉 z-index。
