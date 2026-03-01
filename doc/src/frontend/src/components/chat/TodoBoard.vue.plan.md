# TodoBoard.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/TodoBoard.vue
- 文档文件: doc/src/frontend/src/components/chat/TodoBoard.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat (聊天模块)

## 2. 核心职责
- 任务看板组件，以 DAG 拓扑排序的分层布局展示 Agent 的 Todo 任务列表及其依赖关系和执行状态。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Props (todos: TodoItem[])
- 输出结果: 渲染任务进度板 UI

## 4. 关键实现细节
- **Props 定义**: `todos: TodoItem[]`
- **导出类型**: `TodoItem` 接口 — id, title, status (pending/in_progress/done/failed/blocked), depends_on?: string[]
- **DAG 拓扑排序** (`layers` computed):
  - 计算每个任务的入度 (inDegree) 和子节点映射 (children)
  - 使用 BFS 逐层处理零入度节点
  - 处理残留的环形或孤立节点
  - 输出分层的 TodoItem[][] 用于渲染
- **状态配置**: 5 种状态 (pending/in_progress/done/failed/blocked) 各有对应图标、颜色和标签
- **统计计算**: stats (各状态计数), progressPercent (完成百分比)
- **模板结构**: 头部 (标题 + 百分比 + 进度条 + 状态统计) → DAG 分层列表 (层间有箭头连接器)，每项显示状态图标 + 标题 + 依赖标签 + ID
- **样式**: in_progress 状态有呼吸发光动画 (todoGlow)，done 状态标题有删除线

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `vue` (computed)、`naive-ui` (NIcon)、`@vicons/ionicons5` (CheckmarkCircle, EllipseOutline, Reload, CloseCircle, LockClosed, ArrowForward)

## 6. 变更影响面
- `TodoItem` 类型修改影响 TimelineEventItem 中的 parsedTodos 解析
- 状态类型扩展需同步 statusConfig 映射
- DAG 排序算法修改影响任务展示顺序

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- TodoItem 接口同时被 TimelineEventItem 导入使用，类型变更需注意双向影响。
- 拓扑排序对环形依赖有容错处理，但 UI 上无明确提示。
