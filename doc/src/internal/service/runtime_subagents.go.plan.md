# runtime_subagents.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_subagents.go`
- 所属模块: service

## 2. 核心职责
- 将 `config.SubagentDefinitionConfig` 转换成 `agent.SubagentRegistry`。
- 让 Runtime `Agent` tool 使用配置定义，而不是固定写死 subagent 类型。

## 3. 关键实现细节
- 空配置回退到 `agent.DefaultSubagentRegistry()`。
- `backgroundAllowed` 缺失时默认为 true。
- `allowedTools`、`defaultIsolation`、`instruction` 原样进入 runtime registry，由 Agent tool 执行时校验。

## 4. 维护建议
- 新配置字段应先进入 `internal/config`，再在本文件转换到 runtime 定义。
- 不在本文件做权限授权；这里只构造能力白名单，执行前仍经过 permission queue。
