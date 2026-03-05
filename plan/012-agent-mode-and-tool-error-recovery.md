# 012 - Plan 模式编排约束与工具错误可恢复执行

## 目标

建立明确的主代理/子代理边界，并修复工具调用参数错误导致整轮中断的问题：
- 在 Plan Mode 强制主代理只做编排与验收，具体执行由 subagent 完成
- 支持手动切换模式，并对复杂任务自动切换到 Plan Mode
- 将可恢复工具错误回传给 agent 自修复，避免 NodeRunError 直接中断

## 问题分析

### 现状
- 主代理在不同模式下工具权限边界不够严格，Plan 模式行为约束不充分
- 前端缺少显式模式切换控件，模式状态与后端同步不完整
- 工具包装层在错误时直接返回 `err`，ADK 将其视为节点失败，导致流程中断
- `str_replace_editor` 与 `read_file` 常见参数/路径错误本质可恢复，但实际被当作 fatal

### 目标
- Plan Mode 中主代理只做计划、委派、验收与任务列表维护
- Default Mode 维持灵活执行；复杂请求可自动升级到 Plan Mode
- 可恢复错误通过 `tool_result` 回传，允许 agent 调整参数重试
- 增加重复错误阈值，防止无限自旋

## 具体任务

### Phase 1: 模式化深代理构建

- [x] `internal/agent/deep_agent.go`: 新增 `DeepAgentMode` 与 `BuildDeepAgentForMode`
- [x] Default 模式保留 `extraTools`；Plan 模式不挂载 `extraTools`
- [x] Plan 模式仅保留主代理编排工具与 todo 工具

### Phase 2: Plan Mode 提示词约束

- [x] `internal/agent/prompts.go`: 新增 `DeepAgentPlanPrompt`
- [x] 明确主代理职责：规划、委派、验收、更新任务状态、最终汇报
- [x] 明确 subagent 职责：代码编辑/命令执行/文件批处理

### Phase 3: 运行时模式切换

- [x] `internal/service/chat.go`: 增加复杂度判定 `shouldAutoPlanMode`
- [x] default 模式复杂请求自动切换 plan，并广播 `agent:mode_changed`
- [x] 为 default/plan 分别构建 runner，按会话模式选择

### Phase 4: 前端模式控制与同步

- [x] `frontend/src/components/chat/ChatPanel.vue`: 增加 Default/Plan 切换按钮
- [x] 调用 `SetMode` 切换后端模式，并更新本地 store
- [x] `frontend/src/App.vue`: 启动时调用 `GetMode` 同步当前模式
- [x] `frontend/src/locales/zh.ts` + `en.ts`: 新增模式文案

### Phase 5: 工具错误可恢复机制

- [x] `internal/tools/error_policy.go`: 新增工具错误分类策略
- [x] `internal/agent/tool_wrapper.go`:
  - 可恢复错误返回 `tool_result + nil error`
  - 不可恢复错误保持原样失败
  - 同签名错误连续 3 次升级为 fatal（防循环）
- [x] `internal/service/chat.go`: run context 透传 `sessionID` 供错误计数按会话隔离
- [x] `internal/agent/prompts.go`: 增加 code_writer 出错后重读上下文再重试规则

### Phase 6: 测试与验证

- [x] `internal/tools/error_policy_test.go`: 分类规则单测
- [x] `internal/agent/tool_wrapper_test.go`: recoverable/fatal/阈值/重置行为单测
- [x] 运行 `go test ./...` 通过
- [x] 运行 `frontend npm run build` 通过

## 涉及文件

**后端修改:**
- `internal/agent/deep_agent.go`
- `internal/agent/prompts.go`
- `internal/agent/tool_wrapper.go`
- `internal/service/chat.go`
- `internal/tools/error_policy.go`（新增）
- `internal/tools/error_policy_test.go`（新增）
- `internal/agent/tool_wrapper_test.go`（新增）

**前端修改:**
- `frontend/src/App.vue`
- `frontend/src/components/chat/ChatPanel.vue`
- `frontend/src/locales/zh.ts`
- `frontend/src/locales/en.ts`
- （同批次已存在）`frontend/src/components/chat/MessageBubble.vue`
- （同批次已存在）`frontend/src/components/chat/TimelineEventItem.vue`

**文档修改:**
- `plan/README.md`
- `plan/012-agent-mode-and-tool-error-recovery.md`（新增）

## 风险与注意事项

- 可恢复错误覆盖范围当前以高频场景为主（`str_replace_editor`、`read_file`、`list_files` 的路径/参数问题）
- 过度放宽 recoverable 会掩盖真实系统故障，因此默认仍采用保守 fatal 策略
- 自动切换 Plan Mode 使用启发式关键词判定，后续可引入更稳健的结构化判定

## 状态

已完成

