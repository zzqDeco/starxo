# App.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/App.vue
- 文档文件: doc/src/frontend/src/App.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src (根组件)

## 2. 核心职责
- 应用根组件，负责全局主题配置、Wails 事件监听注册、以及初始数据恢复。
- 配置 Naive UI 深色主题和自定义主题覆盖。
- 在 `onMounted` 中初始化所有 Wails 后端事件监听器，建立前后端通信桥梁。
- **新增 per-session 事件过滤**: 通过 `isActiveSession()` 函数检查事件的 `sessionId` 字段，确保只处理属于当前活跃会话的事件，防止后台会话的事件污染前端显示。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Wails 事件（session:switched, ssh:progress, ssh:connected, ssh:disconnected, container:progress, container:ready, container:activated, container:deactivated, agent:timeline, agent:done, agent:error, agent:interrupt, agent:mode_changed）
- 输出结果: 渲染 NConfigProvider 包裹的 MainLayout 组件；将 Wails 事件数据分发到对应 Store

## 4. 关键实现细节
- **`isActiveSession(data)` 过滤函数**: 检查事件数据中的 `sessionId` 字段，若存在则与 `sessionStore.activeSessionId` 比对；若不存在 sessionId 则视为属于活跃会话（向后兼容）
- **Wails 事件监听**（SSH/容器事件已从 `sandbox:*` 迁移到分离的命名空间）:
  - `session:switched` -> 完整的 per-session 状态恢复:
    1. 切换活跃会话，恢复消息历史
    2. 同步 agent 运行状态（`agentRunning`、`currentAgent`）到 chatStore
    3. 同步 agent 模式（`mode`）到 chatStore
    4. 同步中断对话框状态（`hasInterrupt`、`interrupt`）到 chatStore
    5. 更新容器和会话列表
  - `ssh:progress` -> 更新 connectionStore SSH 连接进度
  - `ssh:connected` -> 标记 SSH 连接就绪
  - `ssh:disconnected` -> 标记 SSH 断开，清除活跃容器
  - `container:progress` -> 更新 containerStore 容器创建进度
  - `container:ready` -> 设置活跃容器、刷新容器和会话列表
  - `container:activated` -> 设置活跃容器、刷新容器列表
  - `container:deactivated` -> 清除活跃容器
  - `agent:timeline` -> **过滤 sessionId**，仅处理活跃会话事件
  - `agent:done` -> **过滤 sessionId**（接收对象而非 nil，含 sessionId），仅处理活跃会话事件
  - `agent:error` -> **过滤 sessionId**（接收对象，含 sessionId + error），仅处理活跃会话事件
  - `agent:interrupt` -> **过滤 sessionId**，仅处理活跃会话中断
  - `agent:mode_changed` -> **过滤 sessionId**，仅处理活跃会话模式变更
- **会话恢复 (`restoreActiveMessages`)**:
  - 优先通过 `sessionStore.loadSessionData()` 从后端 `session_data.json` 加载统一的 display 数据
  - 如有 `streaming` 中途状态，追加 `[streaming interrupted]` 标记的不完整消息
  - 恢复消息后调用 `chatStore.restoreTodosFromMessages()` 从历史事件中提取最新 todos 快照
  - 后备逻辑: 若 `loadSessionData` 返回空，则尝试旧版 `loadChatDisplay` + `loadActiveMessages`
- **前端不再保存 display 数据**: `agent:done` 处理器中移除了 `saveChatDisplay` 调用，前端变为纯读取消费者

## 5. 依赖关系
- 内部依赖: MainLayout.vue、settingsStore、connectionStore、chatStore、sessionStore、containerStore、types (Session, Message, TurnEvent, InterruptEvent, ModeChangedEvent)
- 外部依赖: naive-ui、vue、wailsjs/runtime
- Wails 绑定: `wailsjs/go/service/SessionService` (LoadSessionData)

## 6. 变更影响面
- `isActiveSession()` 过滤逻辑确保后台会话事件不影响前端显示，是 per-session 并发安全的前端核心保障
- `session:switched` 事件处理器现在接收包含完整状态快照的事件（`agentRunning`、`currentAgent`、`mode`、`hasInterrupt`、`interrupt`），实现无缝的会话切换体验
- `agent:done` 和 `agent:error` 事件载荷从 `nil`/`string` 变为对象（含 `sessionId`），需对应处理
- `restoreActiveMessages` 从统一后端 `session_data.json` 恢复 display 数据，不再依赖前端 `saveChatDisplay`
- `agent:done` 处理器不再调用 `saveChatDisplay`，前端变为纯读取消费者

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增 Wails 事件时需同步更新 Go 后端的事件发射代码。
- SSH 和容器事件分离为两个命名空间，新增事件时保持命名一致性。
- `isActiveSession()` 的向后兼容逻辑（无 sessionId 视为活跃）在所有后端事件都携带 sessionId 后可收紧。
- `restoreActiveMessages` 的后备逻辑（旧版 display.json 加载）在所有用户迁移到 session_data.json 后可移除。
- `session:switched` 事件处理器中的状态恢复顺序（消息 -> 运行状态 -> 模式 -> 中断 -> 容器）应保持不变，避免 UI 闪烁。
