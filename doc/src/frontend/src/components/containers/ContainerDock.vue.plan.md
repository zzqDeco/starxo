# ContainerDock.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/containers/ContainerDock.vue
- 文档文件: doc/src/frontend/src/components/containers/ContainerDock.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/containers

## 2. 核心职责
- 右侧常驻容器 Dock 的壳组件。
- 负责把 `ContainerPanel` 以固定区域形式挂载到主布局。

## 3. 输入与输出
- 输入来源: 无 Props
- 输出结果: 渲染 `ContainerPanel`

## 4. 关键实现细节
- 组件逻辑极简，主要用于布局职责拆分：
  - 模板仅包含 `<ContainerPanel />`
  - 样式保证高度填满和溢出裁剪
- 主布局通过 `SplitHandle` 控制 Dock 宽度，Dock 本身不处理拖拽状态。

## 5. 依赖关系
- 内部依赖: `./ContainerPanel.vue`
- 外部依赖: `vue`（SFC setup）

## 6. 变更影响面
- 作为 MainLayout 右侧常驻区容器，影响容器操作入口可达性。

## 7. 维护建议
- 如需新增 Dock 级标题/筛选，仅在该壳层扩展，避免污染 `ContainerPanel` 业务组件。
