# deep_agent.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/agent/deep_agent.go`
- 文档文件: `doc/src/internal/agent/deep_agent.go.plan.md`
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 构建顶层 `coding_agent`，统一组装：
  - orchestration tools
  - Runtime V2 always-load tools
  - Eino v0.9 ToolSearch/Skill/AGENTS.md/reduction/summarization middleware
  - 可选旧 transfer sub-agents fallback
- 在 `default` / `plan` 两种模式下复用同一套 deep agent 框架，只改变 prompt 与 deferred MCP policy。

## 3. 输入与输出
- 输入来源: `context.Context`、`model.ToolCallingChatModel`、`commandline.Operator`、`extraTools`、`AgentContext`、`DeepAgentMode`
- 输出结果: `adk.Agent`

## 4. 关键实现细节
- `BuildDeepAgentForMode(...)` 现在额外接收：
  - `handlers []adk.ChatModelAgentMiddleware`
  - `unknownToolsHandler`
  - `enableTransferFallback`
- 两种 mode 都会挂载 `extraTools`，区别不再是“plan mode 不带 extraTools”，而是 deferred helper 在运行期决定 searchable/loadable。
- 顶层 direct orchestration tools 保持为：
  - `ask_user`
  - `ask_choice`
  - `notify_user`
  - `write_todos`
  - `update_todo`
- Runtime V2 core tools 由 `ChatService` 作为 `extraTools` 注入，包含 `Read`、`Write`、`Edit`、`Bash`、`Glob`、`Grep`、`TaskOutput`、`TaskStop`、`ExitPlanMode`、`Agent` 等。
- 默认不再构建固定 transfer sub-agents；只有 `agent.runtime.enableBuiltinDeepTransferFallback=true` 时才装配旧 `code_writer` / `code_executor` / `file_manager`。
- Context middleware 通过 `NewEinoV09ContextMiddlewares(...)` 注入 Skill、AGENTS.md、large result reduction 和 summarization。

## 5. 依赖关系
- 内部依赖:
  - `internal/tools`: follow-up、choice、notify、todos
  - `internal/agent/eino_v09_context.go`
  - `internal/agent/codewriter.go`（fallback only）
  - `internal/agent/codeexecutor.go`（fallback only）
  - `internal/agent/filemanager.go`（fallback only）
  - `internal/agent/prompts.go`
- 外部依赖:
  - `github.com/cloudwego/eino/adk`
  - `github.com/cloudwego/eino/adk/prebuilt/deep`
  - `github.com/cloudwego/eino/compose`

## 6. 变更影响面
- 顶层工具面现在允许 Eino v0.9 ToolSearch 和 Starxo deferred middleware 在每次模型调用前裁剪可见工具。
- `UnknownToolsHandler` 成为“已知但未加载工具 -> 提示先 `tool_search`”的恢复路径。
- plan mode 的 read-only MCP 约束不在本文件硬编码，而由上层 permission helper + middleware 驱动。

## 7. 维护建议
- 若新增顶层 direct tool，需同时评估它是否应进入 deferred catalog。
- 不要在本文件引入 session 级状态；top-level deep agent 必须保持跨 session 可共享。
