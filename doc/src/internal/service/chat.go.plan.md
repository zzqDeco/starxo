# chat.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/service/chat.go`
- 文档文件: `doc/src/internal/service/chat.go.plan.md`
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 实现 `ChatService`，负责多会话聊天、runner 生命周期、事件流转、中断恢复、mode 切换。
- 维护共享 runner 与 per-session `SessionRun`，其中 discovery 采用 `SessionData.DiscoveredTools` 持久化、`SessionRun.discoveredTools` 内存态、每次模型调用前按 session 现算。
- 构建并装配 deferred MCP surface：MCP action/resource catalog、`tool_search`、permission gate、per-model-call late binding、announcement 注入。
- 维护 `RunnerBundle` 的安装、retire、freshness probe 和事务式 swap，保证多 session 共享 runner 下的 freshness 更新不会打断正在运行或待 resume 的会话。
- 提供一致性快照导出与 save-time discovery 剪枝接口，供 `SessionService` 原子落盘。

## 3. 输入与输出
- 输入来源:
  - Wails 绑定调用：`SendMessage`、`ResumeWithAnswer`、`ResumeWithChoice`、`SetMode`、`BuildRunners`
  - 依赖注入：`config.Store`、`sandbox.SandboxManager`、`SessionService`
  - 运行时上下文：`contextWithSessionID(...)` 注入的 `sessionID`
- 输出结果:
  - Wails 事件：`agent:timeline`、`agent:error`、`agent:done`、`agent:interrupt`、`agent:mode_changed`
  - 一致性快照：`ExportSessionSnapshot(sessionID)`
  - discovery 状态操作：`RestoreSessionData`、`AddDiscoveredTool`、`ReplaceDiscoveredTools`、`PruneDiscoveredToolsForSave`

## 4. 关键实现细节
- `SessionRun` 现在同时持有：
  - `ctxEngine`
  - `timeline`
  - `streamingState`
  - `discoveredTools map[string]model.DiscoveredToolRecord`
- `SessionRun` 额外记录：
  - `activeBundleGeneration`
  - `activeRunnerKind`
  用于运行中引用 bundle；interrupt 挂起后引用转移到 `PendingInterrupt`
- `contextWithSessionID(...)` 是所有 per-model-call deferred 计算的唯一 sessionID 注入入口；下游只能从 `context.Context` 读取，不从 shared runner 或全局 active session 推断。
- shared runner 已收敛为 `RunnerBundle`：
  - `Generation`
  - `ConfigDigest`
  - `DefaultRunner` / `PlanRunner`
  - `MCPCatalog`
  - `MCPHandles`
  - `LastFreshnessCheckAt`
  - `SurfaceRelevantFingerprint`
  - `CachedSurfaceMetadataByServer`
- freshness coordinator 采用 detached probe：
  - 锁内只读取 installed bundle 快照并登记 singleflight task
  - `RefreshMetadata` / list 拉取全部在锁外进行
  - probe 无变化时只在 generation + digest 仍匹配时回写 freshness 时间戳
  - probe 有变化时锁外 prepare 新 bundle，锁内 install；旧 bundle 进入 retire
- deferred MCP provider 绑定在 runner generation 上：
  - catalog / handles 固定到该代 runner
  - discovery 仍从 `SessionRun` 按 session 读取
  - 避免 runner 重建时污染正在运行的旧会话
- resume 不按当前 session mode 选 runner：
  - interrupt 时把 `BundleGeneration + RunnerKind` 写进 `PendingInterrupt`
  - resume 必须按这两个字段取同代、同类 runner
  - 找不到对应 bundle 或 runner 时显式失败，不 fallback
- save-time discovery 剪枝规则：
  - 运行前重建只读
  - 成功保存时只删除“canonical 不存在”或“已不再属于 deferred MCP 范围”的记录
  - 剪枝结果同时写回内存和磁盘

## 5. 依赖关系
- 内部依赖:
  - `internal/agent`: 构建 deep agent、runner、prompt
  - `internal/context`: `Engine`、`TimelineCollector`
  - `internal/tools`: MCP runtime、catalog、tool_search、permissions、dynamic surface middleware
  - `internal/service/session_svc.go`: save/export 调用方
- 外部依赖:
  - `github.com/cloudwego/eino/adk`
  - `github.com/cloudwego/eino/compose`
  - `github.com/cloudwego/eino/schema`
  - `github.com/wailsapp/wails/v2/pkg/runtime`

## 6. 变更影响面
- 顶层真实工具面已从“registry 全量注入”改成“always-loaded orchestration + deferred MCP surface”。
- default mode 与 plan mode 共用同一套 deferred helper，但 plan mode 通过只读规则收紧 searchable/loadable 池。
- runner 仍是跨 session 共享的，但 deferred tool visibility、announcement、execution gating 都变成 per-model-call、per-session 计算；runner rebuild 则通过 bundle generation 与 retired bundle 生命周期解耦。
- `SessionService.SaveSessionByID` 现在依赖本文件导出的单一一致性快照与 discovery 剪枝接口。

## 7. 维护建议
- 任何新的 deferred MCP 规则都应先落到 shared helper 或 provider，不要在 announcement / tool_search / execution gate 各算一份。
- 若修改 MCP runner 重建逻辑，必须同时验证“旧 runner 继续可用、旧 handles 延迟回收”。
- 若修改 discovery 持久化格式，需同步更新 `session_data.go`、`session_svc.go`、`tool_search.go` 和相关测试。
