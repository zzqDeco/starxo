# containerStore.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/stores/containerStore.ts
- 文档文件: doc/src/frontend/src/stores/containerStore.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/stores (状态管理)

## 2. 核心职责
- Pinia store，管理容器全生命周期：列表查看、**新建容器**、**激活/取消激活**（切换活跃容器）、启动、停止、销毁。跟踪活跃容器 ID 和创建进度。
- 提供按会话筛选的容器视图（当前会话容器 vs 其他容器）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 通过 Wails 绑定调用 Go 后端 `ContainerService` 方法；App.vue 通过事件驱动调用 `setActiveContainer`/`clearActiveContainer`/`updateContainerProgress`
- 输出结果: 响应式容器列表和活跃容器状态，供 ContainerPanel.vue 消费

## 4. 关键实现细节
- **响应式状态**:
  - `containers: Ref<ContainerInfo[]>` — 全部容器列表
  - `loading: Ref<boolean>` — 加载状态
  - `activeContainerID: Ref<string>` — 当前活跃容器 ID（由后端事件驱动更新）
  - `creatingContainer: Ref<boolean>` — 是否正在创建容器
  - `containerProgress: Ref<number>` — 创建进度百分比
  - `containerStep: Ref<string>` — 创建进度步骤描述
- **计算属性**:
  - `activeSessionContainers` — 当前会话的容器
  - `otherContainers` — 其他会话的容器
- **操作方法**:
  - `loadContainers()` — 获取全部容器列表
  - `refreshStatus(id)` — 刷新单个容器状态
  - `createContainer()` — 新建容器（检查 SSH 已连接，调用 `CreateContainer()`）
  - `activateContainer(id)` — 激活容器（调用 `ActivateContainer(id)`）
  - `deactivateContainer()` — 取消激活（调用 `DeactivateContainer()`）
  - `startContainer(id)` / `stopContainer(id)` / `destroyContainer(id)` — 容器生命周期操作
  - `setActiveContainer(id)` / `clearActiveContainer()` — 事件驱动的活跃容器状态更新
  - `updateContainerProgress(step, percent)` — 更新创建进度

## 5. 依赖关系
- 内部依赖: `@/types/session` (ContainerInfo)、`./sessionStore`、`./connectionStore`
- 外部依赖: `pinia` (defineStore)、`vue` (ref, computed)
- Wails 绑定: `wailsjs/go/service/ContainerService` (ListContainers, RefreshContainerStatus, StartContainer, StopContainer, DestroyContainer, CreateContainer, ActivateContainer, DeactivateContainer)

## 6. 变更影响面
- 被 `ContainerPanel.vue` 和 `App.vue` 使用
- `activeContainerID` 由 `container:ready`、`container:activated`、`container:deactivated` 事件驱动
- `createContainer()` 依赖 `connectionStore.sshConnected`
- 容器操作方法依赖后端 `ContainerService`

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 后端 `ContainerService` 新增方法时需同步添加对应操作。
- 活跃容器 ID 完全由后端事件驱动，不应在前端直接设置（除通过 `setActiveContainer`/`clearActiveContainer`）。
