# chat_sandbox_loss_test.go 技术说明

## 文件定位
- 源文件: `internal/service/chat_sandbox_loss_test.go`
- 覆盖 `ChatService.UpdateSandbox(nil)` 对 agent runtime run state 的收敛行为。

## 覆盖范围
- active sandbox 丢失时取消 running agent run。
- active sandbox 丢失时取消 starting agent run，并关闭 startup wait channel。
- 设置新的非 nil sandbox manager 时不取消正在运行的 agent run。

## 维护要点
- 测试直接构造 `SessionRun` 状态，不依赖真实 Eino runner 或 Wails event context。
- 如 `SessionRun` 增加新的运行中引用字段，需同步检查 sandbox loss cleanup 是否会释放。
