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
- `runtimeAgentInput` 支持 `description`、`prompt`、`subagent_type`、`background`、`isolation`，`model` 和 `mode` 作为后续扩展保留字段；内部 `Fork` 标记区分 omitted `subagent_type` 的 fork 路径。
- `runtimeSubagentRunner` 持有 clock、task manager、worktree manager 依赖，`Agent` tool 不再直接作为 `ChatService` 方法执行；`ChatService` 只在 bundle 构建时注入这些依赖。
- `subagent_type` 会通过 `agent.runtime.subagents` 对应的动态 registry 规范化；省略时 fork 当前 agent context，同时沿用 registry default 的 isolation/background policy，但不继承 default definition 的 allowedTools 收窄。
- `Agent` tool schema 会根据当前 registry 生成 `subagent_type` enum 和说明，避免自定义 registry 时继续提示内置名称。
- `isolation` 会规范化为 `none` 或 `worktree`；缺省值来自 subagent definition 的 `defaultIsolation`。
- subagent definition 可限制 `allowedTools`，也可通过 `backgroundAllowed=false` 禁止后台执行。
- `Agent` 是 always-load runtime tool，但执行仍会经过 permission wrapper。
- 子 agent 使用同一个 Eino chat model，并按 subagent definition 注入允许的 Runtime V2/MCP always-load 工具；子工具同样走 permission gate、workspace guard 和 timeline event wrapper。
- non-fork 子 agent 的 Eino ToolSearch 也按同一 `allowedTools` 策略过滤 deferred tools，避免子 agent 通过搜索发现 definition 之外的能力；fork 子 agent 才继承父 agent 的完整可搜索工具面。
- omitted `subagent_type` 使用 `RuntimeForkAgentPrompt` fork 当前 objective；该 prompt 不再暴露递归 `Agent` 委派，但会明确列出 fork 可用的 ask/notify/todo/direct runtime 工具。
- fork 子 agent 额外注入 ask_user、ask_choice、notify_user、write_todos、update_todo，避免提示词承诺的直接工具和实际工具池不一致。
- 子 agent runtime mode 使用“父会话 mode 与调用方 request mode 的更严格值”：父会话在 plan mode 时，显式 `mode=default` 不能降级；默认会话下显式 `mode=plan` 可以进一步收窄。
- 计算出的子 agent mode 会写入 runtime context override，并被 ToolSearch provider 与 permission queue 读取，确保 prompt、可搜索工具和审批策略一致。
- worktree isolation 会派生子 agent 专用 `AgentContext.WorkspacePath`，context middleware、tool wrapper 和 prompt 都使用隔离 worktree path，不污染父会话 workspace。
- `background=true` 时调用 `runtimeTaskManager.StartAgentTask`，任务输出落盘到 runtime task output 文件。
- `isolation=worktree` 时通过 `runtimeWorkspaceManager.CreateIsolatedWorktree` 创建 git worktree，并用 context-scoped workspace override 只影响当前子 agent；父 session 的 active workspace 不会被临时切走。
- 同步和后台 Agent 输出都会包含 worktree path/branch，便于后续人工审阅或合并。

## 5. 依赖关系
- 内部依赖: `internal/agent`、`internal/tools`
- 外部依赖: `github.com/cloudwego/eino/adk`、`github.com/cloudwego/eino/components/tool/utils`

## 6. 变更影响面
- 顶层 Runtime V2 工具面新增动态 `Agent` 能力。
- 后台 agent 与后台 Bash 共用 runtime task lifecycle。
- worktree 隔离依赖当前 workspace 是 git repository。
- 后台 isolated subagent 与同 session 的其他工具调用可以并行运行，workspace routing 不互相污染。

## 7. 维护建议
- 后续若启用 `model`/`mode` override，必须同步处理权限、成本和 runner lifecycle。
- 子 agent 可用工具增加时，应先确认是否需要 deferred loading，而不是默认注入所有工具。
- 新增或修改 subagent definition 时要同步检查 `allowedTools`、默认 worktree 策略和后台执行策略，避免子 agent 获得超出预期的能力。
