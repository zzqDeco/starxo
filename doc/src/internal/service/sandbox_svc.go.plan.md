# sandbox_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/sandbox_svc.go
- 文档文件: doc/src/internal/service/sandbox_svc.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `SandboxService`，负责管理沙箱（SSH + Docker 容器）的完整生命周期。提供创建新容器并连接（`Connect`）、重连已注册容器（`ConnectExisting`）、断开连接保留容器（`Disconnect`）、断开并销毁容器（`DisconnectAndDestroy`）等操作。同时管理容器注册表、设置终端输出转发、启动健康检查监控，并通过回调通知 ChatService 和 SessionService 关于沙箱状态变更。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `Connect()`、`ConnectExisting(containerRegID)`、`Disconnect()`、`DisconnectAndDestroy()`、`GetStatus()`
  - 依赖注入: `config.Store`、`storage.ContainerStore`
- 输出结果:
  - Wails 事件发射: `sandbox:progress`（连接进度）、`sandbox:ready`（连接完成）、`sandbox:disconnected`（连接丢失）、`terminal:output`（终端输出）
  - 回调通知: `onConnect`（沙箱连接后）、`onContainerBound`（容器绑定后）
  - 容器注册表更新

## 4. 关键实现细节
- 结构体/接口定义:
  - `SandboxService`: 沙箱服务结构体，包含 Wails 上下文、SandboxManager 实例、配置存储、容器存储、连接回调、容器绑定回调、当前活动容器注册 ID
- 导出函数/方法:
  - `NewSandboxService(store, containerStore) *SandboxService`: 构造函数
  - `SetContext(ctx)`: 设置 Wails 上下文
  - `SetOnConnect(fn)`: 注册连接成功回调
  - `SetOnContainerBound(fn)`: 注册容器绑定回调
  - `Connect() error`: 创建新容器并连接，注册到容器存储，生成 UUID 短 ID，排除已注册容器避免清理冲突
  - `ConnectExisting(containerRegID) error`: 重连已注册容器，覆盖 SSH 配置为容器存储的信息
  - `Disconnect() error`: 关闭 SSH 但保留容器
  - `DisconnectAndDestroy() error`: 停止并移除容器，从注册表中删除
  - `GetStatus() SandboxStatusDTO`: 获取当前沙箱连接状态
  - `Manager() *sandbox.SandboxManager`: 返回底层 SandboxManager
  - `ActiveContainerRegID() string`: 返回当前活动容器的注册 ID
  - `StartHealthMonitor(ctx)`: 启动后台健康检查 goroutine，每 30 秒执行 `echo ping` 命令
- Wails 绑定方法: `Connect`、`ConnectExisting`、`Disconnect`、`DisconnectAndDestroy`、`GetStatus`
- 事件发射: `sandbox:progress`、`sandbox:ready`、`sandbox:disconnected`、`terminal:output`

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config`: Store（应用配置）
  - `starxo/internal/model`: Container、ContainerRunning 等状态常量
  - `starxo/internal/sandbox`: SandboxManager、NewSandboxManager
  - `starxo/internal/storage`: ContainerStore
- 外部依赖:
  - `github.com/google/uuid`: 生成容器注册 ID
  - `github.com/wailsapp/wails/v2/pkg/runtime` (wailsruntime): EventsEmit
- 关键配置: 通过 `config.Store` 获取 SSH 和 Docker 配置

## 6. 变更影响面
- `Connect`/`ConnectExisting` 的行为变更影响 ChatService 的沙箱依赖
- `onConnect` 回调变更影响 ChatService 的沙箱更新
- `onContainerBound` 回调变更影响 SessionService 的容器绑定
- 容器注册逻辑变更影响 ContainerService 和 SessionService
- 终端输出转发变更影响前端终端组件
- 健康检查频率和策略变更影响断线检测灵敏度

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `Connect` 方法中的容器排除逻辑（`excludeIDs`）确保不会误清理其他会话的容器，修改时需谨慎。
- `ConnectExisting` 覆盖 SSH 配置是为了隔离全局设置变更的影响，这一设计决策应保持。
- 健康检查中的 `echo ping` 命令应保持轻量，避免增加沙箱负担。
- `setupOutputForwarding` 中的 `SetOnOutput` 回调在 goroutine 中执行，需注意线程安全。
