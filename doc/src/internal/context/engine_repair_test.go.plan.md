# engine_repair_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/context/engine_repair_test.go`
- 所属模块: `agentctx`

## 2. 核心职责
- 覆盖 context history 中 assistant `tool_call` 与 tool result 的 provider pairing 修复逻辑。
- 防止 interrupted `ask_user` / `ask_choice` 旧历史再次触发 OpenAI-compatible provider 的 `No tool output found` 错误。

## 3. 覆盖场景
- 单个 orphan tool call 会在 assistant tool-call group 后补 synthetic tool result。
- 旧实现追加到历史末尾的非相邻 tool result 会被移动到正确位置。
- 多 tool call group 只补缺失的 tool result，并保持 tool call 顺序。
- `PrepareMessagesWithCompactFrom(...)` 对 standalone history slice 做防御性修复，不让 stray tool message 进入 prompt。

## 4. 维护建议
- 新增 context windowing 或 compact 逻辑时，应继续保持 `assertProviderToolPairing` 这类约束测试。
- 不要把 synthetic repair message 当成用户可见 timeline 事件；它只服务 provider 协议兼容。
