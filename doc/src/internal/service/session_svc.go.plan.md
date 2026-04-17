# session_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/service/session_svc.go`
- 文档文件: `doc/src/internal/service/session_svc.go.plan.md`
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 管理会话生命周期、会话切换、会话删除与会话持久化。
- 当前保存路径已经升级为：
  - `SessionService.SaveSessionByID(sessionID)` 的按 session 异步保存
  - `SessionService.SaveSessionByIDBlocking(sessionID)` 的显式同步保存入口
  - per-session coalescing
  - 单一一致性快照导出
  - 成功保存后的 discovery 剪枝回写

## 3. 输入与输出
- 输入来源:
  - Wails 绑定：`CreateSession`、`SwitchSession`、`DeleteSession`、`SaveCurrentSession`、`LoadSessionData`
  - `ChatService.ExportSessionSnapshot(sessionID)`
- 输出结果:
  - `session_data.json`
  - `session.json`
  - `session:switched`

## 4. 关键实现细节
- `SaveSessionByID(sessionID)`：
  - 同一 session 最多一个 in-flight save
  - 保存期间若再次请求，只保留一个 trailing save
- `SaveSessionByIDBlocking(sessionID)`：
  - 直接复用当前 blocking save 路径
  - 这条 PR 只是给后续 runner/UI PR 提供同步保存入口
  - 当前前端不主动调用
- `saveSessionByIDBlockingLocked(...)`：
  - 只消费 `ChatService` 导出的单一一致性快照
  - 在落盘前调用 discovery 剪枝
  - 落盘成功后把剪枝结果写回 `ChatService`
- `LoadSessionData()`：
  - 保持 active-session scoped
  - 有文件时返回已经 normalized 的 v4 `SessionData`
  - 无文件时继续返回 `nil`
- startup / session switch 走“store 先 normalize+log，再 service restoreNormalized”路径：
  - 同一条调用链只记录一次 normalize warning
  - `session:switched.mode` 继续来自 `GetSessionRunSnapshot(...)`，因此会反映 restore 后的 persisted mode
- save-time discovery 剪枝已经收敛为“结构性剪枝”：
  - current config 始终可用于删除空 canonical 和已移除 server
  - 当前没有 installed bundle，或 installed bundle config/freshness 不可信时，会停用 runtime-metadata-based 删除并 fail-open 保留 history
  - `CanonicalName == ""` 才直接删除
  - `record.Server != ""` 且 server 已从当前 config 移除时删除
  - 即使 bundle stale 或缺失，`server removed from current config` 这类 current-config-based 删除仍然有效
  - 只有 current config 对应、且 fresh bundle 的已知 metadata 明确证明 canonical 已不存在或已不再属于 deferred MCP 范围时才删除
  - 不因当前 mode、权限或 server 临时状态而丢失 discovered history
- durability 目标是 best effort，不承诺硬崩溃零丢失。
- `SetMode()` / `ClearHistory()` 触发的保存仍然是 async best-effort，不提供 blocking durability。

## 5. 依赖关系
- 内部依赖:
  - `internal/storage`
  - `internal/model`
  - `internal/service/chat.go`
- 外部依赖:
  - `github.com/wailsapp/wails/v2/pkg/runtime`

## 6. 变更影响面
- 会话保存不再只依赖 active session，同一个后台 session 的 discovery 变化也能触发保存。
- `SessionData.DiscoveredTools` 与 plan-mode v2 state 都成为持久化内容的一部分。

## 7. 维护建议
- 如果要修改保存策略，优先保持“单快照导出 + 锁外 IO + 成功后回写”的结构，不要回到分散读取状态。
