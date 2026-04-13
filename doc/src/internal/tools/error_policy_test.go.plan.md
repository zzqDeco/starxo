# error_policy_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/tools/error_policy_test.go`
- 文档文件: `doc/src/internal/tools/error_policy_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: tools

## 2. 核心职责
- 验证 `ClassifyToolError(...)` 的 recoverable / fatal 分类语义。

## 3. 输入与输出
- 输入来源: tool 名称、JSON 参数和原始错误文本
- 输出结果: `ToolErrorDecision`

## 4. 关键测试覆盖
- `str_replace_editor` 的 `view_range` 越界会生成 recoverable decision 和修复 hint
- `old_str not found` 会按 recoverable 处理
- 默认未知错误保持 fatal
- `read_file` 缺路径会产生 `path_not_found` recoverable decision

## 5. 依赖关系
- 内部依赖: `error_policy.go`
- 外部依赖: `errors`、`strings`

## 6. 变更影响面
- 直接影响 `eventEmittingTool` 是否允许 Agent 自修复而不是立即中断。

## 7. 维护建议
- 增加新的 recoverable 模式时，需同步补标准化消息和 signature 断言。
