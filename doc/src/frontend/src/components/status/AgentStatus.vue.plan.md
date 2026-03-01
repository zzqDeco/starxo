# AgentStatus.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/status/AgentStatus.vue
- 文档文件: doc/src/frontend/src/components/status/AgentStatus.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/status (状态指示模块)

## 2. 核心职责
- Agent 工作状态指示器组件，在 Agent 正在生成时显示思考动画和当前 Agent 名称。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Props (agent: string)
- 输出结果: 渲染 Agent 工作状态条 UI

## 4. 关键实现细节
- **Props 定义**: `agent: string` — 当前活跃 Agent 名称
- **Agent 颜色映射** (`agentColor` computed):
  - 包含 "coder"/"code" → cyan
  - 包含 "plan" → blue
  - 包含 "review" → amber
  - 包含 "test" → emerald
  - 默认 → cyan
- **模板结构** (v-if="agent"):
  - 思考动画: 3 个彩色圆点，使用 pulse 动画和交错延迟 (0s/0.2s/0.4s)
  - Agent 标签: 使用 agent 名称，带颜色
  - 状态文本: "正在工作..."

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `vue` (computed)、`vue-i18n` (useI18n)

## 6. 变更影响面
- 被 ChatPanel 在 isStreaming 时条件渲染
- Agent 颜色映射修改影响不同 Agent 的视觉区分

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- Agent 颜色使用 CSS 变量引用，与全局设计令牌保持一致。
- 新增 Agent 类型时可在 agentColor computed 中添加颜色映射分支。
