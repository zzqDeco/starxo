# runtime_workspaces_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_workspaces_test.go`
- 文档文件: `doc/src/internal/service/runtime_workspaces_test.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 覆盖 Runtime V2 worktree manager 的 session workspace 切换、dirty 删除保护、diff review 和 merge restore 行为。

## 3. 输入与输出
- 输入来源: fake commandline operator、带 sessionID 的 context。
- 输出结果: worktree state、执行命令记录、错误断言。

## 4. 关键测试覆盖
- `EnterWorktree` 创建 session-scoped worktree 后，`CurrentWorkspace` 返回 worktree path。
- worktree 创建命令会在 `git worktree add` 前更新 Git `info/exclude`，并且不触碰 tracked `.gitignore`。
- `ExitWorktree(action=keep)` 恢复默认 workspace，并保留 worktree。
- dirty worktree 在未设置 `discard_changes=true` 时拒绝 remove，并保持 active worktree 状态。
- `CreateIsolatedWorktree` 不切换 session workspace；只有带 context override 的子 agent context 会路由到 isolated worktree。
- `DiffWorktree` 返回 status/stat/patch 和单独的 untracked patch，使用 parent/worktree 的 merge-base 作为 review base，stat 覆盖 untracked 文件，并按 `max_bytes` 截断 patch。
- `MergeWorktree` 会提交 worktree 修改、校验 active branch、按 resolved worktree HEAD merge 到 parent、可选 remove worktree，并恢复默认 workspace。
- worktree 分支被用户或 agent 切换时，`MergeWorktree` 会拒绝 merge 并保持 active worktree 状态。
- failed merge 会触发 `git merge --abort`，同时保持 active worktree 状态供后续修复。
- Git conflict merge 会返回结构化 `merge_conflict` output，包含去重后的 conflict files、merge output 和 recovery hint，并保持 active worktree。
- Git conflict 后如果 `git merge --abort` 失败，会返回错误而不是误报结构化恢复成功。

## 5. 维护建议
- 新增 worktree action 时同步补状态转换和命令构造测试。
