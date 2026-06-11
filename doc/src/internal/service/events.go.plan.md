# events.go 技术说明

## 文件定位
- 源文件: `internal/service/events.go`
- 定义前后端事件和 DTO。

## 核心职责
- `SandboxStatusDTO` 新增 runtime 中性字段：`runtimeAvailable`、`sandboxActive`、`activeSandboxID`、`activeSandboxName`。
- 旧 `dockerRunning`、`dockerAvailable`、`containerID`、`activeContainerID` 字段保留一版兼容前端过渡。
- `WorkspaceInfoDTO` 和 `WorkspaceCleanupResultDTO` 支撑工作区抽屉元信息和 tmp 清理结果；`activeContainerID` 使用 registry ID 供前端 lifecycle 事件去重。
- `WorkspaceChangedEvent` 用于通知前端当前 workspace 发生变化，携带可选 `sessionId`、`containerID`、`path`、`source`、`action` 和 `createdAt`，供全局 dirty store 与 WorkspacePanel 做过滤、debounce 和短延迟重试刷新。
- `TerminalCommandResult` 返回用户提交的 sandbox terminal 命令、stdout/stderr 和 exitCode。
- `SandboxProgressEvent` 继续用于 SSH/runtime 初始化和 sandbox 创建进度。
