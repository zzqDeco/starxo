# error_policy.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/tools/error_policy.go
- 文档文件: doc/src/internal/tools/error_policy.go.plan.md
- 文件类型: Go 源码
- 所属模块: tools

## 2. 核心职责
- 该文件定义工具错误分类策略，将工具执行错误划分为可恢复（recoverable）与不可恢复（fatal），并为可恢复错误生成标准化提示信息与错误签名。
- 分类结果供 `internal/agent/tool_wrapper.go` 使用，用于决定是否将错误回传给 Agent 继续自修复，或升级为节点失败。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `toolName`、`toolArgs`、`error`
- 输出结果: `ToolErrorDecision`
  - `Recoverable`: 是否可恢复
  - `NormalizedMsg`: 规范化错误消息（附带重试提示）
  - `Signature`: 错误签名（用于重复错误计数）

## 4. 关键实现细节
- 结构体/接口定义:
  - `ToolErrorDecision`: 错误分类结果结构体
- 导出函数/方法:
  - `ClassifyToolError(toolName, toolArgs string, err error) ToolErrorDecision`
- 内部方法:
  - `classifyStrReplaceEditorError(...)`: 处理 `str_replace_editor` 常见可恢复错误（`view_range` 越界、`old_str not found`、`invalid line range`）
  - `isMissingPathError(lowerErr string) bool`: 判断路径不存在类错误（兼容 Linux/Windows 常见报错文本）
  - `shortHash(s string) string`: FNV-1a 64-bit 短哈希，生成签名片段
- 策略范围（当前）:
  - `str_replace_editor`: 常见参数错误归类为 recoverable
  - `read_file` / `list_files`: 路径不存在归类为 recoverable
  - 其他工具错误默认 fatal（保守策略）
- Wails 绑定方法: 无
- 事件发射: 无（仅返回分类结果）

## 5. 依赖关系
- 内部依赖: 被 `internal/agent/tool_wrapper.go` 调用
- 外部依赖:
  - `fmt`
  - `strings`
  - `hash/fnv`

## 6. 变更影响面
- 影响 Agent 工具错误传播语义：可恢复错误不再立即导致 NodeRunError 中断
- 影响子代理自修复能力：错误提示文本会进入 `tool_result` 并反馈给 LLM
- 影响循环保护：错误签名用于重复错误计数与升级策略

## 7. 维护建议
- 修改匹配规则时，优先保持“保守默认 fatal”，避免把真正系统故障误判为 recoverable。
- 新增 recoverable 场景时，建议同步补充 `internal/tools/error_policy_test.go` 和 `internal/agent/tool_wrapper_test.go`。
- 错误签名应保持稳定且可区分，避免不同错误被错误合并计数。
