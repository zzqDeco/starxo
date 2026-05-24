# runtime_behavior_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/agent/runtime_behavior_test.go`
- 文档文件: `doc/src/internal/agent/runtime_behavior_test.go.plan.md`
- 所属模块: agent test

## 2. 核心职责
- 覆盖 RuntimeBehaviorMiddleware 的 tool-call continuation guard。

## 3. 覆盖点
- 当前 objective 与旧任务无关时，明显 stale 的 Bash/tool 参数会被 middleware 拦截，不会执行真实 endpoint。
