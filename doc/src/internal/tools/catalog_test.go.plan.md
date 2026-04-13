# catalog_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/tools/catalog_test.go`
- 文档文件: `doc/src/internal/tools/catalog_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: tools

## 2. 核心职责
- 验证 MCP canonical name 规范化和 `ToolCatalog` 的注册 / 查找 / 冲突处理。

## 3. 输入与输出
- 输入来源: MCP server/tool 名称和手工构造的 `CatalogEntry`
- 输出结果: canonical name、`ToolCatalog` 查找结果和注册错误

## 4. 关键测试覆盖
- 名称规范化会清理空白与非法字符
- `CanonicalMCPToolName(...)` 生成稳定 canonical name
- catalog 支持 canonical 和 alias 的大小写无关 exact lookup
- canonical 冲突和 alias 冲突都会被拒绝

## 5. 依赖关系
- 内部依赖: `catalog.go`
- 外部依赖: `github.com/cloudwego/eino/components/tool`、`github.com/cloudwego/eino/schema`

## 6. 变更影响面
- 保护 deferred MCP name 语义，影响 announcement、tool_search、discovery 持久化与 late binding。

## 7. 维护建议
- 修改 normalize 或 alias 规则时，必须同步维护这些冲突与 lookup 测试。
