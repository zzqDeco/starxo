# config_test.go 技术说明

## 文件定位
- 源文件: `internal/config/config_test.go`
- 配置默认值、JSON 行为和迁移逻辑测试。

## 核心覆盖
- 默认 SSH / Sandbox / LLM / Agent / MCP 配置值正确。
- AppConfig JSON round-trip 保留 sandbox 配置。
- 空 SSH 密钥和 LLM headers 仍按预期省略。
- 旧 Docker 配置可迁移到新 SandboxConfig。
