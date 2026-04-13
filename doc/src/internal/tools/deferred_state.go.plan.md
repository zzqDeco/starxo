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
- `SearchablePoolForMode` / `LoadablePoolForMode` 只属于真正 deferred 的 entries：`ShouldDefer == true && !AlwaysLoad`
- `effectiveDiscovered = discoveredTools ∩ current catalog ∩ loadablePoolForMode`
- `CurrentLoadedTools` 仍表示“当前直接可用的已加载集合”：
  - 已加载 deferred entries
  - 全部 `AlwaysLoad == true` entries
- `ShouldDefer == false && AlwaysLoad == false` 的 entry 在这一层只被排除出 deferred activation，不在本文件额外定义其直接暴露策略
- `ReadOnlyTrusted` / MCP read-only gate 仍只作用于 MCP deferred pool；非 MCP hidden/test-only sample 不在这套 gate 内

## 5. 依赖关系
- 内部依赖: `permissions.go`、`catalog.go`

## 6. 变更影响面
- announcement、tool_search、WrapModel、tool execution gating 都必须共享这一层结果
- 2C 之后这一层不再是 MCP-only 特例
- `SearchDecisions` / `LoadDecisions` 仍需对所有 catalog entry 保持全量覆盖，不能因 pool 收紧而被顺手裁掉

## 7. 维护建议
- 若 helper 输出字段变化，需同步检查 ChatService provider、middleware 和测试
