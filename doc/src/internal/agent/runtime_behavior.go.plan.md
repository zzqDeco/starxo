# runtime_behavior.go 技术说明

## 1. 文件定位
- 源文件: `internal/agent/runtime_behavior.go`
- 文档文件: `doc/src/internal/agent/runtime_behavior.go.plan.md`
- 所属模块: agent

## 2. 核心职责
- 提供 `RuntimeBehaviorMiddleware`，在每次 ChatModelAgent 调用模型前注入/更新 `<current-objective>` 消息。
- 把 Starxo 的 current-objective sidecar 转成模型可见的行为边界。

## 3. 关键实现细节
- 从 `context.Context` 读取 `tools.RuntimeObjectiveFromContext(...)`。
- `BeforeModelRewriteState(...)` 会去重旧 objective message，并把最新 objective 插入 system message 后。
- objective message 强调 older conversation 是 historical context，除 continuation 外不得恢复旧任务。
- `WrapInvokableToolCall(...)` 对同步工具调用应用 current-objective guard；明显追逐旧 debug/release/review 任务的调用会返回 tool error，不会进入真实工具执行。

## 4. 维护边界
- 这里不执行工具权限或任务取消；只改写模型消息视图，并提供高置信 stale tool-call 拦截。
- objective 文案变更会直接影响 agent 行为，应配套更新 prompt 测试和手工回归。
