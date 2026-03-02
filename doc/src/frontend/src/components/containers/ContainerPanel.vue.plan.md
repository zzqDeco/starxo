# ContainerPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/containers/ContainerPanel.vue
- 文档文件: doc/src/frontend/src/components/containers/ContainerPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/containers (容器管理)

## 2. 核心职责
- 容器管理面板，支持容器全生命周期操作：**新建容器**、**激活/取消激活**（切换活跃容器）、启动、停止、销毁、状态刷新。
- 分两个区域：当前会话容器（上方）和其他容器（下方可折叠）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `containerStore`（容器列表/操作/活跃容器/创建进度）、`connectionStore`（SSH 连接状态控制按钮启用）、`sessionStore`（会话信息）
- 输出结果: 渲染容器卡片列表和操作按钮

## 4. 关键实现细节
- **Header 区域**:
  - "新建容器"按钮：SSH 未连接或正在创建时禁用，调用 `containerStore.createContainer()`
  - 刷新按钮：刷新容器列表
- **创建进度**: 创建容器时显示 NProgress 进度条和步骤描述
- **容器卡片操作** (container-card):
  - **激活** (RadioButtonOn): 非活跃 + running + SSH 连接时显示，调用 `containerStore.activateContainer(id)`
  - **取消激活** (RadioButtonOff): 活跃容器显示，调用 `containerStore.deactivateContainer()`
  - Start (stopped 时) / Stop (running 且非活跃时) / Refresh / Destroy (NPopconfirm 确认)
- **空状态提示**:
  - SSH 已连接: "SSH 已连接，点击'新建容器'开始"
  - SSH 未连接: "请先连接 SSH"
- **活跃容器判断**: 通过 `containerStore.activeContainerID` 驱动（不再依赖 session.activeContainerID）

## 5. 依赖关系
- 内部依赖: `@/stores/containerStore`、`@/stores/connectionStore`、`@/stores/sessionStore`、`@/types/session` (ContainerInfo)
- 外部依赖: naive-ui (NButton, NIcon, NEmpty, NSpin, NCollapse, NCollapseItem, NTag, NPopconfirm, NProgress)、@vicons/ionicons5 (Refresh, Play, Stop, Trash, Server, Add, RadioButtonOn, RadioButtonOff)、vue-i18n、vue

## 6. 变更影响面
- 容器操作直接调用 `containerStore` 方法，影响后端容器状态
- 新建容器依赖 `connectionStore.sshConnected`
- 激活/取消激活影响 Agent 使用的容器

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增容器操作时需同步 `containerStore` 和后端 `ContainerService`。
- 活跃容器高亮和按钮显示逻辑完全由 `containerStore.activeContainerID` 驱动。
