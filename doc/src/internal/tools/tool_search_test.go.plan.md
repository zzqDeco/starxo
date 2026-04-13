# tool_search_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/tools/tool_search_test.go`
- 文档文件: `doc/src/internal/tools/tool_search_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: tools

## 2. 核心职责
- 验证 `ExecuteToolSearch(...)` 的 canonical 输出、discovery 写入边界和 pending server 语义。

## 3. 输入与输出
- 输入来源: `ToolSearchInput` 和 `ToolSearchState`
- 输出结果:
  - `ToolSearchOutput`
  - 新 discovery 记录集合

## 4. 关键测试覆盖
- exact match 命中当前已加载工具时只返回 canonical，不写 discovery
- alias 命中后仍返回 canonical names
- `select:` 支持部分命中而不是全量失败
- 零命中时才返回 `pending_mcp_servers`
- always-loaded 工具 exact match 不会产生 discovery
- keyword search 会写 discovery，并保留 `DiscoveredAt` 时间戳

## 5. 依赖关系
- 内部依赖: `tool_search.go`、`catalog.go`
- 外部依赖: `time`、`internal/model`

## 6. 变更影响面
- 保护 deferred MCP discovery 的写入边界与模型交互格式稳定性。

## 7. 维护建议
- 若扩展 ranking 或 query 语法，优先保住 canonical output 和 pending server 的现有语义。
