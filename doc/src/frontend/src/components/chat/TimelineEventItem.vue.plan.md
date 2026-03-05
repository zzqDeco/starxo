# TimelineEventItem.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/TimelineEventItem.vue
- 文档文件: doc/src/frontend/src/components/chat/TimelineEventItem.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat

## 2. 核心职责
- 渲染单条时间线事件（消息、工具调用、中断、推理、思考等）。
- 对工具调用使用“短横条摘要 + 按需展开详情”的高密度样式。

## 3. 输入与输出
- 输入来源: `Props.event`、`Props.showAgentBadge`
- 输出结果: 事件可视化 UI

## 4. 关键实现细节
- 工具分类:
  - `file`, `edit`, `shell`, `agent`, `todo`, `notify`, `other`
- 摘要条信息:
  - `action`（动作）
  - `primary`（主信息，如路径/命令）
  - `secondary`（辅助信息，如退出码/行数）
- 结果处理:
  - toolResult 超过 500 字符默认截断，可手动展开
- todo 工具策略:
  - `write_todos/update_todo` 仅显示摘要统计，不再内嵌 `TodoBoard`
  - `hasDetails` 对 todo 分类返回 false，避免重复展开冗余内容
- 支持事件类型:
  - `message`, `tool_call`, `transfer`, `interrupt`, `info`, `reasoning`, `thinking`

## 5. 依赖关系
- 内部依赖:
  - `@/types/message` (`TurnEvent`)
  - `@/composables/useHelpers` (`useMarkdown`)
- 外部依赖:
  - `vue`, `naive-ui`, `@vicons/ionicons5`, `vue-i18n`

## 6. 变更影响面
- 时间线中 todo 工具的可视形态由“详情组件”改为“摘要条”，减少视觉堆叠。

## 7. 维护建议
- 新增工具名时在 `toolInfo` 中显式分类，避免落入 `other` 丢失语义。
- 若扩展 todo 详情，建议放在独立面板而非消息时间线内。
