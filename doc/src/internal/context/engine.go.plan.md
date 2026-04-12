# engine.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/context/engine.go`
- 文档文件: `doc/src/internal/context/engine.go.plan.md`
- 文件类型: Go 源码
- 所属模块: `agentctx`

## 2. 核心职责
- 维护对话历史与文件上下文，并在发送给模型前生成消息列表。
- 现在额外支持 synthetic pinned prefix 注入位点，供 deferred MCP announcement 这类“非持久化、非 timeline”提示使用。

## 3. 输入与输出
- 输入来源: 历史消息、文件上下文、系统提示词、可选 pinned prefix
- 输出结果:
  - `PrepareMessages()`
  - `PrepareMessagesWithPinnedPrefix(...)`
  - `ExportMessages()` / `ImportMessages(...)`

## 4. 关键实现细节
- `PrepareMessages()` 保留原有默认行为。
- `PrepareMessagesWithPinnedPrefix(...)` 把消息组装分成三段：
  - system prompt
  - pinned prefix
  - windowed history
- pinned prefix 只参与当前发送给模型的输入，不应被持久化为普通历史消息。

## 5. 依赖关系
- 内部依赖:
  - `ConversationHistory`
  - `FileContext`
  - `windowing.go`
  - `internal/model`
- 外部依赖:
  - `github.com/cloudwego/eino/schema`

## 6. 变更影响面
- 为 deferred MCP announcement、后续 memory summary / policy reminder 预留了统一前缀注入通道。
- `ExportMessages()` 仍只导出真实历史，不包含 pinned prefix。

## 7. 维护建议
- 新增 pinned prefix 内容时，不要把它当作第二状态源；权威状态仍应存放在明确的数据结构中。
