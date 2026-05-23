# todos_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/todos_test.go`
- 文档文件: `doc/src/internal/tools/todos_test.go.plan.md`
- 所属模块: tools tests

## 2. 核心职责
- 覆盖 `write_todos` / `update_todo` 的 session-scoped todo 存储行为。

## 3. 覆盖点
- 带 `ctx.Value("sessionID")` 的工具调用只读写对应 session bucket。
- 两个 session 的 todo 状态互相隔离。
- 缺少 `sessionID` 的工具调用仍回退到全局兼容 bucket。
