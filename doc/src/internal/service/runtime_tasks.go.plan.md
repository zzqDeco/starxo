# runtime_tasks.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_tasks.go`
- 文档文件: `doc/src/internal/service/runtime_tasks.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 提供 Runtime V2 后台任务管理器和 ChatService Wails API。
- 支撑后台 Bash 和后台 `Agent` 统一进入 task lifecycle。
- 提供 CC-style persistent task graph，支撑 `TaskCreate` / `TaskGet` / `TaskUpdate` / `TaskList`。

## 3. 输入与输出
- 输入来源:
  - `tools.RuntimeTaskManager` 调用
  - Wails 绑定方法：`ListRuntimeTasks`、`ReadRuntimeTaskOutput`、`StopRuntimeTask`、`ApproveToolPermission`、`DenyToolPermission`
- 输出结果:
  - `tools.RuntimeTaskSnapshot`
  - `tools.RuntimeTaskOutput`
  - `tools.RuntimeTaskItem`
- task snapshot 包含 `durationMs` 和 `outputSize`，供前端 Runtime Tasks 面板展示运行耗时和输出规模。
- Wails events：`runtime:task_started`、`runtime:task_completed`、`runtime:task_stopped`
- Task graph event：`runtime:task_graph_changed`

## 4. 关键实现细节
- task 状态：`running`、`completed`、`failed`、`killed`
- task graph item 状态：`todo`、`in_progress`、`blocked`、`completed`、`canceled`
- task output 存储在 `~/.starxo/sessions/<session>/runtime-tasks/<task>.output`
- 大型 foreground tool result 可存储在 `~/.starxo/sessions/<session>/tool-results/<id>.txt`
- `StartShellTask` 负责后台 shell 命令，task type 为 `shell`。
- `StartAgentTask` 负责后台动态子 agent，task type 为 `agent`，并继承 session context 以支持 permission、worktree 等 per-session 能力。
- `CreateTaskItem` / `GetTaskItem` / `UpdateTaskItem` / `ListTaskItems` 管理 session-scoped task graph records，字段包含 title、description、owner、priority、depends_on 和 timestamps。
- closed task graph items 默认从 `TaskList` 隐藏，可用 `include_closed` 或显式 status filter 查看。
- `ReadTaskOutput` 支持 byte offset/limit，便于前端增量读取。
- `StopTask` 对 running task 调用 cancel，并保留明确的 `killed` 状态。
- `CompactSnapshots(sessionID)` 将 task snapshot 转成 `model.RuntimeTaskCompact`，供 Runtime context compact 持久化。
- `CompactTaskItems(sessionID)` 将 task graph 转成 `model.RuntimeTaskItemCompact`，供 Runtime context compact 持久化。
- `RestoreCompactTasks(...)` 从 compact state 恢复 task 可见性；原本 `running` 的 task 在 reload 后标记为 `failed`，因为进程已不再附着。
- `RestoreCompactTaskItems(...)` 从 compact state 恢复 task graph records，供 reload 后继续追踪 workflow/delegation 状态。
- permission API 方法名仍保留在本文件，但实际队列和 grant 逻辑已下沉到 `runtime_permissions.go`。
- `GetRuntimeLSPStatus(sessionID)` 也在本文件暴露为 Wails API，读取 runtime LSP manager 当前状态。
- `wailsEmit` 对 nil / 非 Wails runtime context 短路，保证 Go 单测或纯 service 调用不会触发 Wails runtime fatal。

## 5. 依赖关系
- 内部依赖: `internal/model`、`internal/tools`
- 外部依赖: `github.com/wailsapp/wails/v2/pkg/runtime`

## 6. 变更影响面
- `ChatService` 增加 runtime task manager 字段并暴露新的 Wails 方法。
- Runtime V2 `Bash` background mode 和动态 `Agent(background=true)` 依赖本文件。
- Runtime context compact 依赖本文件保存和恢复后台任务 output pointer。
- Runtime context compact 也依赖本文件保存和恢复 persistent task graph。

## 7. 维护建议
- 增加新的后台 tool 类型时复用同一 task snapshot/result contract。
- permission queue 完整实现后仍应保留当前 Wails API 名称，避免前端绑定断裂。
