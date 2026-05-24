# runtime_objective.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_objective.go`
- 文档文件: `doc/src/internal/service/runtime_objective.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 为 ChatService 提供 current-objective sidecar 的分类、克隆、prompt pinned message 和 compact 过滤逻辑。

## 3. 关键实现细节
- `objectiveScope(...)` 根据用户文本识别 continuation 请求，例如 `继续`、`下一步`、`do the above`、`continue previous task`。
- continuation intent 会先规范化末尾常见标点；bare `resume` 是 continuation，但 `resume ...` 前缀只在明确引用 previous/current task 时才继承旧上下文，避免 `resume parser design` 这类新任务误带历史。
- standalone objective 会记录 `HistoryStartIndex`，后续 prompt 只带本 turn 后的历史。
- `scopeCompactForObjective(...)` 在 standalone 下移除旧 summary、tasks、file read state、diff summary、todos 和 plan，同时保留 ToolSearch、permission grants 和 workspace routing。
- `runtimeObjectivePinnedMessages(...)` 生成模型可见 `<current-objective>`。

## 4. 维护边界
- continuation 信号要保守；泛化词如 `above` / `previous` 不能单独作为 substring 命中，`resume` 后跟具体对象时也不能默认 continuation，否则会让 standalone 请求误带旧历史。
- compact 过滤不能删除权限、ToolSearch 或 workspace guard 相关状态。
