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
- running 状态下的新消息通过 `WithPreemptTimeout(AfterToolCalls, 15s)` 进入队列，旧 turn 到安全点后让位给新 objective。
- business interrupt 由 TurnLoop 保存 checkpoint；`ResumeWithAnswer` / `ResumeWithChoice` push resume item 后重建 loop，通过 `GenResume` 恢复原 interrupted objective，resume turn 正常完成后主动删除 stale checkpoint。
- 普通新 user turn 在启动前由 `ChatService.SendMessage` 丢弃旧 checkpoint；这样内存 pending interrupt 丢失或 checkpoint 残留时，新请求会走正常 `GenInput` 而不是无 payload 的 `GenResume`。
- `StopGeneration` 使用 `WithImmediate + WithSkipCheckpoint`，用户显式停止不会留下可恢复 checkpoint。
- 被 reset 替换掉的旧 TurnLoop 退出时视为 stale loop，只做后台收敛，不再向 UI 发 `agent:error` / `agent:done`。

## 4. 维护边界
- subagent 内部同步执行仍使用局部 runner，本文件只迁移顶层 ChatService session loop。
- Starxo 的 permission、ToolSearch discovery、workspace guard、tasks 和 compact sidecar 仍在现有 service/tools 层维护。
