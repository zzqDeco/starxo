# runtime_permissions.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_permissions.go`
- 文档文件: `doc/src/internal/service/runtime_permissions.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 实现 Runtime V2 工具执行审批队列。
- 支持前端审批弹窗、allow once、allow session、deny，以及 session grant 持久化。

## 3. 输入与输出
- 输入来源:
  - `tools.ToolExecutionPermissionProvider.RequestToolPermission(...)`
  - Wails 方法：`ApproveToolPermission`、`DenyToolPermission`
- 输出结果:
  - Wails events：`runtime:permission_request`、`runtime:permission_resolved`、`runtime:permission_canceled`
  - `model.RuntimePermissionGrant` 持久化到 `SessionData.PermissionGrants`

## 4. 关键实现细节
- read-only trusted 工具直接 allow once，不进入审批队列。
- 已存在 `allow_session` grant 的工具直接放行。
- 无 Wails UI context 时 fail-closed，避免后台测试或 headless 运行静默执行危险工具。
- 每个请求生成 `perm-<timestamp>` request id，并把请求挂入 `permissionRequests` map。
- 前端通过 request id 回调 allow once / allow session / deny。
- `allow_session` 会写入当前 session 的 `permissionGrants` 并异步保存 session。
- 请求随 tool-call context 取消或 10 分钟超时会发 `runtime:permission_canceled`。

## 5. 维护建议
- 新增危险 runtime tools 时不应绕过 `WrapMCPToolWithPermissionCheck`。
- 若增加全局 allowlist 或项目级 rules，应先落在本文件，再让前端只展示最终 request。
