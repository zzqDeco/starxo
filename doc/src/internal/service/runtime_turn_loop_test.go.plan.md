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
- `sandbox_lost` / `user_stop` 这类行政性 stop cause 会抑制 TurnLoop exit error，其他错误仍会上报。
- TurnLoop checkpoint id 保持 session scoped：`runtime-turn:<sessionID>`。
- `GenInput` 在 bundle 准备失败前已经持久化 user message 和 active objective。
- startup 阶段收到替换 user turn 时会先取消旧 startup，最终只运行替换 objective。
- 普通 `SendMessage` 在非 preempt 路径会先删除旧 runtime checkpoint，避免 stale checkpoint 导致新消息卡在无 resume payload 的恢复路径。
- checkpoint store 不支持 Delete 时，runtime checkpoint 删除会落到 zero-length tombstone，确保 Eino TurnLoop 后续把旧 checkpoint 视为不存在。
- stale resume checkpoint 遇到新的 user turn 时，会恢复为正常 user turn 执行，而不是因为缺少 resume payload 失败。
- business interrupt 会记录未完成 tool call ids；新 standalone turn 取代 interrupt 时会修复 orphan tool history 并清理 pending interrupt。
- idle pending interrupt 被 StopGeneration 清理时会补 synthetic tool result，避免后续 provider 请求失败。
- preempted turn 不会留下 unresolved tool-call group 或合成 tool failure。
- preempted turn 的 cleanup 只清理当前 turn 未完成的 tool-call history，不会删除旧历史里同名但已完成的 tool call/result。
- 删除 session 会停止 idle/active persistent TurnLoop、关闭 active run done channel 并移除内存 session。

## 4. 维护建议
- 后续新增 preempt/abort 场景时优先在本测试文件补 focused unit test，再做端到端手工验证。
