# sessionStore.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/stores/sessionStore.ts
- 文档文件: doc/src/frontend/src/stores/sessionStore.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/stores (Pinia 状态管理)

## 2. 核心职责
- 管理会话列表和活跃会话状态，封装与 Go 后端 SessionService 的所有交互。
- 提供会话的 CRUD 操作和消息持久化/恢复功能。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Sidebar.vue 用户操作、App.vue 的 session:switched 事件
- 输出结果: 响应式的会话列表和活跃会话状态，供 Sidebar、App.vue 等组件消费

## 4. 关键实现细节
- **State 属性**:
  - `sessions: Session[]` — 会话列表（含容器富信息）
  - `activeSessionId: string | null` — 当前活跃会话 ID
  - `loading: boolean` — 会话切换加载状态
- **Getters**:
  - `activeSession` — 从列表中查找当前活跃会话对象
- **Actions**:
  - `loadSessions()` — 调用 `ListSessionsEnriched` 加载富化会话列表并通过 `GetActiveSession` 同步活跃会话
  - `createSession(title?)` — 创建新会话并设为活跃
  - `switchSession(sessionId)` — 切换到指定会话
  - `deleteSession(sessionId)` — 删除会话
  - `renameSession(sessionId, title)` — 重命名会话
  - `loadActiveMessages()` — 加载活跃会话的基础消息（通过 `GetActiveSessionMessages`）
  - `saveChatDisplay(messages)` — 持久化富显示消息（含时间线事件），使用 JSON 序列化
  - `loadChatDisplay()` — 加载富显示消息，使用 JSON 反序列化
  - `loadSessionData()` — 从后端加载统一的 `SessionData`（通过 `SessionService.LoadSessionData` Wails 绑定），返回包含 messages + display + streaming 的完整会话数据
  - `setActiveSession(session)` — 从 Wails 事件更新本地会话数据
- **Wails 绑定调用**: ListSessionsEnriched, CreateSession, SwitchSession, DeleteSession, RenameSession, GetActiveSession, GetActiveSessionMessages, SaveChatDisplay, LoadChatDisplay, LoadSessionData

## 5. 依赖关系
- 内部依赖: `@/types/session` (Session)
- 外部依赖: `pinia` (defineStore)、`vue` (ref, computed)
- Wails 绑定: `wailsjs/go/service/SessionService` (10 个方法，含新增 LoadSessionData)

## 6. 变更影响面
- 修改会话列表结构影响 Sidebar 的会话渲染
- 修改消息持久化逻辑影响会话切换时的消息恢复
- `loadSessionData` 被 `App.vue` 的 `restoreActiveMessages` 调用，是会话恢复的主路径
- `saveChatDisplay`/`loadChatDisplay` 为旧版接口，保留向后兼容；新代码应使用 `loadSessionData`
- Wails 绑定方法签名变更需同步 Go 后端 SessionService

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `saveChatDisplay`/`loadChatDisplay` 为旧版接口，前端不再主动保存 display 数据（由后端统一持久化）。待旧版数据完全迁移后可考虑移除。
- `loadSessionData` 返回后端的 `model.SessionData` 结构，其 TypeScript 类型定义在 `frontend/wailsjs/go/models.ts` 中自动生成。
- 新增 SessionService 方法时需同步更新 Wails 绑定导入。
