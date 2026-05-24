# runtime_objective_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/runtime_objective_test.go`
- 文档文件: `doc/src/internal/tools/runtime_objective_test.go.plan.md`
- 所属模块: tools test

## 2. 核心职责
- 覆盖 tool context 中 current-objective guard 的基础行为。

## 3. 覆盖点
- 与当前 objective 无关的旧测试/debug 追问会被拒绝。
- 与当前 objective 相关的问题允许继续触发 ask tool interrupt。
- 与当前 objective 无关的旧任务工具调用会被拒绝。
- continuation objective 中引用旧任务的 ask/tool 调用允许通过。
- `preview` / `stage` 等合法词不会被误判为 `review` / `tag` stale family。
