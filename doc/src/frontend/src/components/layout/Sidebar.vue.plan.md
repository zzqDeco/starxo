# Sidebar.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/Sidebar.vue
- 文档文件: doc/src/frontend/src/components/layout/Sidebar.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout (布局模块)

## 2. 核心职责
- 左侧边栏组件，管理会话列表的展示和操作（新建、切换、重命名、删除），以及底部 **SSH 连接状态**和连接/断开操作。**仅管理 SSH 连接，不涉及容器管理**。
- 支持 `compact` 模式，供主布局在 workspace inspector 打开或窗口宽度不足时压缩为图标 rail。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: sessionStore 状态、connectionStore 状态 (sshConnected, connecting, error)、chatStore.sessionRunStates
- 输出结果: 渲染会话列表和 SSH 连接控制 UI

## 4. 关键实现细节
- **底部连接区域**:
  - 仅显示 SSH 状态点（移除了 Docker 状态点）
  - Connect 按钮调用 `connectionStore.connect()`（仅 SSH 连接）
  - Disconnect 按钮调用 `connectionStore.disconnect()`（仅 SSH 断开）
  - 按钮切换条件改为 `connectionStore.sshConnected`（不再依赖 `isReady`）
- **会话项渲染**: 图标 + 标题 + 消息数和时间 + 容器状态徽标 + 运行态徽标 + 操作菜单
- **Compact rail**: 仅保留新建、会话图标和 SSH 状态点，隐藏标题、元信息、运行态和连接按钮，避免左侧留下大面积空白。
- **运行态徽标**:
  - `running`: 显示当前 agent 名称
  - `waiting`: 显示等待用户输入
  - `plan/default`: 显示最近同步到该 session 的模式
- **辅助函数**:
  - `formatTime(ts)` — 今天显示时间，否则显示日期
  - `containerStatusDot(status)` — 容器状态对应的点颜色类名
  - `runStateFor/runStateLabel/runStateClass` — 读取并格式化 session 运行态

## 5. 依赖关系
- 内部依赖: `@/stores/chatStore`、`@/stores/connectionStore`、`@/stores/sessionStore`
- 外部依赖: vue、naive-ui、@vicons/ionicons5、vue-i18n

## 6. 变更影响面
- SSH 连接操作修改需同步 connectionStore
- 容器管理已完全移至 ContainerPanel.vue
- `agent:run_state` 事件会改变会话项徽标，不影响会话切换逻辑

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- Sidebar 仅负责 SSH 连接控制，容器相关操作不应添加到此组件。
- macOS native 样式应保持 source list 语义：主操作与连接按钮使用中性/细描边控件，不使用大面积蓝色网页 CTA；compact rail 仍要保留新建、会话菜单和连接/断开入口。
- macOS sidebar 顶部必须给系统 traffic-light 标题栏让出空间，避免新建会话按钮与窗口控制按钮重叠。
- scoped CSS 的 macOS 分支必须使用 `:global(:root[data-platform="macos"] .selector)`，避免把组件规则编译到 `:root`。
