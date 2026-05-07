# runtime_workspaces_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_workspaces_test.go`
- 文档文件: `doc/src/internal/service/runtime_workspaces_test.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 覆盖 Runtime V2 worktree manager 的 session workspace 切换和 dirty 删除保护。

## 3. 输入与输出
- 输入来源: fake commandline operator、带 sessionID 的 context。
- 输出结果: worktree state、执行命令记录、错误断言。

## 4. 关键测试覆盖
- `EnterWorktree` 创建 session-scoped worktree 后，`CurrentWorkspace` 返回 worktree path。
- worktree 创建命令会在 `git worktree add` 前更新 Git `info/exclude`，并且不触碰 tracked `.gitignore`。
- `ExitWorktree(action=keep)` 恢复默认 workspace，并保留 worktree。
- dirty worktree 在未设置 `discard_changes=true` 时拒绝 remove，并保持 active worktree 状态。
- `CreateIsolatedWorktree` 不切换 session workspace；只有带 context override 的子 agent context 会路由到 isolated worktree。

## 5. 维护建议
- 新增 worktree action 时同步补状态转换和命令构造测试。
