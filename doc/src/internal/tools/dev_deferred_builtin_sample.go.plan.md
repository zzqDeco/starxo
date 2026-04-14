# dev_deferred_builtin_sample.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/dev_deferred_builtin_sample.go`
- 文档文件: `doc/src/internal/tools/dev_deferred_builtin_sample.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 定义 dev-only experimental deferred builtin sample，用来真实压测 top-level deferred runtime 链路。

## 3. 输入与输出
- 输入来源: top-level catalog 注册流程
- 输出结果: 一个 `CatalogEntry`，其 canonical name 固定为 `dev_deferred_builtin_sample`

## 4. 关键实现细节
- sample 身份由单一常量 `DevDeferredBuiltinSampleCanonicalName` 标识。
- 注册、announcement、`tool_search`、discovered-state cleanup、测试全部复用该常量。
- sample 固定属性：
  - builtin
  - `ShouldDefer=true`
  - `AlwaysLoad=false`
  - `IsMcp=false`
  - `ToolClass=builtin`
  - `DeferReason=dev_experimental`
- sample 本身必须无副作用，适合开发态手工 smoke 和回归测试。

## 5. 维护建议
- 不要把这个 sample 当成真实生产 rollout 候选；默认生产 registry 不注册它。
- 关闭 sample 后的 cleanup 必须走显式 canonical-name 规则，不能靠隐式推断。
