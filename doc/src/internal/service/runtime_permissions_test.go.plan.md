# runtime_permissions_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_permissions_test.go`
- 文档文件: `doc/src/internal/service/runtime_permissions_test.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 覆盖 Runtime V2 permission queue 的基础行为。

## 3. 关键测试覆盖
- request id 可被 `ApproveToolPermission` resolve 为 `allow_once`。
- resolved request 会从 pending queue 中移除。
- pending requests 可按 session 过滤，并按创建时间稳定排序。
- 已存在 session grant 时危险工具直接放行。
- 无 UI context 时危险工具 fail-closed。
- `allow_session` grant 会进入 session snapshot 并可持久化。
- session grants 可通过 API 列出并撤销。
