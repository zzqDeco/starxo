# deferred_state.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/deferred_state.go`
- 文档文件: `doc/src/internal/tools/deferred_state.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 统一计算 per-session、per-mode 的 deferred helper 输出。

## 3. 输入与输出
- 输入来源: `ToolCatalog`、`discoveredTools`、`ToolPermissionContext`
- 输出结果: `DeferredMCPState`

## 4. 关键实现细节
- 输出固定包含：
  - `SearchablePoolForMode`
  - `LoadablePoolForMode`
  - `EffectiveDiscovered`
  - `CurrentLoadedTools`
  - `PendingMCPServers`
- `effectiveDiscovered = discoveredTools ∩ current catalog ∩ loadablePoolForMode`
- `ReadOnlyTrusted` / MCP read-only gate 仍只作用于 MCP deferred pool；非 MCP hidden/test-only sample 不在这套 gate 内

## 5. 依赖关系
- 内部依赖: `permissions.go`、`catalog.go`

## 6. 变更影响面
- announcement、tool_search、WrapModel、tool execution gating 都必须共享这一层结果
- 2C 之后这一层不再是 MCP-only 特例

## 7. 维护建议
- 若 helper 输出字段变化，需同步检查 ChatService provider、middleware 和测试
