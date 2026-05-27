# runtime_tasks_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_tasks_test.go`
- 文档文件: `doc/src/internal/service/runtime_tasks_test.go.plan.md`
- 所属模块: service tests

## 2. 核心职责
- 覆盖 Runtime task manager 的后台任务生命周期和 persistent task graph 行为。

## 3. 关键测试覆盖
- 后台 shell task 可启动、完成、写入 output，并通过 byte range 读取。
- running task 可被 stop，状态变为 `killed`。
- `TaskCreate` 对应的 task graph item 会规范化 status 和 dependencies。
- 同一时钟 tick 连续创建 task graph items 不会发生 id 覆盖。
- `TaskUpdate` 可关闭 task graph item，并记录 completed timestamp。
- `TaskList` 默认隐藏 closed items，`include_closed` 可列出 completed/canceled items。
- clear task graph 会在释放 manager lock 后发出 callback，callback 可安全读取 compact task items。
- task graph compact/restore 后仍可通过 id 读取。
- task graph restore 会替换当前 session 旧 records，空 restore 会清空当前 session，同时保留其他 session records。
