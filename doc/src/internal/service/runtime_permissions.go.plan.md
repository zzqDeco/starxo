# runtime_permissions.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_permissions.go`
- 文档文件: `doc/src/internal/service/runtime_permissions.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 实现 Runtime V2 工具执行审批队列。
- 支持前端审批弹窗、allow once、allow session、deny，以及 session grant 持久化。
- 暴露 pending request / grant 查询和 grant 撤销 API，支持前端 reload 或切会话后恢复队列状态。
- 记录非 read-only 工具的 permission decision audit，给后续规则系统、自动化和审计 UI 使用。

## 3. 输入与输出
- 输入来源:
  - `tools.ToolExecutionPermissionProvider.RequestToolPermission(...)`
  - Wails 方法：`ApproveToolPermission`、`DenyToolPermission`
  - Wails 方法：`ListToolPermissionRequests`、`ListToolPermissionGrants`、`RevokeToolPermissionGrant`、`ClearToolPermissionGrants`
  - Wails 方法：`ListToolPermissionAudit`
- 输出结果:
  - Wails events：`runtime:permission_request`、`runtime:permission_resolved`、`runtime:permission_canceled`、`runtime:permission_grants_changed`、`runtime:permission_audit_changed`
  - `model.RuntimePermissionGrant` 持久化到 `SessionData.PermissionGrants`
  - `model.RuntimePermissionAudit` 持久化到 `SessionData.PermissionAudit` 和 runtime compact

## 4. 关键实现细节
- read-only trusted 工具直接 allow once，不进入审批队列。
- 工具调用 context 中存在 runtime mode override 时，审批队列使用该 override 而不是原始 session mode；用于保证 subagent plan/default 约束与 provider/tool surface 一致。
- 原始 session mode 读取与 `SetMode` 写入同样受 `ChatService.mu` 保护，避免并发 mode 切换时审批逻辑读到 data race。
- 已存在 `allow_session` grant 的工具直接放行。
- bypass mode 与 session grant 都会生成 audit record，避免“自动放行但无解释”。
- 无 Wails UI context 时 fail-closed，避免后台测试或 headless 运行静默执行危险工具。
- missing UI context、用户审批、超时/取消都会写入 audit record。
- 每个请求生成 `perm-<timestamp>` request id，并把请求挂入 `permissionRequests` map。
- `ListToolPermissionRequests(sessionID)` 会按创建时间排序返回 pending 队列；空 sessionID 返回全部队列。
- 前端通过 request id 回调 allow once / allow session / deny。
- resolve 时先从 pending map 删除，再通知等待中的 tool call，避免前端刷新时看到已处理请求。
- `allow_session` 会写入当前 session 的 `permissionGrants` 并异步保存 session。
- Settings / Permissions 面板可列出、撤销、清空当前 session 的持久授权。
- 请求随 tool-call context 取消或 10 分钟超时会发 `runtime:permission_canceled`。
- audit record 最多保留最近 200 条，compact prompt 只注入最近一批摘要，避免长会话膨胀。

## 5. 维护建议
- 新增危险 runtime tools 时不应绕过 `WrapMCPToolWithPermissionCheck`。
- 若增加全局 allowlist 或项目级 rules，应先落在本文件，再让前端只展示最终 request。
- 后续 rules / acceptEdits / automation 应复用 audit record，避免另建一套不可追踪的许可路径。
