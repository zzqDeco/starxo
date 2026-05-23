# WorktreeReviewPanel.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/files/WorktreeReviewPanel.vue`
- 所属模块: frontend/src/components/files

## 核心职责
- 在工作区抽屉内展示当前会话的 Runtime V2 active worktree。
- 提供 worktree review、copy review、merge、exit-and-keep 操作。

## 关键实现细节
- 通过 ChatService Wails 绑定调用 `GetRuntimeWorktreeState`、`ReviewRuntimeWorktree`、`MergeRuntimeWorktree`、`ExitRuntimeWorktree`。
- merge 前使用危险确认弹窗，提示目标 branch 和 original workspace。
- merge 失败时保留错误信息，提示 active worktree 仍被保留，用户可再次 review 或 exit keep。
- merge 返回 `conflicted=true` 时展示 warning、conflict files、后端 recovery hint 和 merge output；不把冲突当作成功合并，也不清空 review。
- 监听 `runtime:worktree_changed` 事件，当前 session 命中时自动刷新状态。

## 维护建议
- 不在前端构造 git 命令；所有 worktree 操作都必须走后端受控方法。
- 如果增加 remove/discard 操作，必须继续使用确认弹窗并依赖后端 dirty guard。
