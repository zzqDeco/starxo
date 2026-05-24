# runtime_agents_build.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_agents_build.go`
- 文档文件: `doc/src/internal/service/runtime_agents_build.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 集中选择顶层 agent 构建路径。
- 默认构建 ChatModelAgent runtime；只有配置开启 `agent.runtime.enableBuiltinDeepTransferFallback` 时才构建旧 deep-transfer agent。

## 3. 关键实现细节
- `buildTopLevelRuntimeAgents(...)` 是无 `ChatService` receiver 的纯构建 helper，为 default/plan 分别传入不同 middleware surface。
- 默认路径调用 `agent.BuildRuntimeAgent(...)`。
- fallback 路径调用 `agent.BuildDeepAgentForMode(...)`，用于调试旧 transfer 行为。

## 4. 维护边界
- 不在调用点分散判断 fallback，避免 default/plan 行为分叉；bundle 组装由 `runtimeBundleBuilder` 调用该 helper。
- 如新增 runtime engine，优先扩展这里的策略分支，并保持 `ChatService.prepareRunnerBundleFromSurface` 简洁。
