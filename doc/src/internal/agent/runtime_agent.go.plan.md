# runtime_agent.go 技术说明

## 1. 文件定位
- 源文件: `internal/agent/runtime_agent.go`
- 文档文件: `doc/src/internal/agent/runtime_agent.go.plan.md`
- 所属模块: agent

## 2. 核心职责
- 构建默认顶层 Eino v0.9 `ChatModelAgent` runtime。
- 将直接工具、Runtime V2 工具、ToolSearch/context middleware、current-objective behavior middleware 装配成 Claude Code-style ReAct loop。

## 3. 关键实现细节
- `BuildRuntimeAgent(...)` 接收模型、sandbox operator、always-load tools、AgentContext、mode、middleware 和 unknown-tool handler。
- 默认 direct tools 包括 ask/choice/notify/todos；Runtime V2 core tools 由 service 层通过 `extraTools` 注入。
- `RuntimeAgentPrompt(...)` 明确：当前 objective 是唯一 active task；旧消息只作为历史；`Agent` 是可选委派而不是默认分工。
- plan mode 只改变 prompt 和 permission surface，执行仍在同一个 ChatModelAgent loop 内完成。

## 4. 维护边界
- 不在这里做权限、workspace guard 或 ToolSearch state 持久化；这些仍由 service/tools 层负责。
- 如新增 always-load direct tool，需要同时检查 permission wrapper、timeline 展示和 docs。
