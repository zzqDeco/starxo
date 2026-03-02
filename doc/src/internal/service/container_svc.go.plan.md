# container_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/container_svc.go
- 文档文件: doc/src/internal/service/container_svc.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `ContainerService`，负责管理已注册容器的生命周期操作。提供容器列表查看、状态刷新、启动、停止和销毁功能。与 `SandboxService` 协作，对于当前活动容器使用 SandboxManager 进行操作，对于非活动容器直接操作容器注册表。状态刷新通过 Docker inspect 命令检查容器的实际运行状态。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `ListContainers()`、`RefreshContainerStatus(containerRegID)`、`StopContainer(containerRegID)`、`StartContainer(containerRegID)`、`DestroyContainer(containerRegID)`
  - 依赖注入: `storage.ContainerStore`、`SandboxService`
- 输出结果:
  - `ListContainers`: 返回 `[]model.Container`
  - `RefreshContainerStatus`: 返回更新后的 `*model.Container`，状态可为 Running/Stopped/Destroyed/Unknown
  - `StopContainer`/`StartContainer`/`DestroyContainer`: 执行容器操作并更新注册表

## 4. 关键实现细节
- 结构体/接口定义:
  - `ContainerService`: 容器服务结构体，包含 Wails 上下文、ContainerStore、SandboxService 引用、SessionService 引用
- 导出函数/方法:
  - `NewContainerService(containerStore, sandboxService) *ContainerService`: 构造函数
  - `SetContext(ctx)`: 设置 Wails 上下文
  - `SetSessionService(svc)`: 设置 SessionService 引用（销毁容器时更新所属会话）
  - `ListContainers() ([]model.Container, error)`: 列出所有已注册容器
  - `RefreshContainerStatus(containerRegID) (*model.Container, error)`: 刷新容器实际状态，通过 `docker.InspectContainer` 检查容器是否存在、是否运行
  - `StopContainer(containerRegID) error`: 停止容器，如果是活动容器则通过 SandboxManager 停止
  - `StartContainer(containerRegID) error`: 启动容器，委托给 `sandboxService.ConnectExisting` 进行重连
  - `DestroyContainer(containerRegID) error`: 销毁容器，活动容器使用 `DisconnectAndDestroy`，非活动容器直接从注册表移除。销毁后同步更新所属会话的 Containers 列表（通过 SessionService 调用 `RemoveContainer`）
- Wails 绑定方法: `ListContainers`、`RefreshContainerStatus`、`StopContainer`、`StartContainer`、`DestroyContainer`
- 事件发射: 无直接事件发射（通过 SandboxService 间接触发）

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/model`: Container、ContainerRunning/Stopped/Destroyed/Unknown 状态常量
  - `starxo/internal/storage`: ContainerStore
  - 同包 `service`: SandboxService（获取 Manager、Docker、ActiveContainerRegID）
- 外部依赖: 无
- 关键配置: 无

## 6. 变更影响面
- 容器操作逻辑依赖 `SandboxService` 的连接状态和 Docker 管理器
- `RefreshContainerStatus` 的状态判断逻辑影响前端容器面板的显示
- `DestroyContainer` 对非活动容器仅从注册表移除但不实际销毁 Docker 容器（需要 SSH 连接到目标主机）
- `StartContainer` 通过 `ConnectExisting` 实现，会影响当前活动的沙箱连接
- 容器状态模型变更需同步 `model.Container` 和前端类型定义

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `DestroyContainer` 销毁后自动更新所属会话的容器列表，保证父子关系一致性。非活动容器仅从注册表移除，未来可考虑通过独立 SSH 连接执行 `docker rm` 命令。
- `RefreshContainerStatus` 依赖已有的 SandboxManager 连接，如果连接到不同的 SSH 主机则无法检查状态，标记为 Unknown。
- `StopContainer` 对活动容器的停止操作需要已有的 SandboxManager，如果 manager 为 nil 则仅更新注册表状态。
- 可考虑添加批量状态刷新功能以减少前端的 N 次 API 调用。
