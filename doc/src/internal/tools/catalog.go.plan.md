# catalog.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/tools/catalog.go`
- 文档文件: `doc/src/internal/tools/catalog.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 提供 metadata-aware `ToolCatalog`，用于统一保存 deferred MCP 工具的 canonical name、aliases、权限提示和展示属性。

## 3. 输入与输出
- 输入来源: MCP action/resource tool entries
- 输出结果: 可查询、可排序、可精确匹配的 catalog

## 4. 关键实现细节
- canonical format: `mcp__<normalize(server)>__<normalize(tool)>`
- `NormalizeMCPNamePart(...)` 统一用于 catalog、announcement、tool_search、discovery 持久化
- exact match key 为大小写不敏感的 canonical/alias 精确匹配
- canonical 冲突、alias 冲突都直接报错

## 5. 依赖关系
- 外部依赖: `github.com/cloudwego/eino/components/tool`

## 6. 变更影响面
- 决定 announcement、tool_search、late binding 的名字语义是否一致

## 7. 维护建议
- 新增 MCP tool naming 规则时必须只改这一处，不要分散复制 normalize 逻辑
