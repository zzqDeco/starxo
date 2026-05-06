# config.ts 技术说明

## 文件定位
- 源文件: `frontend/src/types/config.ts`
- 前端设置与状态类型。

## 核心职责
- `SandboxConfig` 定义轻量沙箱运行时配置。
- `AppSettings` 使用 `sandbox`，旧 `docker` 为可选兼容字段。
- `SandboxStatus` 新增 runtime 中性字段，同时保留旧 Docker/Container 字段兼容事件消费。
- `SandboxDiagnosticsResult`、`SandboxDiagnosticCheck`、`SandboxFixSuggestion` 对齐后端诊断 DTO。
- `WorkspaceInfo` 和 `WorkspaceCleanupResult` 支撑工作区抽屉元信息与 tmp 清理结果。
