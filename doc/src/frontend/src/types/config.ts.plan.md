# config.ts 技术说明

## 文件定位
- 源文件: `frontend/src/types/config.ts`
- 前端设置与状态类型。

## 核心职责
- `SandboxConfig` 定义轻量沙箱运行时配置。
- `AppSettings` 使用 `sandbox`，旧 `docker` 为可选兼容字段。
- `SandboxStatus` 新增 runtime 中性字段，同时保留旧 Docker/Container 字段兼容事件消费。
- `SandboxDiagnosticsResult`、`SandboxDiagnosticCheck`、`SandboxFixSuggestion` 对齐后端诊断 DTO。
- `WorkspaceInfo` 和 `WorkspaceCleanupResult` 支撑工作区抽屉元信息与 tmp 清理结果；`activeContainerID` 是 registry ID，用于前端区分 active sandbox lifecycle 事件。
- `AgentRuntimeConfig` 对齐后端 Eino v0.9 runtime 配置，包含 `engine`、`toolSearchMode`、`agenticProtocol`、旧 transfer fallback 开关和 `SubagentDefinitionConfig[]`。
- `SubagentDefinitionConfig` 支持动态 Agent tool 子 agent 定义：名称、描述、指令、allowed tools、默认 isolation、后台执行策略。
