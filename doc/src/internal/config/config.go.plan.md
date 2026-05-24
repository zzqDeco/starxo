# config.go 技术说明

## 文件定位
- 源文件: `internal/config/config.go`
- 应用配置结构定义和默认值。

## 核心职责
- `AppConfig` 包含 SSH、Sandbox、LLM、MCP、Agent 配置。
- `SandboxConfig` 替代旧 `DockerConfig`，包含 runtime、rootDir、workDirName、network、memoryLimitMB、commandTimeoutSec、bootstrapPython、pythonPackages。
- `AgentConfig.WebSearch` 定义 Runtime V2 `WebSearch` provider 配置，默认启用 `duckduckgo`。
- `WebSearchProviderConfig` 支持 `type=duckduckgo|http|tinyfish`；TinyFish provider 包含 `apiKeyEnv`、`location`、`language`、`page` 等官方 Search API 适配字段。
- `AgentConfig.LSP` 定义 Runtime V2 常驻 LSP 配置，包含启用开关、请求超时、最大结果字节数、自定义 server command 和扩展名映射。
- `AgentConfig.Runtime` 定义 Eino v0.9 runtime 行为：`engine`、`toolSearchMode`、`agenticProtocol`、旧 deep-transfer fallback 开关和动态 subagent definitions。
- `SubagentDefinitionConfig` 支持配置 `name`、`description`、`instruction`、`allowedTools`、`defaultIsolation`、`backgroundAllowed`，运行时由 `Agent` tool registry 使用。
- `DockerConfig` 仅保留为一版 JSON 兼容字段，不应被运行时逻辑使用。

## 迁移逻辑
- `MigrateLegacyDockerConfig` 将旧 Docker memory/network/workDir 映射到新 sandbox 配置。
- `NormalizeAppConfig` 填充缺省值并清空 `Docker` 字段，确保保存后写出新结构。
- `WebSearchConfig.Enabled` 缺失时默认为 true；`DefaultProvider` 缺失时默认为 `duckduckgo`。
- `RuntimeLSPConfig.Enabled` 缺失时默认为 true；`RequestTimeoutMS` 默认 15000，`MaxResultBytes` 默认 16384。
- `AgentRuntimeConfig.Engine` 默认 `eino_v09`，`ToolSearchMode` 默认 `client`，`AgenticProtocol` 默认 `off`。
- 缺失 subagent definitions 时填充 `general`、`code_writer`、`code_executor`、`file_manager`、`reviewer`。
