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
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Wails 事件（session:switched, **ssh:progress**, **ssh:connected**, **ssh:disconnected**, **container:progress**, **container:ready**, **container:activated**, **container:deactivated**, agent:timeline, agent:done, agent:error, agent:interrupt, agent:mode_changed）
- 输出结果: 渲染 NConfigProvider 包裹的 MainLayout 组件；将 Wails 事件数据分发到对应 Store

## 4. 关键实现细节
- **Wails 事件监听**（SSH/容器事件已从 `sandbox:*` 迁移到分离的命名空间）:
  - `session:switched` → 切换活跃会话、恢复消息、**更新容器状态（不清除 SSH 连接）**
  - `ssh:progress` → 更新 connectionStore SSH 连接进度
  - `ssh:connected` → 标记 SSH 连接就绪
  - `ssh:disconnected` → 标记 SSH 断开，清除活跃容器
  - `container:progress` → 更新 containerStore 容器创建进度
  - `container:ready` → 设置活跃容器、刷新容器和会话列表
  - `container:activated` → 设置活跃容器、刷新容器列表
  - `container:deactivated` → 清除活跃容器
  - `agent:timeline` / `agent:done` / `agent:error` / `agent:interrupt` / `agent:mode_changed` → 不变（已移除 `agent:plan` 幽灵监听）
- **会话恢复 (`restoreActiveMessages`)**:
  - 优先通过 `sessionStore.loadSessionData()` 从后端 `session_data.json` 加载统一的 display 数据
  - 如有 `streaming` 中途状态，追加 `[streaming interrupted]` 标记的不完整消息
  - 后备逻辑: 若 `loadSessionData` 返回空，则尝试旧版 `loadChatDisplay` + `loadActiveMessages`
- **前端不再保存 display 数据**: `agent:done` 处理器中移除了 `saveChatDisplay` 调用，前端变为纯读取消费者

## 5. 依赖关系
- 内部依赖: MainLayout.vue、settingsStore、connectionStore、chatStore、sessionStore、containerStore、types
- 外部依赖: naive-ui、vue、wailsjs/runtime
- Wails 绑定: `wailsjs/go/service/SessionService` (LoadSessionData)

## 6. 变更影响面
- 事件名变更（sandbox:* → ssh:*/container:*）是后端与前端的通信契约
- session:switched 不再清除 SSH 状态，仅更新容器状态
- `restoreActiveMessages` 从统一后端 `session_data.json` 恢复 display 数据，不再依赖前端 `saveChatDisplay`
- 移除 `agent:plan` 幽灵监听器（后端未发射此事件）
- `agent:done` 处理器不再调用 `saveChatDisplay`，前端变为纯读取消费者

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增 Wails 事件时需同步更新 Go 后端的事件发射代码。
- SSH 和容器事件分离为两个命名空间，新增事件时保持命名一致性。
- `restoreActiveMessages` 的后备逻辑（旧版 display.json 加载）在所有用户迁移到 session_data.json 后可移除。
- 前端不再负责 display 数据持久化，如需恢复此功能需同步修改后端保存逻辑。
