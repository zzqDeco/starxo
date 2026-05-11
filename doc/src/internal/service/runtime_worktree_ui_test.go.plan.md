# runtime_worktree_ui_test.go 技术说明

## 文件定位
- 源文件: `internal/service/runtime_worktree_ui_test.go`
- 所属模块: service tests

## 覆盖范围
- `GetRuntimeWorktreeState` 能从 active session 返回当前 runtime worktree 状态。
- `FileService.workspacePath` 会跟随当前 active session 的 runtime worktree。
- `timelineToolResultLimit` 为 `WorktreeDiff` 保留更大的 timeline payload，同时保持普通工具默认截断。
- 大型 `WorktreeDiff` timeline payload 会保持可解析 JSON，并在字段级标记截断。
- 大型 `Edit` / `Write` timeline payload 会保持可解析 JSON，并对 patch 字段标记截断。

## 维护建议
- 新增 worktree UI API 时优先补无 sandbox 依赖的状态/路由单测。
