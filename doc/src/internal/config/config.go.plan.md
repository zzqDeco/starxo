# config.go 技术说明

## 文件定位
- 源文件: `internal/config/config.go`
- 应用配置结构定义和默认值。

## 核心职责
- `AppConfig` 包含 SSH、Sandbox、LLM、MCP、Agent 配置。
- `SandboxConfig` 替代旧 `DockerConfig`，包含 runtime、rootDir、workDirName、network、memoryLimitMB、commandTimeoutSec、bootstrapPython、pythonPackages。
- `DockerConfig` 仅保留为一版 JSON 兼容字段，不应被运行时逻辑使用。

## 迁移逻辑
- `MigrateLegacyDockerConfig` 将旧 Docker memory/network/workDir 映射到新 sandbox 配置。
- `NormalizeAppConfig` 填充缺省值并清空 `Docker` 字段，确保保存后写出新结构。
