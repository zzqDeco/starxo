# runtime_tasks.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_tasks.go`
- 文档文件: `doc/src/internal/service/runtime_tasks.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 提供 Runtime V2 后台任务管理器和 ChatService Wails API。
- 支撑后台 Bash/subagent 统一进入 task lifecycle 的第一层实现。

## 3. 输入与输出
- 输入来源:
  - `tools.RuntimeTaskManager` 调用
  - Wails 绑定方法：`ListRuntimeTasks`、`ReadRuntimeTaskOutput`、`StopRuntimeTask`、`ApproveToolPermission`、`DenyToolPermission`
- 输出结果:
  - `tools.RuntimeTaskSnapshot`
  - `tools.RuntimeTaskOutput`
  - Wails events：`runtime:task_started`、`runtime:task_completed`、`runtime:task_stopped`、`runtime:permission_resolved`

## 4. 关键实现细节
- task 状态：`running`、`completed`、`failed`、`killed`
- task output 存储在 `~/.starxo/sessions/<session>/runtime-tasks/<task>.output`
- 大型 foreground tool result 可存储在 `~/.starxo/sessions/<session>/tool-results/<id>.txt`
- `ReadTaskOutput` 支持 byte offset/limit，便于前端增量读取。
- `StopTask` 对 running task 调用 cancel，并保留明确的 `killed` 状态。
- permission API 当前是基础事件层：校验 requestID 后发出 resolved event；完整审批队列后续补齐。
- `wailsEmit` 对 nil context 短路，保证 Go 单测不会触发 Wails runtime fatal。

## 5. 依赖关系
- 内部依赖: `internal/tools`
- 外部依赖: `github.com/wailsapp/wails/v2/pkg/runtime`

## 6. 变更影响面
- `ChatService` 增加 runtime task manager 字段并暴露新的 Wails 方法。
- Runtime V2 `Bash` background mode 依赖本文件。

## 7. 维护建议
- 增加后台 subagent 或 worktree task 时复用同一 task snapshot/result contract。
- permission queue 完整实现后仍应保留当前 Wails API 名称，避免前端绑定断裂。
