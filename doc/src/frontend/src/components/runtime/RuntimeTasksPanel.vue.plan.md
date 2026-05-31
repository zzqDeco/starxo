# RuntimeTasksPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/runtime/RuntimeTasksPanel.vue
- 文档文件: doc/src/frontend/src/components/runtime/RuntimeTasksPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/runtime

## 2. 核心职责
- 展示 Runtime V2 后台任务面板，覆盖当前会话的 Bash / Agent background tasks。
- 调用 Wails `ChatService.ListRuntimeTasks`、`ReadRuntimeTaskOutput`、`StopRuntimeTask`。
- 监听 `runtime:task_started`、`runtime:task_completed`、`runtime:task_stopped` 事件，按当前 session 过滤并更新列表。

## 3. 输入与输出
- 输入来源: `Props.show`、`Props.embedded`、`sessionStore.activeSessionId`、Runtime task Wails events。
- 输出结果: `update:show` 关闭面板；用户可刷新任务、读取输出、复制输出、停止运行中任务。

## 4. 关键实现细节
- 右侧抽屉布局:
  - 左侧任务列表按 `startedAt` 倒序稳定展示。
  - 右侧详情区展示任务 id/type/status/duration/outputSize、命令、输出和错误。
- Embedded inspector:
  - `embedded=true` 时不渲染 backdrop/关闭按钮，面板填满 MainLayout inspector body。
  - 供 Figma v0.5 四态 inspector 的 `tasks` 模式使用。
- macOS 下作为轻量 inspector sheet 处理：透明 backdrop、blur panel、低对比度行选中态，避免覆盖式网页 drawer 的厚重感。
- 自动刷新:
  - 面板打开时拉取当前会话任务。
  - 每 4 秒刷新任务列表；选中任务仍在运行时同步刷新输出。
- 输出读取:
  - 单次读取上限 128KB，避免大输出阻塞 UI。
  - `truncated` 时展示提示，完整输出仍保留在后端 output file。
- 停止任务:
  - 仅对 `running` / `pending` 状态显示停止按钮。
  - 停止后刷新 snapshot 和输出；后端取消成功时状态为 `killed`，前端按终止态展示和统计。

## 5. 依赖关系
- 内部依赖: `sessionStore`、`useUiFeedback`、Wails `ChatService` bindings。
- 外部依赖: `vue`、`naive-ui`、`@vicons/ionicons5`。

## 6. 变更影响面
- Runtime V2 后台任务能力从纯后端 API 变成前端可见可控。
- Header 和 CommandPalette 新增入口，MainLayout 负责 `tasks` inspector mode；旧 overlay 形态保留兼容。

## 7. 维护建议
- 如果未来 task output 支持分页加载，应复用后端 `offset/limit/nextOffset` 字段，而不是一次性读取全部内容。
- 如果新增 task 类型，优先保持 `RuntimeTaskSnapshot.type/status` 通用显示，不在前端硬编码每类任务协议。
- 事件监听需保留 `EventsOn` 返回的 cleanup，并在 `onUnmounted` 中逐个释放。
