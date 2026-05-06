# runtime_agent_tool.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_agent_tool.go`
- 文档文件: `doc/src/internal/service/runtime_agent_tool.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 为 Runtime V2 提供 `Agent` 工具，把动态子 agent 纳入统一 catalog、permission、timeline 和 task 体系。
- 支持同步委派、后台委派和可选 worktree 隔离。

## 3. 输入与输出
- 输入来源: 顶层 agent 的 `Agent` tool call、当前 sandbox operator、当前模型、runtime permission provider。
- 输出结果:
  - 同步模式返回子 agent 汇总文本。
  - 后台模式返回 runtime task id 和输出文件路径。
  - worktree 隔离模式附带 worktree path/branch。

## 4. 关键实现细节
- `runtimeAgentInput` 支持 `description`、`prompt`、`subagent_type`、`background`、`isolation`，`model` 和 `mode` 作为后续扩展保留字段。
- `Agent` 是 always-load runtime tool，但执行仍会经过 permission wrapper。
- 子 agent 使用同一个 Eino chat model，并注入 Runtime V2 core tools；子工具同样走 permission gate 和 timeline event wrapper。
- `background=true` 时调用 `runtimeTaskManager.StartAgentTask`，任务输出落盘到 runtime task output 文件。
- `isolation=worktree` 时通过 `runtimeWorkspaceManager.EnterWorktree` 创建 session-scoped git worktree；完成后保持 worktree 在磁盘上并恢复原 workspace。

## 5. 依赖关系
- 内部依赖: `internal/agent`、`internal/tools`
- 外部依赖: `github.com/cloudwego/eino/adk`、`github.com/cloudwego/eino/components/tool/utils`

## 6. 变更影响面
- 顶层 Runtime V2 工具面新增动态 `Agent` 能力。
- 后台 agent 与后台 Bash 共用 runtime task lifecycle。
- worktree 隔离依赖当前 workspace 是 git repository。

## 7. 维护建议
- 后续若启用 `model`/`mode` override，必须同步处理权限、成本和 runner lifecycle。
- 子 agent 可用工具增加时，应先确认是否需要 deferred loading，而不是默认注入所有工具。
