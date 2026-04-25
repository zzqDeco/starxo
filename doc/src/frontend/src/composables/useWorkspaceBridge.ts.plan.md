# useWorkspaceBridge.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/composables/useWorkspaceBridge.ts
- 文档文件: doc/src/frontend/src/composables/useWorkspaceBridge.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/composables

## 2. 核心职责
- 提供轻量的前端内工作区打开事件桥，用于从聊天时间线打开 WorkspaceDrawer 并预览指定路径。

## 3. 关键实现细节
- 事件名固定为 `starxo:workspace-open-path`。
- `openWorkspacePath(path?)` 保存 pending path，并通过 `window.dispatchEvent(CustomEvent)` 广播。
- `consumePendingWorkspacePath()` 供 WorkspacePanel 首次 mounted 时消费，避免抽屉尚未挂载时丢失路径。
- `onWorkspaceOpenPath(handler)` 注册监听并返回清理函数。

## 4. 变更影响面
- MainLayout 监听该事件来打开工作区抽屉。
- WorkspacePanel 监听并选择/预览对应文件。
- TimelineEventItem 对文件工具路径调用该桥。
