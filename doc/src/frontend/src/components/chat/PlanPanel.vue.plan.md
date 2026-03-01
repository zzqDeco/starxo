# PlanPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/PlanPanel.vue
- 文档文件: doc/src/frontend/src/components/chat/PlanPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat (聊天模块)

## 2. 核心职责
- 执行计划面板组件，以可折叠列表形式展示 Agent 的计划步骤及其执行状态。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: chatStore.planSteps 状态
- 输出结果: 渲染计划步骤列表 UI

## 4. 关键实现细节
- **Pinia Store 交互**: chatStore — planSteps
- **计算属性**:
  - `steps` — planSteps 引用
  - `hasSteps` — 是否有步骤（控制组件可见性）
  - `completedCount` — 已完成 (done) 步骤数
  - `totalCount` — 总步骤数
  - `progressPercent` — 完成百分比
- **状态图标映射** (`statusIcon`): done(checkmark), doing(play), failed(cross), skipped(dash), todo(circle)
- **模板结构**: 条件渲染 (hasSteps) → NCollapse (默认展开) → 步骤列表，每步显示状态图标 + 描述 + 可选执行结果
- **样式**: 每种状态有对应的颜色和动画（doing 状态有 pulse 动画）

## 5. 依赖关系
- 内部依赖: `@/stores/chatStore`
- 外部依赖: `vue` (computed)、`naive-ui` (NCollapse, NCollapseItem)、`vue-i18n` (useI18n)

## 6. 变更影响面
- 步骤状态类型扩展需同步 PlanStepDTO 类型和 statusIcon/statusClass 映射
- 样式修改影响计划面板的视觉呈现

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增 PlanStepDTO.status 值时需同步更新 statusIcon 和对应 CSS 类。
