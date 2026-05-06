# runtime_workspaces.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_workspaces.go`
- 文档文件: `doc/src/internal/service/runtime_workspaces.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 管理 Runtime V2 session-scoped active workspace。
- 提供 `EnterWorktree` / `ExitWorktree` 的后端实现，让文件、搜索、编辑和 Bash 工具可在隔离 worktree 中运行。
- 为动态 `Agent` 工具提供 context-scoped isolated worktree，不污染 session-scoped active workspace。

## 3. 输入与输出
- 输入来源: Runtime V2 worktree tools、当前 session context、sandbox operator、默认 workspace path。
- 输出结果: `tools.WorktreeOutput`，包含 action、当前 workspace、worktree path/branch 和人类可读 message。

## 4. 关键实现细节
- 每个 session 同时只允许一个 active worktree。
- worktree 创建位置固定在 `<workspace>/.starxo/worktrees/<slug>`。
- 分支名固定为 `starxo/<slug>`，slug 会做字符过滤和长度限制。
- `CreateIsolatedWorktree` 只创建 worktree，不写入 session active state；子 agent 通过 context workspace override 路由到该 worktree。
- `CurrentWorkspace` 优先读取 context-scoped override，其次读取 session-scoped active worktree，最后回退默认 workspace。
- `ExitWorktree(action=keep)` 只恢复原 workspace，不删除 worktree。
- `ExitWorktree(action=remove)` 会先检查 `git status --porcelain`；dirty worktree 必须显式 `discard_changes=true` 才允许删除。
- `CurrentWorkspace` 被 Runtime V2 core/deferred tools 调用，统一决定当前 session 的实际执行目录。
- `CompactSnapshot(...)` / `RestoreCompactSnapshot(...)` 让 active worktree routing 可随 Runtime context compact 持久化和恢复。

## 5. 依赖关系
- 内部依赖: `internal/model`、`internal/tools`
- 外部依赖: `github.com/cloudwego/eino-ext/components/tool/commandline`

## 6. 变更影响面
- Runtime V2 的 Read/Write/Edit/Grep/Glob/Bash 可以透明跟随 active worktree。
- `Agent(isolation=worktree)` 可以和同 session 的其他工具并发执行，其他工具仍使用原 session workspace。
- 需要远端 workspace 是 git repository；否则 `git worktree add` 会失败。
- Runtime context compact restore 后会恢复 session 的 active worktree path；不重新创建远端 worktree。

## 7. 维护建议
- 不要把 active worktree 存到全局 active session；必须继续从 tool context 读取 sessionID。
- 删除 worktree 时保持 dirty guard，避免误删 agent 生成但尚未审阅的修改。
