# mcp_resources.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/mcp_resources.go`
- 文档文件: `doc/src/internal/tools/mcp_resources.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 实现 MCP resources 子集：
  - `list_mcp_resources`
  - `list_mcp_resource_templates`
  - `read_mcp_resource`

## 3. 输入与输出
- 输入来源: `MCPHandleSource`
- 输出结果: invokable tools 与对应 catalog entries

## 4. 关键实现细节
- 资源工具默认 `ShouldDefer=true`
- 资源工具作为 global MCP entries 进入 deferred catalog
- `read_mcp_resource` 文本可截断，二进制默认只回 metadata，显式请求时才返回 base64

## 5. 依赖关系
- 内部依赖: `mcp_runtime.go`

## 6. 变更影响面
- 资源读取链路不再依赖 `officialmcp`，而由 Starxo 自己维护

## 7. 维护建议
- 若后续做 server-scoped resource announcement，仍应保持这三个工具名稳定
