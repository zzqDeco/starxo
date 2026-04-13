# dynamic_mcp_surface_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/tools/dynamic_mcp_surface_test.go`
- 文档文件: `doc/src/internal/tools/dynamic_mcp_surface_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: tools

## 2. 核心职责
- 验证 deferred MCP middleware 对 visible tools、announcement 和 execution gating 的最终执行语义。

## 3. 输入与输出
- 输入来源: fake `DeferredMCPProvider`、构造的 `ToolCatalog`
- 输出结果:
  - 过滤后的 `ToolInfo` 集合
  - announcement 注入结果
  - `ensureToolCallable(...)` 的允许/拒绝结果

## 4. 关键测试覆盖
- `WrapModel(...)` 只暴露当前 loaded tools 和非 catalog direct tools
- announcement 只显示 canonical names，不泄漏 search hint
- 已加载的 deferred tool 直接可调用
- 未加载但可搜索的 deferred tool 会被引导先用 `tool_search`
- catalog 中存在但当前不可搜索的 tool 会直接返回 unavailable

## 5. 依赖关系
- 内部依赖: `dynamic_mcp_surface.go`、`deferred_state.go`
- 外部依赖: `github.com/cloudwego/eino/adk`

## 6. 变更影响面
- 保护 deferred MCP 工具面对模型和执行层的一致性。

## 7. 维护建议
- 若调整 visible surface 规则，应同时验证 announcement 与 execution gate 不会分叉。
