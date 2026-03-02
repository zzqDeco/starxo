# containerStore.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/stores/containerStore.ts
- 文档文件: doc/src/frontend/src/stores/containerStore.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/stores (状态管理)

## 2. 核心职责
- Pinia store，封装 ContainerService Wails 绑定，管理容器列表状态和容器生命周期操作。
- 提供按会话筛选的容器视图（当前会话容器 vs 其他容器）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 通过 Wails 绑定调用 Go 后端 `ContainerService` 的方法
- 输出结果: 响应式容器列表 `containers`，以及 `activeSessionContainers` / `otherContainers` 计算属性

## 4. 关键实现细节
- **响应式状态**:
  - `containers: Ref<ContainerInfo[]>` — 全部容器列表
  - `loading: Ref<boolean>` — 加载状态
- **计算属性**:
  - `activeSessionContainers` — 过滤 `sessionStore.activeSessionId` 匹配的容器
  - `otherContainers` — 不属于当前会话的容器
- **操作方法**:
  - `loadContainers()` — 调用 `ListContainers()` 获取全部容器，映射为 `ContainerInfo[]`
  - `refreshStatus(id)` — 调用 `RefreshContainerStatus(id)` 刷新单个容器状态
  - `startContainer(id)` — 调用 `StartContainer(id)` 启动容器后刷新列表
  - `stopContainer(id)` — 调用 `StopContainer(id)` 停止容器后刷新列表
  - `destroyContainer(id)` — 调用 `DestroyContainer(id)` 销毁容器后刷新列表
- **Go 返回值映射**: `ListContainers()` 返回的对象字段名使用驼峰命名，直接映射到 `ContainerInfo` 类型

## 5. 依赖关系
- 内部依赖: `@/types/session` (ContainerInfo)、`./sessionStore` (useSessionStore — 获取 activeSessionId)
- 外部依赖: `pinia` (defineStore)、`vue` (ref, computed)
- Wails 绑定: `wailsjs/go/service/ContainerService` (ListContainers, RefreshContainerStatus, StartContainer, StopContainer, DestroyContainer)

## 6. 变更影响面
- 被 `ContainerPanel.vue` 和 `App.vue` 使用
- 容器操作方法依赖后端 `ContainerService`，接口变更需同步
- `activeSessionContainers` 依赖 `sessionStore.activeSessionId`，会话切换时自动更新

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 后端 `ContainerService` 新增方法时需同步添加对应操作。
- 容器列表在 `sandbox:ready` 和 `session:switched` 事件中由 `App.vue` 触发刷新。
