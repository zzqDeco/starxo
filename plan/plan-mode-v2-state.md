# Plan Mode V2 State

## Summary
- 这条实现只落 plan-mode v2 的 persisted state 底座，不改当前用户可见 runner 行为。
- 目标是把 `Mode`、`PlanDocument`、`PendingPlanApproval`、`PendingPlanAttachment` 变成 `SessionData` / `SessionSnapshot` / `session_data.json` 的一等字段。

## Decisions
- `SessionData.Version` 升到 `4`。
- normalize helper 固定放在 `internal/model`，storage / service / tests 共用同一套 downgrade/default 规则。
- `LoadSessionData()` 保持 `nil-on-empty`；空 snapshot 规范化只放在 `ExportSessionSnapshot(...)`。
- `SetMode()` / `ClearHistory()` 只触发 async best-effort save，不切 blocking durability。
- `ClearHistory()` 清 plan state，但不切 mode，也不处理 workspace `plan.md` artifact。

## Acceptance
- old session 缺字段或有脏值时不会恢复失败。
- startup / session switch / direct restore 都能看到 normalized v4 in-memory view。
- 前端现有 `GetMode()` / `session:switched.mode` 恢复链路能看到 persisted mode。
- 后续 runner/UI PR 可以直接在这套 persisted state 上接 approval flow。
