# deferred_surface_debug.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/deferred_surface_debug.go`
- 文档文件: `doc/src/internal/service/deferred_surface_debug.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 提供 phase-2 收口使用的纯计算 helper，统一产出 deferred surface 的 debug / preview 视图。
- 承载 `DeferredSurfaceDebug`、preview 结构、runtime option 锁存语义，以及 Wails debug API 的只读 contract。

## 3. 输入与输出
- 输入来源:
  - `SessionRun` 侧的只读快照
  - service / bundle / config 侧的只读快照
- 输出结果:
  - `DeferredSurfaceDebug`
  - synthetic preview 对应的纯计算结果

## 4. 关键实现细节
- 日志、snapshot debug 计算、Wails explicit debug API 都复用同一个纯计算核心，但 snapshot export 与 explicit debug API 保持不同的暴露语义。
- helper 只读输入快照，不调用 `PrepareDeferredSyntheticMessages(...)`，不推进任何 delta state。
- `DeferredSurfaceDebug` 是 best-effort runtime debug view，不是强一致落盘快照。
- 组装约束固定为：
  - 先分别复制 run / bundle / config 输入
  - 再在锁外纯计算
  - 不允许在单个 helper 内嵌套持有 `run.stateMu` 与 `ChatService.mu`
- `ConfigSnapshotError` 采用部分退化语义：
  - config 相关字段降级为零值
  - bundle / pool / preview / visibility 尽量保留
- `BuildWarnings` 固定去重、稳定排序
- debug API gating 和 dev-only sample gating 都通过启动时锁存的 runtime options 控制，不在请求路径现读环境变量。
- `GetDeferredSurfaceDebug(sessionID)` 是 explicit debug API：
  - 开关关闭时固定报 `deferred surface debug API is disabled`
  - 不返回部分数据
- `ExportSessionSnapshot(sessionID)` 的 debug 字段是可选附带信息：
  - 开关关闭时正常返回 snapshot，但 `DeferredSurfaceDebug == nil`
  - snapshot gating 必须在 helper 最前面短路，不能“先算再丢”
  - 只有开关开启时，snapshot 才允许进入 session lookup / missing-session debug / `buildDeferredSurfaceDebug(...)`
- snapshot/API parity 只在 debug 开启时适用；关闭时二者刻意不同：
  - snapshot 不带 debug 字段
  - explicit debug API 固定报错

## 5. 变更影响面
- 影响 `chat.go` 的 snapshot/export、provider observability 日志和 dev-only debug API。
- 影响 README 中的调试入口说明。

## 6. 维护建议
- 新增 debug 字段时先决定其零值与降级语义，再加入 helper，避免 snapshot / API / 日志各自拼装。
- 不要把 debug helper 演化成带副作用的“预执行”路径；commit 逻辑必须留在实际模型调用成功后。
