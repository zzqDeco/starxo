# runtime_worktree_ui.go 技术说明

## 文件定位
- 源文件: `internal/service/runtime_worktree_ui.go`
- 所属模块: service

## 核心职责
- 为桌面端工作区抽屉提供 Runtime V2 worktree 审阅/合并 Wails API。
- 将内部 `runtimeWorkspaceManager` 的 session-scoped worktree state 转换为前端可读 DTO。

## 关键实现细节
- `GetRuntimeWorktreeState(sessionID)` 支持空 sessionID，空值时读取 ChatService 当前 active session。
- `ReviewRuntimeWorktree(...)` 调用 `DiffWorktree`，返回 status、diff stat 和可选 patch。
- `MergeRuntimeWorktree(...)` 调用 `MergeWorktree`，成功后发出 `runtime:worktree_changed`；结构化 conflict output 不作为错误抛出，但事件 action 使用 `merge_conflict`，避免监听方误判为成功合并。
- `ExitRuntimeWorktree(...)` 复用 `ExitWorktree` 的 keep/remove 语义，成功后发出 `runtime:worktree_changed`。
- UI API 只操作已有 active worktree；不存在 active worktree 时直接返回明确错误，避免隐式创建。

## 维护建议
- 这些方法是 UI 操作入口，不替代 agent tool permission pipeline。
- 如果后续允许 destructive remove，需要继续保持前端确认和后端 dirty guard 双重保护。
