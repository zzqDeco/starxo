# workspaceDirtyStore.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/stores/workspaceDirtyStore.ts
- 文档文件: doc/src/frontend/src/stores/workspaceDirtyStore.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/stores

## 2. 核心职责
- 常驻记录后端 `workspace:changed` 事件，避免 WorkspacePanel 未挂载时丢失 terminal、agent 或上传产生的文件变化。
- 提供单调递增的 `revision` 和最近一次 `lastEvent`，由 WorkspacePanel 统一消费并走同一条刷新路径。

## 3. 输入与输出
- 输入来源: App.vue 的全局 Wails 事件监听。
- 输出结果: `revision`、`lastEvent`、`markChanged(event)`，供 WorkspacePanel 按 session/container/path/source/action 做过滤、debounce 和短延迟重试。

## 4. 维护要点
- `createdAt` 缺失时由 store 兜底补当前时间，兼容旧后端事件。
- store 不直接调用 Wails API、不枚举文件，只保存 dirty metadata；实际刷新仍由 WorkspacePanel 调用 FileService。
- 不应把 dirty event 当成真实文件列表，避免 UI 与远端 SFTP 状态分叉。
