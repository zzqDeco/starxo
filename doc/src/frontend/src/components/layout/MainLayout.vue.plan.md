# MainLayout.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/MainLayout.vue
- 文档文件: doc/src/frontend/src/components/layout/MainLayout.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout (布局模块)

## 2. 核心职责
- 应用主布局组件，定义三栏可拖拽布局：左侧边栏 (Sidebar) + 中央内容 (Header + ChatPanel) + 右侧面板 (Terminal/Files/Containers)。
- 管理右侧面板的显隐和 Tab 切换、设置面板模态的显隐、面板宽度的拖拽调整和持久化。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 子组件事件 (toggle-settings, toggle-right-panel)、SplitHandle 拖拽事件 (update:size)、@vueuse/core useWindowSize
- 输出结果: 渲染完整应用布局

## 4. 关键实现细节
- **内部状态**:
  - `showSettings: boolean` — 设置面板可见性
  - `rightPanelTab: 'terminal' | 'files' | 'containers'` — 右侧面板当前 Tab
  - `showRightPanel: boolean` — 右侧面板可见性
  - `leftWidth: number` — 左侧栏宽度（默认 240，范围 180-360，localStorage 持久化）
  - `rightWidth: number` — 右侧面板宽度（默认 380，范围 280-600，localStorage 持久化）
  - `tabIndex: computed` — 当前 Tab 索引（用于滑动指示器定位）
  - `effectiveLeftWidth: computed` — 窗口 < 900px 时自动缩窄到 180px
- **布局结构** (纯 CSS flexbox + SplitHandle 组件):
  - `div.main-layout` (flex row) — 最外层
    - `div.left-panel` (动态宽度) — 左侧边栏，包含 Sidebar
    - `SplitHandle` (horizontal, min=180, max=360, default=240) — 左侧拖拽分割条
    - `div.center-section` (flex: 1, flex column) — 中央区域
      - `Header` — 顶部导航栏
      - `div.content-area` (flex row) — 内容区
        - `div.chat-area` (flex: 1) — 聊天内容，包含 ChatPanel
        - `SplitHandle` (v-if showRightPanel, horizontal, min=280, max=600, default=380, reverse) — 右侧拖拽分割条
        - `div.right-panel` (v-if showRightPanel, 动态宽度) — 右侧面板
          - Tab 切换按钮 + 滑动指示器 (.tab-indicator, transition: transform 250ms ease-out)
          - `TerminalPanel` (v-show) / `FileExplorer` (v-show) / `ContainerPanel` (v-show)
  - `SettingsPanel` (v-model:show) — 设置模态
- **Tab 滑动指示器**: 底部 2px 高亮线通过 `transform: translateX(tabIndex * 100%)` 实现平滑切换动画

## 5. 依赖关系
- 内部依赖: `./Header.vue`、`./Sidebar.vue`、`./SplitHandle.vue`、`@/components/chat/ChatPanel.vue`、`@/components/terminal/TerminalPanel.vue`、`@/components/files/FileExplorer.vue`、`@/components/containers/ContainerPanel.vue`、`@/components/settings/SettingsPanel.vue`
- 外部依赖: `vue` (ref, computed)、`@vueuse/core` (useWindowSize)、`vue-i18n` (useI18n)

## 6. 变更影响面
- 布局结构从 Naive UI NLayout 迁移到纯 flexbox，不再依赖 NLayout/NLayoutSider/NLayoutContent
- 面板宽度可拖拽调整，通过 localStorage 持久化
- 窗口 < 900px 时左侧栏自动缩窄
- 新增面板或 Tab 需在 rightPanelTab 类型、tabIndex 计算和模板中添加

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 右侧面板使用 v-show 而非 v-if，确保 Terminal 和 FileExplorer 的状态在 Tab 切换时保持。
- SplitHandle 的 storageKey 用于 localStorage 持久化，修改 key 名会导致用户已保存的宽度丢失。
- 右侧 SplitHandle 使用 `reverse: true`，因为拖拽方向与面板增长方向相反。
