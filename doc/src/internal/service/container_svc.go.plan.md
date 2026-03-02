# container_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/container_svc.go
- 文档文件: doc/src/internal/service/container_svc.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `ContainerService`，负责管理已注册容器的完整生命周期操作。提供容器列表查看、状态刷新、启动、停止、销毁功能，以及**新建容器**、**激活（切换）容器**和**取消激活**操作。与 `SandboxService` 协作完成容器操作。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `ListContainers()`、`RefreshContainerStatus(containerRegID)`、`StopContainer(containerRegID)`、`StartContainer(containerRegID)`、`DestroyContainer(containerRegID)`、`CreateContainer()`、`ActivateContainer(containerRegID)`、`DeactivateContainer()`
  - 依赖注入: `storage.ContainerStore`、`SandboxService`、`SessionService`
- 输出结果:
  - `ListContainers`: 返回 `[]model.Container`
  - `RefreshContainerStatus`: 返回更新后的 `*model.Container`
  - 其他方法: 执行操作并返回 `error`

## 4. 关键实现细节
- 结构体/接口定义:
  - `ContainerService`: 持有 Wails 上下文、ContainerStore、SandboxService 引用、SessionService 引用
- 导出函数/方法:
  - `NewContainerService(containerStore, sandboxService) *ContainerService`: 构造函数
  - `SetContext(ctx)`: 设置 Wails 上下文
  - `SetSessionService(svc)`: 设置 SessionService 引用
  - **容器查询**:
    - `ListContainers() ([]model.Container, error)`: 列出所有已注册容器
    - `RefreshContainerStatus(containerRegID) (*model.Container, error)`: 通过 Docker inspect 刷新容器实际状态
  - **容器生命周期**:
    - `CreateContainer() error`: 创建新容器，委托 `sandboxService.CreateAndActivateContainer()`
    - `ActivateContainer(containerRegID) error`: 激活（切换到）已注册容器，委托 `sandboxService.ActivateContainer()`
    - `DeactivateContainer() error`: 取消激活当前容器，委托 `sandboxService.DeactivateContainer()`
    - `StartContainer(containerRegID) error`: 启动已停止容器，委托 `sandboxService.ConnectExisting()`
    - `StopContainer(containerRegID) error`: 停止容器
    - `DestroyContainer(containerRegID) error`: 销毁容器，更新所属会话的容器列表
- Wails 绑定方法: `ListContainers`、`RefreshContainerStatus`、`StopContainer`、`StartContainer`、`DestroyContainer`、`CreateContainer`、`ActivateContainer`、`DeactivateContainer`
- 事件发射: 无直接事件发射（通过 SandboxService 间接触发）

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/model`: Container 及状态常量
  - `starxo/internal/storage`: ContainerStore
  - 同包 `service`: SandboxService、SessionService
- 外部依赖: 无

## 6. 变更影响面
- `CreateContainer`/`ActivateContainer`/`DeactivateContainer` 是前端容器面板的新入口
- 容器操作逻辑依赖 `SandboxService` 的 SSH 连接状态
- `DestroyContainer` 对非活动容器仅从注册表移除但不实际销毁 Docker 容器
- `StartContainer` 通过 `ConnectExisting` 实现，可能触发 SSH 重连

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `CreateContainer`、`ActivateContainer`、`DeactivateContainer` 是对 SandboxService 方法的薄封装，逻辑变更应在 SandboxService 中进行。
- 前端 ContainerPanel.vue 直接调用这些方法，接口签名变更需同步前端。
