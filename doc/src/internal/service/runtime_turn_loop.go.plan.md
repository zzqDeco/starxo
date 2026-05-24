# runtime_turn_loop.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_turn_loop.go`
- 文档文件: `doc/src/internal/service/runtime_turn_loop.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 用 Eino v0.9 `TurnLoop` 承接 Starxo 顶层 session/turn lifecycle。
- 保留 `ChatModelAgent` 作为内层 ReAct agent；TurnLoop 只负责 push queue、preempt、stop、checkpoint resume 和事件消费编排。

## 3. 关键实现细节
- 每个 session 维护一个可重建 TurnLoop，checkpoint id 为 `runtime-turn:<sessionID>`。
- user turn 在 `GenInput` 中完成 bundle 准备、objective 创建、prompt messages 组装和 default/plan agent 选择。
- user turn / objective 会在 bundle 准备前写入内存历史；即使模型、sandbox 或 config refresh 初始化失败，用户刚提交的请求也不会从 session 状态中丢失。
- starting 状态下的新 user turn 不走 TurnLoop safe-point preempt；先取消当前 startup wait 并等待其收敛，再启动替换 objective。
- running 状态下的新消息通过 `WithPreemptTimeout(AfterToolCalls, 15s)` 进入队列，旧 turn 到安全点后让位给新 objective。
- business interrupt 由 TurnLoop 保存 checkpoint；`ResumeWithAnswer` / `ResumeWithChoice` push resume item 后重建 loop，通过 `GenResume` 恢复原 interrupted objective，resume turn 正常完成后主动删除 stale checkpoint。
- business interrupt 保留 resume 语义，但 pending interrupt 会记录未完成 tool call ids；如果用户用普通新 turn 取代该 interrupt，`ChatService` 会先修复 orphan tool-call history，再丢弃 checkpoint。
- 普通新 user turn 在启动前由 `ChatService.SendMessage` 丢弃旧 checkpoint；这样内存 pending interrupt 丢失或 checkpoint 残留时，新请求会走正常 `GenInput` 而不是无 payload 的 `GenResume`。
- 若 TurnLoop 因 stale checkpoint 进入 `GenResume`，但队列里只有新的 user turn、没有 resume payload，则删除/覆盖旧 checkpoint 并把该 user turn 重新入队到正常 `GenInput` 路径。
- `deleteRuntimeTurnCheckpoint` 优先使用 store Delete；store 不支持 Delete 或 Delete 失败时写入 zero-length tombstone，因为 Eino TurnLoop 会把空 checkpoint 当作不存在。
- `StopGeneration` 使用 `WithImmediate + WithSkipCheckpoint`，用户显式停止不会留下可恢复 checkpoint。
- 显式 stop 或 sandbox loss 清理 pending interrupt 时，同样需要补 synthetic tool result，避免下次模型调用遇到未配对 tool call。
- `sandbox_lost` 和 `user_stop` 属于行政性停止原因；TurnLoop exit reason 不再额外 emit `agent:error`，避免和 sandbox loss 主路径重复报错。
- 被 reset 替换掉的旧 TurnLoop 退出时视为 stale loop，只做后台收敛，不再向 UI 发 `agent:error` / `agent:done`。
- preempted turn 通过 `TurnContext.Preempted` 识别；该路径只收敛 run state，不写 assistant completion、不触发 done callback。
- preempted turn 若已有未完成 tool call，会从历史中移除对应 tool-call group，而不是注入“tool execution failed”合成结果。
- session 删除会主动 finalize active run state，确保 `runDone` / `startDone` 等等待者不会因 TurnLoop 指针被清空而悬挂。

## 4. 维护边界
- subagent 内部同步执行仍使用局部 runner，本文件只迁移顶层 ChatService session loop。
- Starxo 的 permission、ToolSearch discovery、workspace guard、tasks 和 compact sidecar 仍在现有 service/tools 层维护。
