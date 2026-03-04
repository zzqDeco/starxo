# tool_wrapper.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/tool_wrapper.go
- 文档文件: doc/src/internal/agent/tool_wrapper.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件实现了工具事件发射包装器，使子代理（code_writer、code_executor、file_manager）的工具调用对前端可见。它通过 `eventEmittingTool` 结构体装饰 `tool.BaseTool`，在工具调用前后分别发射 `tool_call` 和 `tool_result` 事件。`WrapToolsWithEvents` 函数批量包装一组工具，为每个工具注入代理名称和事件回调。
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
- 导出函数/方法:
  - `WrapToolsWithEvents(agentName string, tools []tool.BaseTool, ac AgentContext) []tool.BaseTool`: 批量包装工具，如果 `ac.OnToolEvent` 为 nil 则直接返回原工具列表
- 内部方法:
  - `(*eventEmittingTool) Info(ctx) (*schema.ToolInfo, error)`: 委托给内部工具
  - `(*eventEmittingTool) InvokableRun(ctx, argumentsInJSON, opts...) (string, error)`: 生成唯一 callID（基于纳秒时间戳），**传递 ctx 到 onEvent**（使事件能路由到正确的 session），发射 tool_call 事件，执行内部工具，发射 tool_result 事件（错误时结果为 "Error: ..."）
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
