# runtime_workspaces.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_workspaces.go`
- 文档文件: `doc/src/internal/service/runtime_workspaces.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 管理 Runtime V2 session-scoped active workspace。
- 提供 `EnterWorktree` / `ExitWorktree` 的后端实现，让文件、搜索、编辑和 Bash 工具可在隔离 worktree 中运行。
- 提供 `DiffWorktree` / `MergeWorktree`，让 agent 可以先审阅 active worktree 修改，再把修改合并回原 workspace。
- 为动态 `Agent` 工具提供 context-scoped isolated worktree，不污染 session-scoped active workspace。

## 3. 输入与输出
- 输入来源: Runtime V2 worktree tools、当前 session context、sandbox operator、默认 workspace path。
- 输出结果: `tools.WorktreeOutput`、`tools.WorktreeDiffOutput`、`tools.WorktreeMergeOutput`，包含 action、当前 workspace、worktree path/branch、diff/merge metadata 和人类可读 message。

## 4. 关键实现细节
- 每个 session 同时只允许一个 active worktree。
- worktree 创建位置固定在 `<workspace>/.starxo/worktrees/<slug>`。
- 创建 worktree 前会把 `.starxo/` 追加到本地 Git `info/exclude`，避免改动 tracked `.gitignore`。
- 分支名固定为 `starxo/<slug>`，slug 会做字符过滤和长度限制。
- `CreateIsolatedWorktree` 只创建 worktree，不写入 session active state；子 agent 通过 context workspace override 路由到该 worktree。
- `CurrentWorkspace` 优先读取 context-scoped override，其次读取 session-scoped active worktree，最后回退默认 workspace。
- `ExitWorktree(action=keep)` 只恢复原 workspace，不删除 worktree。
- `ExitWorktree(action=remove)` 会先检查 `git status --porcelain`；dirty worktree 必须显式 `discard_changes=true` 才允许删除。
- `DiffWorktree` 先解析原 workspace 当前 `HEAD` 与 worktree `HEAD` 的 `merge-base` 作为 review base，再在 active worktree 工作树上执行 diff，因此 parent workspace 后续新增 commit 不会被误显示为 worktree 删除；diff stat 会合并 untracked 文件摘要，`include_patch=true` 会先捕获 untracked 文件的 patch-like 内容并单独返回 `untrackedDiff`，避免 merge 前审阅缺口。
- patch 采集命令在远端通过 `head -c maxBytes+1` 先截断，再由 Go 侧设置 `truncated` / `untrackedTruncated`，避免大文件把完整 patch 拉回本地。
- `MergeWorktree` 先拒绝 dirty parent workspace，再把 worktree 的未提交修改提交到 `starxo/<slug>` 分支，解析当前 worktree `HEAD`，随后在原 workspace 执行 `git merge --no-ff <worktree-head-sha>`。
- merge prepare 阶段会在提交前后校验 active worktree 当前分支仍等于记录的 `starxo/<slug>`；如果用户或 agent 在 worktree 内切到其他分支，会拒绝 merge，避免把修改提交到未被合并的分支。
- `MergeWorktree` 将 prepare 和 merge 分成两次远端调用；如果 merge 调用返回非零 exit code 且像 Git conflict，会先读取 `diff --name-only --diff-filter=U`，再对原 workspace 执行 `git merge --abort`。
- 只有确认 abort 成功后才返回 `action=merge_conflict` 的结构化恢复结果并保留 active worktree；abort 失败会作为错误冒泡，避免误报 parent workspace 已恢复。
- 非冲突 merge 失败仍会执行 `git merge --abort` 后返回错误，避免 parent workspace 留在中间状态。
- `MergeWorktree(remove_worktree=true)` 在 merge 成功后移除 worktree 并尝试删除本地分支；无论是否移除，成功后都会恢复 session 的原 workspace。
- `CurrentWorkspace` 被 Runtime V2 core/deferred tools 调用，统一决定当前 session 的实际执行目录。
- `CompactSnapshot(...)` / `RestoreCompactSnapshot(...)` 让 active worktree routing 可随 Runtime context compact 持久化和恢复。

## 5. 依赖关系
- 内部依赖: `internal/model`、`internal/tools`
- 外部依赖: `github.com/cloudwego/eino-ext/components/tool/commandline`

## 6. 变更影响面
- Runtime V2 的 Read/Write/Edit/Grep/Glob/Bash 可以透明跟随 active worktree。
- `WorktreeDiff` 是 read-only review surface，可在 plan/read-only 场景中用于审查改动；`WorktreeMerge` 是 writable surface，必须走权限队列。
- `Agent(isolation=worktree)` 可以和同 session 的其他工具并发执行，其他工具仍使用原 session workspace。
- 需要远端 workspace 是 git repository；否则 `git worktree add` 会失败。
- Runtime context compact restore 后会恢复 session 的 active worktree path；不重新创建远端 worktree。

## 7. 维护建议
- 不要把 active worktree 存到全局 active session；必须继续从 tool context 读取 sessionID。
- 删除 worktree 时保持 dirty guard，避免误删 agent 生成但尚未审阅的修改。
- merge 失败或冲突时不要清理 active state；让用户或 agent 能继续查看 worktree、处理冲突并重试。
