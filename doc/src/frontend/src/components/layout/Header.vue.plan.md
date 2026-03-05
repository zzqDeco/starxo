# Header.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/Header.vue
- 文档文件: doc/src/frontend/src/components/layout/Header.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout

## 2. 核心职责
- 顶部导航栏组件，展示应用标题、连接状态、工作区抽屉开关、语言切换与设置按钮。
- 作为 Wails 可拖拽标题区域（`wails-drag`）。

## 3. 输入与输出
- 输入来源: `Props.workspaceDrawerVisible`
- 输出结果: `toggle-workspace-drawer`、`toggle-settings` 事件

## 4. 关键实现细节
- Props:
  - `workspaceDrawerVisible: boolean` — 工作区抽屉是否打开
- Emits:
  - `toggle-workspace-drawer`
  - `toggle-settings`
- 语言切换:
  - `toggleLocale()` 在 `en/zh` 间切换
  - 通过 `localStorage('locale')` 持久化
- 右上工具按钮:
  - 工作区按钮（FolderOpen 图标）按状态显示 `header.workspaceOpen / header.workspaceClose`
  - 语言按钮
  - 设置按钮

## 5. 依赖关系
- 内部依赖: `@/components/status/ConnectionStatus.vue`
- 外部依赖:
  - `naive-ui` (`NButton`, `NTooltip`)
  - `@vicons/ionicons5` (`Settings`, `FolderOpen`)
  - `vue-i18n`

## 6. 变更影响面
- 事件名从右侧面板切换语义迁移为工作区抽屉语义，父组件需同步。
- 新增 `header.workspaceOpen / header.workspaceClose` i18n 键。

## 7. 维护建议
- 修改 emits 时同步更新 `MainLayout.vue` 的监听逻辑。
- 保持 `wails-drag` 只用于可拖拽区域，避免影响按钮点击交互。
