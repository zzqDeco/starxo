# Sidebar.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/Sidebar.vue
- 文档文件: doc/src/frontend/src/components/layout/Sidebar.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout (布局模块)

## 2. 核心职责
- 左侧边栏组件，管理会话列表的展示和操作（新建、切换、重命名、删除），以及底部 **SSH 连接状态**和连接/断开操作。**仅管理 SSH 连接，不涉及容器管理**。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: sessionStore 状态、connectionStore 状态 (sshConnected, connecting, error)
- 输出结果: 渲染会话列表和 SSH 连接控制 UI

## 4. 关键实现细节
- **底部连接区域**:
  - 仅显示 SSH 状态点（移除了 Docker 状态点）
  - Connect 按钮调用 `connectionStore.connect()`（仅 SSH 连接）
  - Disconnect 按钮调用 `connectionStore.disconnect()`（仅 SSH 断开）
  - 按钮切换条件改为 `connectionStore.sshConnected`（不再依赖 `isReady`）
- **会话项渲染**: 图标 + 标题 + 消息数和时间 + 容器状态徽标 + 操作菜单
- **辅助函数**:
  - `formatTime(ts)` — 今天显示时间，否则显示日期
  - `containerStatusDot(status)` — 容器状态对应的点颜色类名

## 5. 依赖关系
- 内部依赖: `@/stores/chatStore`、`@/stores/connectionStore`、`@/stores/sessionStore`
- 外部依赖: vue、naive-ui、@vicons/ionicons5、vue-i18n

## 6. 变更影响面
- SSH 连接操作修改需同步 connectionStore
- 容器管理已完全移至 ContainerPanel.vue

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- Sidebar 仅负责 SSH 连接控制，容器相关操作不应添加到此组件。
