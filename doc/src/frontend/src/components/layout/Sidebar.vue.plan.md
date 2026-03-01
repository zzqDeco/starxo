# Sidebar.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/Sidebar.vue
- 文档文件: doc/src/frontend/src/components/layout/Sidebar.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout (布局模块)

## 2. 核心职责
- 左侧边栏组件，管理会话列表的展示和操作（新建、切换、重命名、删除），以及底部连接状态和连接/断开操作。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: sessionStore 状态 (sessions, activeSessionId)、connectionStore 状态 (sshConnected, dockerRunning, connecting, error, isReady)
- 输出结果: 渲染会话列表和连接控制 UI；调用 store actions 执行会话和连接操作

## 4. 关键实现细节
- **Pinia Store 交互**:
  - chatStore: clearMessages
  - connectionStore: sshConnected, dockerRunning, connecting, initStep, error, isReady, connect, disconnect
  - sessionStore: sessions, activeSessionId, createSession, switchSession, deleteSession, renameSession
- **会话操作**:
  - 新建会话: createSession() + clearMessages()
  - 切换会话: switchSession(id)，跳过当前活跃会话
  - 重命名: 内联 NInput 编辑，Enter/blur 确认，Escape 取消
  - 删除: 通过下拉菜单触发 deleteSession(id)
  - 下拉菜单: NDropdown 提供 rename/delete 选项
- **会话项渲染**: 图标 + 标题 (NEllipsis) + 消息数和时间 + 容器状态徽标 (dot + name) + 操作菜单按钮 (hover 显示)
- **底部连接区域**: SSH/Docker 状态点 + 连接进度 + 错误提示 + 连接/断开按钮
- **辅助函数**:
  - `formatTime(ts)` — 今天显示时间，否则显示日期
  - `containerStatusDot(status)` — 容器状态对应的点颜色类名

## 5. 依赖关系
- 内部依赖: `@/stores/chatStore`、`@/stores/connectionStore`、`@/stores/sessionStore`
- 外部依赖: `vue` (ref)、`naive-ui` (NButton, NIcon, NDropdown, NInput, NEllipsis)、`@vicons/ionicons5` (Add, ChatbubbleEllipses, EllipsisVertical)、`vue-i18n` (useI18n)

## 6. 变更影响面
- 会话操作修改需同步 sessionStore 的 action 签名
- 连接操作修改需同步 connectionStore
- 容器状态新增需更新 containerStatusDot 映射

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 内联重命名使用 blur 事件自动确认，注意边界情况（如空标题处理）。
- 会话列表较长时需关注滚动性能，当前使用 overflow-y: auto。
