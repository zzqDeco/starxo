# tool_search.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/tool_search.go`
- 文档文件: `doc/src/internal/tools/tool_search.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 实现 Starxo-native 的 `tool_search` backend，服务于 deferred MCP 子集。

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
- `matches` 一律返回 canonical name
- 零命中时才返回 `pending_mcp_servers`
- 只有新发现 deferred tool 才产生 `DiscoveredToolRecord`

## 5. 依赖关系
- 内部依赖: `catalog.go`、`session_data.go`

## 6. 变更影响面
- 决定 deferred MCP discovery 的写入边界和模型与工具面的交互稳定性

## 7. 维护建议
- 若将来扩展 ranking，不要改变 canonical output / no-op 语义
