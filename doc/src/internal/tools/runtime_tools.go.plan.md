# runtime_tools.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/runtime_tools.go`
- 文档文件: `doc/src/internal/tools/runtime_tools.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 定义 Runtime V2 core tool 的 DTO、catalog entry 和执行逻辑。
- 把文件、搜索、编辑、shell、后台任务工具纳入统一 catalog/permission/ToolSearch 体系。

## 3. 输入与输出
- 输入来源: remote sandbox `commandline.Operator`、workspace path、runtime task manager、runtime workspace manager
- 输出结果: catalog entries 和结构化 tool result

## 4. 关键实现细节
- always-load 工具：
  - `Bash`
  - `Read`
  - `Write`
  - `Edit`
  - `Glob`
  - `Grep`
  - `TaskOutput`
  - `TaskStop`
  - `ExitPlanMode`
  - `Agent`
- deferred core 工具：
  - `EnterWorktree`
  - `ExitWorktree`
  - `WorktreeDiff`
  - `WorktreeMerge`
- legacy aliases：
  - `shell_execute`
  - `read_file`
  - `write_file`
  - `list_files`
  - `str_replace_editor`
- `Read`/`Glob`/`Grep`/`TaskOutput`/`ExitPlanMode` 标记为 read-only trusted，plan mode 可见。
- `WorktreeDiff` 标记为 read-only trusted，可用于审阅 active worktree 修改。
- `Bash`/`Write`/`Edit`/`TaskStop`/`Agent`/`EnterWorktree`/`ExitWorktree`/`WorktreeMerge` 是 writable/destructive surface，plan mode 下不加载或需先退出计划模式。
- 所有 workspace path 都走 guard：拒绝空 workspace、`..` traversal 和 active workspace 外 absolute path；worktree mode 下以当前 active worktree 作为唯一边界。
- `Read`/`Write`/`Edit`/`Glob`/`Grep`/`Bash` 会通过 `RuntimeWorkspaceManager.CurrentWorkspace` 解析 session 当前 workspace，因此可透明运行在 active worktree 中。
- `Bash` 支持 foreground/background。background 通过 task manager 持久化输出。
- `Read` 支持 line offset/limit。
- `Write` 返回 created/bytes/linesAdded/linesRemoved 和 bounded patch，方便前端结构化审阅写入结果；覆盖旧文件时优先使用 operator 的 bounded preview，且为新增内容保留 patch 预算，避免为生成 diff 读取完整大文件或只显示删除内容。
- `Edit` 使用精确字符串替换并返回 bounded patch 摘要、行数变化和是否替换成功；大段替换时和 `Write` 一样为 replacement 内容保留 patch 预算。
- bounded patch 对超长单行保留可容纳的行前缀，再追加 truncation marker，避免只显示“已截断”而没有实际变更内容。
- `Glob` 通过远端 `find` 稳定排序；`Grep` 通过远端 `rg` 并支持 `content/count/files_with_matches`。

## 5. 依赖关系
- 内部依赖: `catalog.go`、`permissions.go`
- 外部依赖: `github.com/cloudwego/eino/components/tool`

## 6. 变更影响面
- 顶层 agent prompt 现在可以直接看到 Runtime V2 core tools。
- LSP/Skill/Web/Notebook deferred tools 已通过相同 metadata contract 接入。
- Worktree tools 会改变当前 session 的 active workspace，影响后续 runtime tool 路径解析。
- Worktree review/merge tools 让 active worktree 具备从“隔离执行”到“审阅并合并回原 workspace”的闭环。

## 7. 维护建议
- 新增 runtime tool 时先定义 metadata、permission 和 read-only 语义，再接入实现。
- 任何文件类工具都必须复用 workspace guard，不要在工具内部各写一套 path 清洗。
- 任何会切换 workspace 的能力都必须基于 context sessionID 管理状态，不能依赖全局 active session。
