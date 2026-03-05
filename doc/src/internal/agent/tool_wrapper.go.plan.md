# tool_wrapper.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/tool_wrapper.go
- 文档文件: doc/src/internal/agent/tool_wrapper.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件实现了工具事件发射包装器，使子代理（code_writer、code_executor、file_manager）的工具调用对前端可见。它通过 `eventEmittingTool` 结构体装饰 `tool.BaseTool`，在工具调用前后分别发射 `tool_call` 和 `tool_result` 事件。`WrapToolsWithEvents` 函数批量包装一组工具，为每个工具注入代理名称和事件回调。
- 新增可恢复错误处理链路：对工具参数类错误可回传给 Agent 继续自修复，避免直接 NodeRunError 中断；同签名重复失败达到阈值后再升级为 fatal。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - `WrapToolsWithEvents`: 代理名称（string）、工具列表（`[]tool.BaseTool`）、`AgentContext`（提供 OnToolEvent 回调）
- 输出结果:
  - 返回包装后的 `[]tool.BaseTool`，行为与原工具相同但增加了事件发射副作用
  - 通过 `AgentContext.OnToolEvent` 回调发射事件（参数: ctx, agentName, eventType, toolName, toolArgs, toolID, result）

## 4. 关键实现细节
- 结构体/接口定义:
  - `eventEmittingTool`: 内部结构体，实现 `tool.BaseTool` 和 `tool.InvokableTool` 接口
    - `inner tool.BaseTool`: 被包装的原始工具
    - `agentName string`: 所属代理名称
    - `toolName string`: 工具名称
    - `onEvent func(ctx context.Context, agentName, eventType, toolName, toolArgs, toolID, result string)`: 事件回调。**类型变更**: 新增 `ctx context.Context` 作为首参数，用于传播 session 身份
    - `recoverableErrCount map[string]int`: 按会话和错误签名统计连续可恢复错误次数
- 常量:
  - `recoverableErrorEscalationThreshold = 3`
- 导出函数/方法:
  - `WrapToolsWithEvents(agentName string, tools []tool.BaseTool, ac AgentContext) []tool.BaseTool`: 批量包装工具，如果 `ac.OnToolEvent` 为 nil 则直接返回原工具列表
- 内部方法:
  - `(*eventEmittingTool) Info(ctx) (*schema.ToolInfo, error)`: 委托给内部工具
  - `(*eventEmittingTool) sessionScope(ctx) string`: 读取会话作用域（`sessionID`）
  - `(*eventEmittingTool) incrementRecoverableError(...)` / `clearRecoverableError(...)` / `clearRecoverableErrorsForSession(...)`: 计数与清理
  - `(*eventEmittingTool) InvokableRun(ctx, argumentsInJSON, opts...) (string, error)`:
    - 始终先发射 `tool_call`
    - 调用 `tools.ClassifyToolError` 分类错误
    - recoverable: 返回 `normalized error text + nil`，让 Agent 继续下一步
    - 连续同签名 recoverable 错误达到阈值: 升级为 fatal，返回非空 error
    - fatal: 保持原有失败路径
    - 成功调用后清空当前会话的 recoverable 计数
- Wails 绑定方法: 无
- 事件发射: 通过 `onEvent` 回调发射 `tool_call`（含 toolArgs）和 `tool_result`（含 result 或 error）事件

## 5. 依赖关系
- 内部依赖:
  - 同包 `agent`: `AgentContext`
- 外部依赖:
  - `context`（标准库）: 用于传播 session 身份
  - `github.com/cloudwego/eino/components/tool`: BaseTool、InvokableTool、Option 接口
  - `github.com/cloudwego/eino/schema`: ToolInfo
- 关键配置: 无

## 6. 变更影响面
- `onEvent` 回调类型变更（新增 `ctx context.Context` 首参数）与 `AgentContext.OnToolEvent` 签名一致
- `InvokableRun` 中的 `ctx` 参数现在被传递到 `onEvent` 回调，使 `chat.go` 中的 `emitTimelineForSession` 能通过 `SessionIDFromContext(ctx)` 提取 sessionID
- `InvokableRun` 的错误返回语义变更会影响 ADK 对节点失败的判定：recoverable 错误不再立即中断整轮 run
- 新增重复错误阈值会影响错误升级时机（第 3 次同签名可恢复错误升级）
- 事件格式变更影响前端时间线渲染（`internal/service/chat.go` 中的 `buildAgentContext` 生成事件回调）
- callID 生成策略变更可能影响前端的工具调用-结果配对
- 被 `codewriter.go`、`codeexecutor.go`、`filemanager.go` 三个子代理文件调用
- 如果内部工具未实现 `InvokableTool` 接口会返回错误

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- callID 使用纳秒时间戳生成，在极高并发下可能重复，如需更高唯一性可考虑使用 UUID。
- 事件回调中的 result 可能包含敏感信息（文件内容等），如需脱敏应在此处添加截断/过滤逻辑。
- 新增子代理时应始终通过 `WrapToolsWithEvents` 包装其工具列表，保持前端可观测性。
- `ctx` 的传播是 per-session 事件路由的关键链路，确保子代理执行时 `ctx` 中包含正确的 sessionID。
