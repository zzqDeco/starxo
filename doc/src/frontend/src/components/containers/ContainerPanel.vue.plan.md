# ContainerPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/containers/ContainerPanel.vue
- 文档文件: doc/src/frontend/src/components/containers/ContainerPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/containers (容器管理)

## 2. 核心职责
- 容器管理面板，在右侧面板 "Containers" Tab 中展示容器列表，支持查看状态、启动、停止、销毁容器。
- 分两个区域：当前会话容器（上方）和其他容器（下方可折叠）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `containerStore` (容器列表/操作)、`sessionStore` (活跃会话/会话列表)
- 输出结果: 渲染容器卡片列表，提供操作按钮触发容器生命周期管理

## 4. 关键实现细节
- **两区域布局**:
  - "当前会话"区域: 显示 `containerStore.activeSessionContainers`，无容器时提示 "无容器 — 点击连接创建"
  - "其他容器"区域: 使用 `NCollapse` 折叠显示 `containerStore.otherContainers`，仅在有数据时渲染
  - 全空状态: `containerStore.containers.length === 0` 时显示 `NEmpty`
- **容器卡片** (container-card):
  - 头部: Server 图标 + 容器名(或 ID 前8位) + 状态 NTag (running=success, stopped=warning, destroyed=error) + 活跃标记
  - 详情: 镜像名、SSH 地址端口、最后使用时间
  - 操作: Start (stopped 时) / Stop (running 时) / Refresh / Destroy (NPopconfirm 确认)
- **辅助函数**:
  - `statusType(status)` — 映射状态到 NTag type
  - `statusLabel(status)` — 映射状态到 i18n 翻译文本
  - `isActive(container)` — 判断是否为当前会话活跃容器
  - `formatTime(ts)` — 格式化时间戳（今天显示时间，其他显示日期+时间）
  - `sessionTitle(sessionID)` — 根据 sessionID 查找会话标题
- **活跃容器高亮**: `.container-card.active` 添加 cyan 边框和渐变背景

## 5. 依赖关系
- 内部依赖: `@/stores/containerStore`、`@/stores/sessionStore`、`@/types/session` (ContainerInfo)
- 外部依赖: `naive-ui` (NButton, NIcon, NEmpty, NSpin, NCollapse, NCollapseItem, NTag, NPopconfirm)、`@vicons/ionicons5` (Refresh, Play, Stop, Trash, Server)、`vue-i18n` (useI18n)、`vue` (onMounted)

## 6. 变更影响面
- 在 `MainLayout.vue` 中通过 `v-show="rightPanelTab === 'containers'"` 控制显隐
- 容器操作直接调用 `containerStore` 方法，影响后端容器状态
- 样式使用 CSS 变量，与全局主题体系一致

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增容器操作按钮时需同步 `containerStore` 和后端 `ContainerService`。
- 卡片样式参考了 `FileExplorer.vue` 和 `Sidebar.vue` 的设计模式。
- 组件在 onMounted 时自动加载容器列表。
