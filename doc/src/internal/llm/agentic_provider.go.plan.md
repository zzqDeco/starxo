# agentic_provider.go 技术说明

## 1. 文件定位
- 源文件: `internal/llm/agentic_provider.go`
- 所属模块: llm

## 2. 核心职责
- 提供 Eino v0.9 agentic model beta adapter。
- 支持 `agentic_openai`、`agentic_ark` 和 `auto` 协议选择。
- 保持默认 Message runtime 不受影响。

## 3. 关键实现细节
- `NewAgenticModel(...)` 只在 `agent.runtime.agenticProtocol != off` 时由 service 层探测调用。
- OpenAI/DeepSeek 使用 `agenticopenai` adapter；Ark 使用 `agenticark` adapter。
- HTTP headers 复用现有 provider helper，避免和普通 ChatModel 配置割裂。

## 4. 维护建议
- 不要把 ChatService 默认历史、Wails IPC 或 MCP 流程直接切到 AgenticMessage。
- agentic 初始化失败必须允许回退，避免 beta provider 影响默认运行。
