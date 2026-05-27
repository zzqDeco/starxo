# plan_mode_state_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/plan_mode_state_test.go`
- 文档文件: `doc/src/internal/service/plan_mode_state_test.go.plan.md`
- 所属模块: service tests

## 2. 核心职责
- 覆盖 plan mode v2 的 session persistence、restore、mode 切换和 clear-history 行为。

## 3. 关键测试覆盖
- session snapshot 会携带 plan document、pending approval、pending attachment 和 mode。
- mode restore / no-op save / blocking save 行为保持稳定。
- `ClearHistory` 清空 plan state 和消息，但保留当前 mode。
- `ClearHistory` 同时清理当前 session 的 task graph items，避免旧 `TaskCreate` 状态被 compact/save 后重新注入。
- task graph create/update 会触发 session persistence，保证 workflow checkpoint 或 interrupt 前的 mutation 进入 `RuntimeContextCompact.TaskItems`。
