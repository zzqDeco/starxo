# WorkspaceDrawer.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/files/WorkspaceDrawer.vue
- 文档文件: doc/src/frontend/src/components/files/WorkspaceDrawer.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/files

## 2. 核心职责
- 工作区右侧抽屉容器，承载 `WorkspacePanel`。
- 提供抽屉开关动画、遮罩关闭、ESC 关闭和宽度拖拽。

## 3. 输入与输出
- 输入来源: `Props.show`
- 输出结果: `update:show`（双向绑定抽屉开关）

## 4. 关键实现细节
- 宽度控制:
  - `drawerWidth` 默认 980
  - `SplitHandle` 持久化 key: `starxo-workspace-drawer-width`
  - `minDrawerWidth` / `maxDrawerWidth` 根据窗口宽度约束
- 交互行为:
  - 点击遮罩关闭
  - `Esc` 关闭
  - 抽屉开关通过 `v-model:show`
- 视觉层级:
  - 抽屉层采用较高 z-index，避免被聊天浮层覆盖

## 5. 依赖关系
- 内部依赖:
  - `@/components/layout/SplitHandle.vue`
  - `./WorkspacePanel.vue`
- 外部依赖:
  - `vue`, `@vueuse/core`, `naive-ui`, `@vicons/ionicons5`, `vue-i18n`

## 6. 变更影响面
- 影响工作区入口交互方式（从常驻区域迁移为按需抽屉）。

## 7. 维护建议
- 如需持久化“默认是否打开”，可在父组件存储 `show` 状态。
- 调整抽屉动效时，注意同步 backdrop 与 panel 的过渡时序。
