# events.go 技术说明

## 文件定位
- 源文件: `internal/service/events.go`
- 定义前后端事件和 DTO。

## 核心职责
- `SandboxStatusDTO` 新增 runtime 中性字段：`runtimeAvailable`、`sandboxActive`、`activeSandboxID`、`activeSandboxName`。
- 旧 `dockerRunning`、`dockerAvailable`、`containerID`、`activeContainerID` 字段保留一版兼容前端过渡。
- `SandboxProgressEvent` 继续用于 SSH/runtime 初始化和 sandbox 创建进度。
