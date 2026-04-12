# permissions.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/permissions.go`
- 文档文件: `doc/src/internal/tools/permissions.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 定义 MCP deferred 子集的 permission-first 规则，统一计算 searchable / loadable 判定。

## 3. 输入与输出
- 输入来源: `ToolPermissionContext`、`CatalogEntry`
- 输出结果:
  - `CanSearchCatalogEntry(...)`
  - `CanLoadCatalogEntry(...)`
  - 执行期 wrapper `WrapMCPToolWithPermissionCheck(...)`

## 4. 关键实现细节
- `plan mode` 只接受显式且可信的 `ReadOnlyHint=true`
- `pending` server 只有具备 cached metadata 时才允许贡献 searchable names
- global resource tools 通过 `SupportsResources` 聚合判定

## 5. 依赖关系
- 内部依赖: `catalog.go`、`mcp_runtime.go`

## 6. 变更影响面
- announcement、tool_search、late binding、execution gating 都依赖这一层的 mode / runtime / trust 规则

## 7. 维护建议
- 不要在 prompt 或 middleware 里重复实现只读/权限逻辑，统一落在这里
