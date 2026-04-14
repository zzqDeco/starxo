# tool_search.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/tool_search.go`
- 文档文件: `doc/src/internal/tools/tool_search.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 实现 Starxo-native 的 `tool_search` backend，服务于 generic deferred framework。

## 3. 输入与输出
- 输入来源: `ToolSearchInput`
- 输出结果: `ToolSearchOutput`

## 4. 关键实现细节
- 支持：
  - `select:<tool>`
  - `select:A,B,C`
  - bare exact-name
  - 关键词搜索
  - `+term` 必选词
- exact-name 对 canonical 和 aliases 做大小写无关匹配
- provider 传入的 `CurrentLoaded` 语义固定为 loaded deferred only，不直接复用全部 current loaded tools
- 命中当前已加载 deferred tool 时直接返回 canonical name，不再重复写 discovery
- `matches` 一律返回 canonical name
- 零命中时才返回 `pending_mcp_servers`
- 只有新发现 deferred tool 才产生 `DiscoveredToolRecord`
- 对非 MCP deferred sample，`CanonicalName == tool name`
- exact-name、`select:`、keyword search 对非 MCP sample 也返回同一个名字
- `AlwaysLoad == true` 和 `ShouldDefer == false` 的 entry 不应通过 `tool_search` 暴露；它们的可见性由正常工具面决定，不走 deferred activation
- `ToolSearchUnavailableNoDeferredMessage` 是共享 contract，unknown-tool fallback 与 middleware 都复用同一来源
- dev-only experimental sample 走和其它非 MCP deferred builtin 相同的名字语义：`CanonicalName == tool name`

## 5. 依赖关系
- 内部依赖: `catalog.go`、`session_data.go`

## 6. 变更影响面
- 决定 generic deferred discovery 的写入边界和模型与工具面的交互稳定性

## 7. 维护建议
- 若将来扩展 ranking，不要改变 canonical output / no-op 语义
