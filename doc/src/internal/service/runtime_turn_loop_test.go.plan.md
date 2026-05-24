# runtime_turn_loop_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_turn_loop_test.go`
- 文档文件: `doc/src/internal/service/runtime_turn_loop_test.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 覆盖 TurnLoop 顶层 session lifecycle 的关键边界。
- 重点保护 resume 使用 interrupted objective，而不是 session 当前 mutable objective。

## 3. 关键测试覆盖
- `GenResume` 从 TurnLoop interrupted item 恢复原 objective，并生成对应 interrupt id 的 resume params。
- TurnLoop checkpoint id 保持 session scoped：`runtime-turn:<sessionID>`。
- `GenInput` 在 bundle 准备失败前已经持久化 user message 和 active objective。
- startup 阶段收到替换 user turn 时会先取消旧 startup，最终只运行替换 objective。
- 普通 `SendMessage` 在非 preempt 路径会先删除旧 runtime checkpoint，避免 stale checkpoint 导致新消息卡在无 resume payload 的恢复路径。
- preempted turn 不会留下 unresolved tool-call group 或合成 tool failure。
- 删除 session 会停止 idle/active persistent TurnLoop、关闭 active run done channel 并移除内存 session。

## 4. 维护建议
- 后续新增 preempt/abort 场景时优先在本测试文件补 focused unit test，再做端到端手工验证。
