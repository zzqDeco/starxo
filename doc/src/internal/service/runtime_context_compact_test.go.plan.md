# runtime_context_compact_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_context_compact_test.go`
- 文档文件: `doc/src/internal/service/runtime_context_compact_test.go.plan.md`
- 所属模块: service tests

## 2. 核心职责
- 覆盖 Runtime context compact 的保存、恢复和 prompt 注入语义。

## 3. 覆盖点
- snapshot 包含 discovered tools、permission grants、file read state、diff summaries、runtime tasks、todos、plan 和 worktree。
- restore 后 compact internals 恢复，task snapshot 仍可见且 output 可读。
- 运行中的 task reload 后变为 failed，避免误表示后台进程仍附着。
- `prepareMessagesForRun(...)` 会注入 compact synthetic message。
- todo snapshot/restore 是深拷贝，且 session-scoped snapshot/restore 不串扰其他 session。
- runtime tool result JSON 解析失败时可从 arguments fallback 得到 file path。
