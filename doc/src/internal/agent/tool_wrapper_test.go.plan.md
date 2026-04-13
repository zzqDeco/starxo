# tool_wrapper_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/agent/tool_wrapper_test.go`
- 文档文件: `doc/src/internal/agent/tool_wrapper_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: agent

## 2. 核心职责
- 验证 `eventEmittingTool` 对可恢复工具错误的归类、升级和重置行为。
- 保护工具包装层“让 Agent 自修复而不是立即中断”的错误处理语义。

## 3. 输入与输出
- 输入来源:
  - `scriptedInvokableTool`
  - 挂载了 `sessionID` 的 `context.Context`
- 输出结果:
  - `InvokableRun(...)` 的返回文本和错误值
  - 可恢复错误计数状态变化

## 4. 关键测试覆盖
- view range 越界会被视为 recoverable，并返回 hint 文本而不是 fatal error
- 同一可恢复错误重复到阈值后会升级为真正错误
- 成功调用会清空该会话的 recoverable backoff，后续重新从低阈值开始累计
- 非 recoverable 错误仍然直接失败
- `read_file` 缺路径被视为 recoverable，并返回面向 Agent 的修正提示

## 5. 依赖关系
- 内部依赖: `tool_wrapper.go`
- 外部依赖: `github.com/cloudwego/eino/components/tool`、`github.com/cloudwego/eino/schema`

## 6. 变更影响面
- 该测试文件直接保护工具包装层的错误升级节奏，影响 `code_writer` / `code_executor` / `file_manager` 的稳定性。

## 7. 维护建议
- 若调整 recoverable error 分类或阈值，需同步更新本文件与 `internal/tools/error_policy_test.go`。
