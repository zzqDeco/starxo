# TaskRail.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/TaskRail.vue
- 文档文件: doc/src/frontend/src/components/layout/TaskRail.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout

## 2. 核心职责
- 任务轨道完整视图，展示任务列表、进度条、状态统计与筛选。

## 3. 输入与输出
- 输入来源: `chatStore.latestTodos` / `chatStore.planSteps`
- 输出结果: 渲染任务卡片列表与统计信息

## 4. 关键实现细节
- 数据优先级:
  - 优先使用 `latestTodos`
  - 无 todo 时回退 `planSteps`
- 筛选:
  - `all`, `running`, `failed`, `done`
- 统计:
  - 进行中/失败/完成数量
  - 完成百分比进度条
- 状态展示:
  - `pending`, `in_progress`, `done`, `failed`, `skipped`
  - 支持显示任务依赖

## 5. 依赖关系
- 内部依赖: `chatStore`
- 外部依赖: `vue`, `naive-ui`, `@vicons/ionicons5`, `vue-i18n`

## 6. 变更影响面
- 当前主布局已改为 `TaskRailFloating` 常驻，`TaskRail` 作为完整视图组件可复用。

## 7. 维护建议
- 需保持 todo 与 plan 两种数据源的状态映射一致。
- 后续若移除该组件，应同步清理 i18n 中无用键。
