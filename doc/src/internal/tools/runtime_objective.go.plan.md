# runtime_objective.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/runtime_objective.go`
- 文档文件: `doc/src/internal/tools/runtime_objective.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 在 tool execution context 中携带 current objective。
- 为 `ask_user` / `ask_choice` 提供基础 stale-question guard。

## 3. 关键实现细节
- `ContextWithRuntimeObjective(...)` 克隆 objective 后写入 context，避免调用方继续修改原指针。
- `RuntimeObjectiveFromContext(...)` 读取时也返回克隆。
- `RuntimeObjectivePromptGuard(...)` 对明显属于旧 debug/release/review 任务的问题 fail closed，返回 structured text error 给模型；continuation objective 会跳过 stale guard。
- `RuntimeObjectiveToolGuard(...)` 对明显属于旧任务的工具参数 fail closed，供 ChatModelAgent middleware 在工具执行前拦截；continuation objective 会跳过 stale guard。
- stale keyword 判断区分 tool/prompt text 的高置信短语和 objective 侧的宽松意图词；当前目标明确是 review/release 时允许相关工具调用，避免把 `review this PR` / `prepare a release` 误判成旧任务。
- ASCII term 使用 token/word 边界，避免 `review current files`、`preview`、`stage` 等正常 plan/tool 文本被误拦。

## 4. 维护边界
- guard 是行为防线，不是完整语义分类器；只拦截高置信 stale task 家族。
- 新增 stale family 时要避免误伤当前 objective 的合法追问；短英文词必须保持边界匹配。
