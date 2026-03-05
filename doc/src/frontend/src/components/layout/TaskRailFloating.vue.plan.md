# TaskRailFloating.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/TaskRailFloating.vue
- 文档文件: doc/src/frontend/src/components/layout/TaskRailFloating.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout

## 2. 核心职责
- 输入区上方的任务浮层组件，提供任务执行的紧凑摘要与可展开列表。

## 3. 输入与输出
- 输入来源: `chatStore.latestTodos` / `chatStore.planSteps`
- 输出结果: 渲染任务摘要条和折叠任务列表

## 4. 关键实现细节
- 信息密度:
  - 头部展示：标题、完成比、当前任务、运行数/失败数
  - 下方展示：进度条
  - 展开后展示前 8 条任务
- 数据逻辑:
  - todo 优先，plan 回退
  - `currentTask` 优先显示进行中任务
- 状态视觉:
  - running/done/failed/pending 图标和颜色区分
  - running 状态图标旋转动画

## 5. 依赖关系
- 内部依赖: `chatStore`
- 外部依赖: `vue`, `naive-ui`, `@vicons/ionicons5`, `vue-i18n`

## 6. 变更影响面
- 替代聊天区内持久 TodoBoard，减少消息区视觉堆叠。
- 与 InputArea 一起构成底部统一交互区。

## 7. 维护建议
- 保持浮层高度上限，避免在大量任务时挤压输入区域。
- 如需查看更多任务，建议引导到完整 `TaskRail` 视图。
