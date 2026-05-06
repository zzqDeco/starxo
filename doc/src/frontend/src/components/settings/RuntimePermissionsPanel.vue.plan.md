# RuntimePermissionsPanel.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/settings/RuntimePermissionsPanel.vue`
- 文档文件: `doc/src/frontend/src/components/settings/RuntimePermissionsPanel.vue.plan.md`

## 核心职责
- Settings / Permissions 分区，用于查看 Runtime V2 permission queue 和当前 session 的持久授权。
- 展示当前 session 的 pending tool approval requests。
- 展示 `allow_session` 产生的 session grants，并支持撤销单个 grant 或清空全部 grants。

## 输入与输出
- 输入:
  - `ChatService.ListToolPermissionRequests(sessionID)`
  - `ChatService.ListToolPermissionGrants(sessionID)`
  - Wails events: `runtime:permission_request`、`runtime:permission_resolved`、`runtime:permission_canceled`、`runtime:permission_grants_changed`
- 输出:
  - `ChatService.RevokeToolPermissionGrant(sessionID, toolName)`
  - `ChatService.ClearToolPermissionGrants(sessionID)`

## 关键实现细节
- 面板按 `sessionStore.activeSessionId` 查询后端，避免跨会话 grants 混淆。
- pending approval 仅展示状态，不在面板里直接审批；审批仍由全局 modal 承担。
- grants 变更后会重新刷新列表，后端负责保存 session data。

## 维护建议
- 新增 permission decision 类型时，需要同步后端 DTO、App 全局 modal 和本面板展示。
- 若后续支持项目级/全局授权，应单独扩展 DTO，避免混入 per-session grants。
