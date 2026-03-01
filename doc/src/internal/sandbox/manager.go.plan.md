# manager.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/sandbox/manager.go
- 文档文件: doc/src/internal/sandbox/manager.go.plan.md
- 文件类型: Go 源码
- 所属模块: sandbox

## 2. 核心职责
- `SandboxManager` 是沙箱生命周期的顶层编排器，负责协调 SSH 连接、Docker 容器管理、远程命令操作、文件传输和环境初始化五个子系统。它提供了全新连接（`Connect`）、重连已有容器（`Reconnect`）、断开连接（`Disconnect`）和销毁容器（`DisconnectAndDestroy`）等完整的生命周期管理能力。所有公共方法均通过 `sync.Mutex` 保证并发安全。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `config.AppConfig`（包含 SSH 和 Docker 配置）、`context.Context`、进度回调函数 `onProgress func(string, int)`、可选的排除容器 ID 列表 `excludeDockerIDs []string`、重连时的 `dockerID` 和 `containerName`
- 输出结果: 返回 `error` 表示操作成败；通过 `onProgress` 回调向前端发送进度通知；通过 `Operator()`、`Transfer()`、`Docker()` 方法暴露子系统实例供外部使用

## 4. 关键实现细节
- 结构体/接口定义:
  - `SandboxManager` — 顶层沙箱管理器，持有 `SSHClient`、`RemoteDockerManager`、`RemoteOperator`、`FileTransfer`、`EnvironmentSetup` 子系统引用及 `config.AppConfig` 和 `sync.Mutex`
- 导出函数/方法:
  - `NewSandboxManager(cfg config.AppConfig) *SandboxManager` — 根据配置创建管理器
  - `Connect(ctx, onProgress) error` — 建立全新沙箱环境（SSH -> Docker -> 环境初始化 -> 操作器/传输子系统）
  - `ConnectWithExclusions(ctx, onProgress, excludeDockerIDs) error` — 同 Connect，但可排除指定容器 ID 不被清理
  - `Reconnect(ctx, dockerID, containerName, onProgress) error` — 重连已有容器
  - `Disconnect(ctx) error` — 关闭 SSH 连接但保留容器
  - `DisconnectAndDestroy(ctx) error` — 停止并删除容器后关闭 SSH
  - `StopContainer(ctx) error` — 仅停止容器不删除
  - `IsConnected() bool` — 判断沙箱是否完全连接
  - `Operator() *RemoteOperator` — 返回远程操作器子系统
  - `Transfer() *FileTransfer` — 返回文件传输子系统
  - `Docker() *RemoteDockerManager` — 返回 Docker 管理子系统
- Wails 绑定方法: 无（被上层 `app.go` 调用）
- 事件发射: 无（通过 `onProgress` 回调通知进度）

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config` — 使用 `AppConfig`（含 `SSHConfig` 和 `DockerConfig`）
  - 同包引用: `SSHClient`、`RemoteDockerManager`、`RemoteOperator`、`FileTransfer`、`EnvironmentSetup`
- 外部依赖:
  - `context`、`fmt`、`sync`（标准库）
- 关键配置: `config.AppConfig`（SSH 主机/端口/认证、Docker 镜像/资源限制/工作目录）

## 6. 变更影响面
- `internal/agent/` — Agent 层通过 `SandboxManager.Operator()` 获取命令执行能力
- `app.go` — 前端绑定层直接调用 `Connect`/`Reconnect`/`Disconnect` 等方法
- `internal/sandbox/` 包内所有子系统文件 — 编排逻辑变更可能影响调用顺序和错误处理
- `internal/config/` — 配置结构变更需同步调整

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增生命周期方法时须确保加锁顺序一致，防止死锁。
- 错误处理需保证资源清理：SSH 连接失败时清空已创建的子系统引用。
- 新增子系统时在 `Connect`/`Reconnect`/`Disconnect` 三处同步维护创建和清理逻辑。
