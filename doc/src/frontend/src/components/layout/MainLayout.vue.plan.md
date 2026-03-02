# MainLayout.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/MainLayout.vue
- 文档文件: doc/src/frontend/src/components/layout/MainLayout.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout (布局模块)

## 2. 核心职责
- 应用主布局组件，定义三栏布局结构：左侧边栏 (Sidebar) + 中央内容 (Header + ChatPanel) + 右侧面板 (Terminal/Files/Containers)。
- 管理右侧面板的显隐和 Tab 切换，以及设置面板模态的显隐。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 子组件事件 (toggle-settings, toggle-right-panel)
- 输出结果: 渲染完整应用布局

## 4. 关键实现细节
- **内部状态**:
  - `showSettings: boolean` — 设置面板可见性
  - `rightPanelTab: 'terminal' | 'files' | 'containers'` — 右侧面板当前 Tab
  - `showRightPanel: boolean` — 右侧面板可见性
- **布局结构** (使用 Naive UI NLayout 组件):
  - `NLayout` (has-sider, position=absolute) — 最外层
    - `NLayoutSider` (width=240) — 左侧边栏，包含 Sidebar
    - `NLayout` (center) — 中央区域
      - `Header` — 顶部导航栏
      - `NLayout` (has-sider, content-area) — 内容区
        - `NLayoutContent` — 聊天内容，包含 ChatPanel
        - `NLayoutSider` (width=380, 条件渲染) — 右侧面板
          - Tab 切换按钮 (Terminal / Files / Containers)
          - `TerminalPanel` (v-show) / `FileExplorer` (v-show) / `ContainerPanel` (v-show)
  - `SettingsPanel` (v-model:show) — 设置模态

## 5. 依赖关系
- 内部依赖: `./Header.vue`、`./Sidebar.vue`、`@/components/chat/ChatPanel.vue`、`@/components/terminal/TerminalPanel.vue`、`@/components/files/FileExplorer.vue`、`@/components/containers/ContainerPanel.vue`、`@/components/settings/SettingsPanel.vue`
- 外部依赖: `vue` (ref)、`naive-ui` (NLayout, NLayoutSider, NLayoutContent)、`vue-i18n` (useI18n)

## 6. 变更影响面
- 布局结构修改影响整体 UI 呈现
- 侧边栏宽度修改需考虑响应式适配
- 新增面板或 Tab 需在 rightPanelTab 类型和模板中添加

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 右侧面板使用 v-show 而非 v-if，确保 Terminal 和 FileExplorer 的状态在 Tab 切换时保持。
- 左侧栏宽度 240px 和右侧栏宽度 380px 为固定值，后续可考虑支持拖拽调整。
