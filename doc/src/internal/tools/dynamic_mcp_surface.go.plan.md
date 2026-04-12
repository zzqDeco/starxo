# dynamic_mcp_surface.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/dynamic_mcp_surface.go`
- 文档文件: `doc/src/internal/tools/dynamic_mcp_surface.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 实现 deferred MCP middleware，在每次模型调用前做 announcement 注入、visible tool filtering 和 execution gating。

## 3. 输入与输出
- 输入来源: `DeferredMCPProvider`
- 输出结果: `adk.ChatModelAgentMiddleware`

## 4. 关键实现细节
- `WrapModel(...)`：
  - 计算当前 deferred state
  - 注入 available deferred MCP announcement
  - 只把 current loaded tools + non-catalog direct tools 暴露给模型
- `WrapInvokableToolCall(...)` / `WrapStreamableToolCall(...)`：
  - `tool_search` 仅在 searchable pool 非空或 pending server 存在时可调用
  - hidden deferred tool 被调用时返回“先用 tool_search”

## 5. 依赖关系
- 内部依赖: `deferred_state.go`
- 外部依赖:
  - `github.com/cloudwego/eino/adk`
  - `github.com/cloudwego/eino/components/model`

## 6. 变更影响面
- 这是 deferred MCP visible surface 的最终执行点

## 7. 维护建议
- announcement 只显示 canonical names，不要在这里泄漏 schema 或 search hints
