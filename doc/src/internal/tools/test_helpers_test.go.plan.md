# test_helpers_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/tools/test_helpers_test.go`
- 文档文件: `doc/src/internal/tools/test_helpers_test.go.plan.md`
- 文件类型: Go 测试辅助文件
- 所属模块: tools

## 2. 核心职责
- 提供 `tools` 测试套件共享的 stub tool 和 `CatalogEntry` 构造器。

## 3. 输入与输出
- 输入来源: 测试中传入的 canonical name
- 输出结果:
  - `stubInvokableTool`
  - `stubCatalogEntry(...)`

## 4. 关键实现细节
- `stubCatalogEntry(...)` 默认构造一个可搜索、可执行、deferred、MCP action 的 catalog entry
- 该辅助使 `deferred_state_test.go`、`dynamic_mcp_surface_test.go`、`tool_search_test.go` 的构造保持一致

## 5. 依赖关系
- 内部依赖: `catalog.go`
- 外部依赖: `github.com/cloudwego/eino/components/tool`、`github.com/cloudwego/eino/schema`

## 6. 变更影响面
- 影响多份 `tools` 测试的初始假设；修改 helper 等于同时修改多份测试样本。

## 7. 维护建议
- 若默认 stub 权限或 `CatalogEntry` 字段发生变化，应检查所有依赖本 helper 的测试语义是否仍成立。
