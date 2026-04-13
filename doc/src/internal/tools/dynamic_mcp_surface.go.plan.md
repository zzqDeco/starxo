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
  - 基于 provider 准备 synthetic delta messages
  - 当前 phase-2A 先注入 `deferred-tools-delta`
  - 只把 current loaded tools + non-catalog direct tools 暴露给模型
  - 只有 `Generate(...)` 成功返回消息或 `Stream(...)` 成功返回 reader 后才执行 state commit
- searchable canonical names 的规范化、delta 计算、wire formatting 都收成单点 helper
- `deferred-tools-delta` wire 固定为：
  - `mode: bootstrap|delta`
  - `added:` / `removed:` 两段始终保留
  - canonical names 稳定排序输出
- `WrapInvokableToolCall(...)` / `WrapStreamableToolCall(...)`：
  - `tool_search` 仅在 searchable pool 非空或 pending server 存在时可调用
  - 已在 current loaded tools 内的 deferred tool 会直接放行，不会再误导模型先去 `tool_search`
  - 未加载但当前可搜索的 deferred tool 被调用时返回“先用 tool_search”
  - catalog 中存在但当前 mode / permission / runtime 下不可搜索的 tool 会直接返回 unavailable，不再误导去搜
  - announcement、tool_search、visible tool list、execution gating 共用同一份 deferred state

## 5. 依赖关系
- 内部依赖: `deferred_state.go`
- 外部依赖:
  - `github.com/cloudwego/eino/adk`
  - `github.com/cloudwego/eino/components/model`

## 6. 变更影响面
- 这是 deferred MCP visible surface 的最终执行点

## 7. 维护建议
- deferred tools delta 只显示 canonical names，不要在这里泄漏 schema 或 search hints
- 后续 2B/2C 扩展时继续复用同一个 synthetic message 准备入口，不要让不同调用点各自计算 delta
