# tool_search.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/tool_search.go`
- 文档文件: `doc/src/internal/tools/tool_search.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 实现 Starxo-native 的 `tool_search` backend，服务于 generic deferred framework，并兼容 Eino v0.9 dynamic toolsearch schema。

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
- provider 传入的 `CurrentLoaded` 语义是当前 runtime surface：已加载 deferred tools + permission 允许的 always-load tools
- 命中当前已加载工具时直接返回 canonical name，不再重复写 discovery
- `matches` 一律返回 canonical name
- `ToolSearchInput.limit` 与 Eino v0.9 `max_results` alias 都会控制返回数量；`limit` 优先。
- 输出同时包含 `loaded`、`pendingSources`，并保留旧 JSON 字段 `pending_mcp_servers`
- `loaded` 始终包含 `tool_search` 自身，因为 Runtime V2 把它作为基础 always-load 工具
- 零命中时才返回 pending source/server 信息
- 只有新发现 deferred tool 才产生 `DiscoveredToolRecord`
- 对非 MCP deferred sample，`CanonicalName == tool name`
- exact-name、`select:`、keyword search 对非 MCP sample 也返回同一个名字
- `AlwaysLoad == true` 的 entry 可以作为 current loaded result 被 exact/select 命中，但不会写 discovery
- `ShouldDefer == false && AlwaysLoad == false` 的 entry 不通过 `tool_search` 暴露
- `ToolSearchUnavailableNoDeferredMessage` 是共享 contract，unknown-tool fallback 与 middleware 都复用同一来源
- dev-only experimental sample 走和其它非 MCP deferred builtin 相同的名字语义：`CanonicalName == tool name`
- Runtime V2 中 `tool_search` 由 Eino v0.9 middleware 暴露到模型面；Starxo 仍复用本文件的 ranking、filtering、pending source 和 discovery contract，并把 Eino tool_search 结果回写为 `DiscoveredToolRecord`。

## 5. 依赖关系
- 内部依赖: `catalog.go`、`runtime_tools.go`、`session_data.go`

## 6. 变更影响面
- 决定 generic deferred discovery 的写入边界和模型与工具面的交互稳定性

## 7. 维护建议
- 若将来扩展 ranking，不要改变 canonical output / no-op 语义
