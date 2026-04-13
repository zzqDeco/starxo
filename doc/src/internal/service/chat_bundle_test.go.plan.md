# chat_bundle_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/service/chat_bundle_test.go`
- 文档文件: `doc/src/internal/service/chat_bundle_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: service

## 2. 核心职责
- 这是 `ChatService` deferred MCP / runner bundle 体系的回归保护主测试文件。
- 它覆盖 runner generation、freshness task、discovery pruning、fallback 语义、startup 引用和 detached build 生命周期。

## 3. 输入与输出
- 输入来源:
  - 手工构造的 `RunnerBundle`、`PendingInterrupt`、`SessionRun`
  - 测试配置快照、fake MCP handles、fake deferred provider
- 输出结果:
  - `ensureBundleReadyForNewRun(...)`、`resolvePendingRunnerLocked(...)`、`PruneDiscoveredToolsForSave(...)` 等行为断言

## 4. 关键测试覆盖
- resume 绑定 `BundleGeneration + RunnerKind`，缺失 bundle 时不会 fallback
- retired bundle cleanup 会尊重 running / pending interrupt / pending start 引用
- freshness task 的 singleflight key、stale no-change 保护、config digest mismatch rebuild、config-version task mismatch 重判
- `ConfigIdentityDigest` 的确定性和 cached metadata identity 匹配规则
- save-time pruning 的 fail-open 语义：
  - 无 installed bundle
  - stale bundle
  - config digest mismatch
  - resource discovery 的 `Server == \"\"`
  - metadata shrink / mismatched cache
  - 只有 clearly invalid 才删
- cold-start / freshness detached task 生命周期：
  - 失败 task 不复用
  - discard 结果会关闭 handles
  - all waiters canceled 后 detached build 仍可产出 warm bundle
- freshness recoverable fallback：
  - config drift 下不会自旋
  - fallback 不跨 config 边界
  - installed bundle 消失时 hard fail
- startup stop 与 `pendingStartBundleGeneration` 生命周期：
  - stop 只取消等待
  - detached build 继续
  - startup 放弃或 session 删除时会清理 pending start 引用

## 5. 依赖关系
- 内部依赖: `chat.go`、`session_svc.go`、`internal/tools/*`
- 外部依赖: `github.com/cloudwego/eino/adk`

## 6. 变更影响面
- 这个测试文件直接保护 deferred MCP 一致性主线，任何 bundle / freshness / pruning / startup 语义回退都会先在这里暴露。

## 7. 维护建议
- 修改 `ensureBundleReadyForNewRun(...)`、freshness task、discovery pruning 或 startup finalize 时，应先扩展本文件测试，再改实现。
