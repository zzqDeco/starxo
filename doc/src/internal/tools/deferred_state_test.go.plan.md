# deferred_state_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/tools/deferred_state_test.go`
- 文档文件: `doc/src/internal/tools/deferred_state_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: tools

## 2. 核心职责
- 验证 `ComputeDeferredMCPState(...)` 在 default / plan mode 下的 searchable、loadable 和 current loaded 计算结果。

## 3. 输入与输出
- 输入来源:
  - `ToolCatalog`
  - discovered tool 记录
  - `ToolPermissionContext`
- 输出结果: `DeferredMCPState`

## 4. 关键测试覆盖
- default mode 下：
  - connected server 可搜索也可加载
  - pending + cached metadata 只可搜索不可加载
  - pending + no cache 不会暴露具体 tool names
  - always-loaded 和 resource read 工具会进入 current loaded / loadable
- plan mode 下：
  - 只有显式可信的只读 deferred tools 才可搜索和加载
  - 未信任的只读提示或可写工具都会被排除

## 5. 依赖关系
- 内部依赖: `deferred_state.go`、`catalog.go`
- 外部依赖: `internal/model`

## 6. 变更影响面
- 保护 announcement、tool_search、visible tool list 和 execution gating 共用的 deferred state 语义。

## 7. 维护建议
- 扩展 server state 或 read-only 规则时，优先在本文件补 mode 级覆盖。
