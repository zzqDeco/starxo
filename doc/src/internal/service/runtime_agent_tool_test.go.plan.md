# runtime_agent_tool_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_agent_tool_test.go`
- 文档文件: `doc/src/internal/service/runtime_agent_tool_test.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 覆盖 Runtime V2 `Agent` tool 的输入规范化、worktree 输出格式和子 agent instruction 元数据。

## 3. 关键测试覆盖
- `normalizeRuntimeAgentInput` 会填充 registry default / 默认 isolation；省略 `subagent_type` 标记为 fork，并拒绝未知 `subagent_type` 或 `isolation`。
- `Agent` tool schema 会使用配置 registry 的 subagent enum，不泄漏被替换掉的内置名称。
- fork 子 agent 不继承 registry default 的 `allowedTools` 收窄；explicit subagent 继续按 definition 过滤。
- worktree isolation 会派生子 agent 专用 `AgentContext`，不修改父 workspace。
- `formatRuntimeAgentRunResult` 会把 isolated worktree path/branch 写入结果文本，后台 task output 也可复用同一格式。
- `runtimeSubagentInstruction` 会明确实际 workspace 和 worktree isolation 语义。

## 4. 维护建议
- 后续支持 model/mode override 时，应补充对应 normalization 和 instruction 测试。
- 若 Agent output DTO 增加字段，需要同步覆盖 formatter 和 Wails bindings。
