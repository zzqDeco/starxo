# sandbox_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/sandbox_svc.go
- 文档文件: doc/src/internal/service/sandbox_svc.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `SandboxService`，管理 SSH 连接和容器两个**独立的生命周期**。SSH 连接通过 `ConnectSSH`/`DisconnectSSH` 管理；容器通过 `CreateAndActivateContainer`/`ActivateContainer`/`DeactivateContainer` 管理。支持在同一 SSH 连接上创建和切换多个容器，同一时间只有一个活跃容器供 Agent 使用。通过回调通知 ChatService 和 SessionService 关于状态变更。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `ConnectSSH()`、`DisconnectSSH()`、`CreateAndActivateContainer()`、`ActivateContainer(regID)`、`DeactivateContainer()`、`GetStatus()`
  - 依赖注入: `config.Store`、`storage.ContainerStore`
- 输出结果:
  - Wails 事件发射:
    - SSH 阶段: `ssh:progress`（连接进度）、`ssh:connected`（连接完成）、`ssh:disconnected`（连接丢失）
    - 容器阶段: `container:progress`（创建/激活进度）、`container:ready`（新容器就绪）、`container:activated`（容器切换完成）、`container:deactivated`（容器分离）
    - 终端: `terminal:output`（终端输出转发）
  - 回调通知: `onConnect`（容器激活后）、`onContainerBound`（容器绑定到会话）、`onContainerDeactivated`（容器分离时清除 ChatService sandbox）

## 4. 关键实现细节
- 结构体/接口定义:
  - `SandboxService`: 持有 Wails 上下文、SandboxManager、配置存储、容器存储、SessionService 引用、三个回调（onConnect、onContainerBound、onContainerDeactivated）、activeContainerRegID
- 导出函数/方法:
  - `NewSandboxService(store, containerStore) *SandboxService`: 构造函数
  - `SetContext(ctx)`: 设置 Wails 上下文
  - `SetOnConnect(fn)`: 注册容器激活后回调
  - `SetOnContainerBound(fn)`: 注册容器绑定到会话的回调
  - `SetOnContainerDeactivated(fn)`: 注册容器分离回调（新增）
  - `SetSessionService(svc)`: 设置 SessionService 引用
  - **SSH 生命周期**:
    - `ConnectSSH() error`: 建立 SSH 连接 + EnsureDocker，发射 `ssh:progress` 和 `ssh:connected` 事件
    - `DisconnectSSH() error`: 分离活跃容器 + 关闭 SSH，发射 `ssh:disconnected` 事件
  - **容器生命周期**:
    - `CreateAndActivateContainer() error`: 创建新容器 + 注册 + 激活，发射 `container:progress` 和 `container:ready` 事件
    - `ActivateContainer(containerRegID) error`: 切换到已注册容器（验证 SSH Host 匹配），发射 `container:activated` 事件
    - `DeactivateContainer() error`: 分离活跃容器（不停止），发射 `container:deactivated` 事件
  - **状态查询**:
    - `GetStatus() SandboxStatusDTO`: 返回 SSH 连接状态、Docker 可用性、活跃容器信息
    - `Manager() *sandbox.SandboxManager`: 返回底层 SandboxManager
    - `ActiveContainerRegID() string`: 返回当前活跃容器注册 ID
  - **健康监控**:
    - `StartHealthMonitor(ctx)`: 双模式健康检查 — 无容器时 SSH ping，有容器时 operator ping
  - **向后兼容 Legacy 方法**:
    - `Connect() error`: 委托 ConnectSSH + CreateAndActivateContainer
    - `ConnectExisting(regID) error`: 如 SSH 未连接先 ConnectSSH，再 ActivateContainer
    - `Disconnect() error`: 委托 DisconnectSSH
    - `DisconnectAndDestroy() error`: 停止并移除活跃容器
- Wails 绑定方法: `ConnectSSH`、`DisconnectSSH`、`CreateAndActivateContainer`、`ActivateContainer`、`DeactivateContainer`、`GetStatus`、`Connect`、`ConnectExisting`、`Disconnect`、`DisconnectAndDestroy`
- 事件发射: `ssh:progress`、`ssh:connected`、`ssh:disconnected`、`container:progress`、`container:ready`、`container:activated`、`container:deactivated`、`terminal:output`

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config`: Store
  - `starxo/internal/model`: Container、ContainerRunning 等状态常量
  - `starxo/internal/sandbox`: SandboxManager
  - `starxo/internal/storage`: ContainerStore
- 外部依赖:
  - `github.com/google/uuid`: 生成容器注册 ID
  - `github.com/wailsapp/wails/v2/pkg/runtime` (wailsruntime): EventsEmit

## 6. 变更影响面
- SSH/容器事件变更影响 App.vue 事件监听器
- `onConnect` 回调影响 ChatService 的沙箱更新
- `onContainerBound` 回调影响 SessionService 的容器绑定
- `onContainerDeactivated` 回调影响 ChatService 清除 sandbox 引用
- 容器注册逻辑影响 ContainerService 和 SessionService
- 健康检查双模式：无容器时 SSH 级别检查，有容器时 operator 级别检查

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- SSH 和容器是独立生命周期，事件命名空间已分离（`ssh:*` vs `container:*`），修改时保持一致。
- `ActivateContainer` 会校验容器的 SSH Host 与当前连接是否匹配，跨主机需先断开再重连。
- 健康检查中无容器模式使用 `ssh.RunCommand`，有容器模式使用 `op.RunCommand`，注意两者接口不同。
