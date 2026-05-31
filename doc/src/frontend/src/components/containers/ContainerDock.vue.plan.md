# ContainerDock.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/containers/ContainerDock.vue
- 文档文件: doc/src/frontend/src/components/containers/ContainerDock.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/containers

## 2. 核心职责
- 右侧 Runtime Inspector 壳组件。
- 负责展示 runtime/sandbox 摘要并承载 `ContainerPanel` 生命周期操作；终端已提升为 MainLayout 的独立 inspector mode。

## 3. 输入与输出
- 输入来源: 无 Props；读取 `containerStore.activeContainerID` 和容器列表
- 输出结果: 渲染 `ContainerPanel`

## 4. 关键实现细节
- 顶部显示 Runtime 标题和当前激活容器摘要。
- Runtime/Terminal 不再使用内部 tablist；MainLayout 通过四态 `SxInspectorSegmented` 控制 `runtime | workspace | terminal | tasks`。
- 样式保证高度填满、tabpanel 独立滚动和右侧 Dock 内部信息密度。
- 主布局通过 `SplitHandle` 控制 Dock 宽度，Dock 本身不处理拖拽状态。

## 5. 依赖关系
- 内部依赖: `./ContainerPanel.vue`, `containerStore`
- 外部依赖: `vue`, `naive-ui`, `@vicons/ionicons5`, `vue-i18n`

## 6. 变更影响面
- 作为 MainLayout `runtime` inspector mode 的内容，影响沙箱生命周期入口可达性。

## 7. 维护建议
- 如需新增 Dock 级标题/筛选，仅在该壳层扩展，避免污染 `ContainerPanel` 业务组件。
