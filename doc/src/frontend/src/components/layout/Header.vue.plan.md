# Header.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/Header.vue
- 文档文件: doc/src/frontend/src/components/layout/Header.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout (布局模块)

## 2. 核心职责
- 应用顶部导航栏组件，显示应用标题、连接状态和操作按钮（右侧面板切换、语言切换、设置）。
- 同时作为 Wails 窗口拖拽区域。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Props (rightPanelVisible: boolean)
- 输出结果: Emits (toggle-settings, toggle-right-panel)；渲染导航栏 UI

## 4. 关键实现细节
- **Props 定义**: `rightPanelVisible: boolean` — 右侧面板当前可见状态
- **Emits 定义**:
  - `toggle-settings` — 切换设置面板
  - `toggle-right-panel` — 切换右侧面板
- **语言切换**: `toggleLocale()` 在 en/zh 间切换，持久化到 localStorage('locale')
- **模板结构**:
  - header-left: 应用图标 + 标题 (Eino Agent)
  - header-center: ConnectionStatus 组件
  - header-right: 右侧面板切换按钮 + 语言切换按钮 + 设置按钮
- **Wails 拖拽**: header 元素带 `wails-drag` CSS 类，启用窗口拖拽

## 5. 依赖关系
- 内部依赖: `@/components/status/ConnectionStatus.vue`
- 外部依赖: `naive-ui` (NButton, NTooltip)、`@vicons/ionicons5` (Settings, Terminal, ChevronForward, ChevronBack)、`vue-i18n` (useI18n)

## 6. 变更影响面
- Emits 修改需同步 MainLayout 的事件处理
- 语言切换逻辑修改影响全局国际化
- 头部高度 (52px) 影响 MainLayout 中 content-area 的 top 偏移

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 头部高度 52px 在 MainLayout 中通过 `top: 52px` 硬编码引用，修改时需同步。
- 语言切换使用 localStorage 持久化，与 locales/index.ts 中的初始化逻辑保持一致。
