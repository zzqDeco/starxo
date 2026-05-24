# runtime_objective_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_objective_test.go`
- 文档文件: `doc/src/internal/service/runtime_objective_test.go.plan.md`
- 所属模块: service test

## 2. 核心职责
- 覆盖 current-objective 行为层的 prompt/history 边界。

## 3. 覆盖点
- standalone 新请求不会携带旧任务历史。
- continuation 请求会继承最近任务上下文。
- generic `above` / `previous` 文本不会被误判为 continuation。
- standalone compact 会丢弃旧 active work，同时保留 ToolSearch 和 permission grants。
