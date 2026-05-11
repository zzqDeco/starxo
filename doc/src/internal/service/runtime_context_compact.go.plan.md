# runtime_context_compact.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_context_compact.go`
- 文档文件: `doc/src/internal/service/runtime_context_compact.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 生成并刷新 Runtime V2 的 session-scoped compact state。
- 在准备模型消息前把 compact state 注入 token-aware windowing。
- 从 runtime tool result 中提取文件 read state 和 diff summary。

## 3. 输入与输出
- 输入来源:
  - `SessionRun` 的 messages、discovered tools、permission grants、plan state
  - `runtimeTaskManager` 的 task snapshots
  - `runtimeWorkspaceManager` 的 active worktree state
  - `tools.SnapshotTodosForSession(sessionID)`
  - Runtime tool call/result JSON
- 输出结果:
  - `model.RuntimeContextCompact`
  - 带 compact synthetic message 的 `[]*schema.Message`

## 4. 关键实现细节
- `refreshRuntimeContextCompact(...)` 是 compact state 的统一构建入口，并只读取当前 `sessionID` 的 todo bucket。
- compact summary 是 deterministic summary，不调用 LLM。
- `Read`/`read_file` 记录 file path、line range、total lines 和返回内容 hash。
- `Write`/`write_file`、`Edit`/`str_replace_editor` 记录最近 diff summary，包括增删行、替换数和可用 patch 摘要。
- `prepareMessagesForRun(...)` 在每次 agent run 前刷新 compact state，并调用 context 层 token-aware windowing。

## 5. 维护建议
- 新增 Runtime tool 时，如果它会改变文件或长期状态，应在 `recordRuntimeToolResult(...)` 中加入 compact 提取逻辑。
- 不要在本文件里调用 `SessionService.GetWorkspacePath()`；保存路径可能持有 SessionService lock，避免死锁。
