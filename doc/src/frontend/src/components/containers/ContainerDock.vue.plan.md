# ContainerDock.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/containers/ContainerDock.vue
- 文档文件: doc/src/frontend/src/components/containers/ContainerDock.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/containers

## 2. 核心职责
- 右侧 Runtime Inspector 壳组件。
- 负责在同一固定区域内组织容器管理和终端输出。

## 3. 输入与输出
- 输入来源: 无 Props；读取 `containerStore.activeContainerID` 和容器列表
- 输出结果: 渲染 `ContainerPanel` / `TerminalPanel` tab

## 4. 关键实现细节
- 顶部显示 Runtime 标题和当前激活容器摘要。
- tablist:
  - `containers`: 当前会话容器、其他容器和生命周期操作（复用 ContainerPanel）
  - `terminal`: xterm 终端输出（复用 TerminalPanel）
- 样式保证高度填满、tabpanel 独立滚动和右侧 Dock 内部信息密度。
- 主布局通过 `SplitHandle` 控制 Dock 宽度，Dock 本身不处理拖拽状态。

## 5. 依赖关系
- 内部依赖: `./ContainerPanel.vue`, `@/components/terminal/TerminalPanel.vue`, `containerStore`
- 外部依赖: `vue`, `naive-ui`, `@vicons/ionicons5`, `vue-i18n`

## 6. 变更影响面
- 作为 MainLayout 右侧常驻区容器，影响容器操作和终端输出入口可达性。

## 7. 维护建议
- 如需新增 Dock 级标题/筛选，仅在该壳层扩展，避免污染 `ContainerPanel` 业务组件。
