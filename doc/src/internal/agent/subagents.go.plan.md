# subagents.go 技术说明

## 1. 文件定位
- 源文件: `internal/agent/subagents.go`
- 所属模块: agent

## 2. 核心职责
- 定义 Claude Code-style runtime subagent metadata。
- 提供默认 subagent registry：`general`、`code_writer`、`code_executor`、`file_manager`、`reviewer`。
- 为 Runtime `Agent` tool 提供类型规范化、allowed tools、默认 isolation 和提示词生成。

## 3. 关键实现细节
- `SubagentDefinition` 是运行时定义，不直接暴露给模型。
- `SubagentRegistry.Normalize(...)` 允许 service 层使用配置生成自定义 registry。
- 自定义 registry 不会隐式注入无限制 `general`；空 `subagent_type` 会落到 registry 的默认 definition，优先使用配置中的 `general`，否则使用第一条有效配置。
- `RuntimeSubagentPrompt(...)` 会把 subagent 类型、workspace 和 isolation 写入子 agent instruction。
- `AllowedTools` 为空表示不做工具白名单限制；非空时由 service 层过滤 runtime core tools。

## 4. 维护建议
- 新增内置 subagent 时同步更新默认配置、README 和前端类型。
- 不要在 registry 中执行权限判断；权限仍由 Starxo permission wrapper 统一处理。
