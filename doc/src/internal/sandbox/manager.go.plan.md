# manager.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/sandbox/manager.go
- 文档文件: doc/src/internal/sandbox/manager.go.plan.md
- 文件类型: Go 源码
- 所属模块: sandbox

## 2. 核心职责
- `SandboxManager` 是沙箱生命周期的顶层编排器，负责协调 SSH 连接、Docker 容器管理、远程命令操作、文件传输和环境初始化五个子系统。生命周期分为两个独立阶段：**SSH 连接**（`ConnectSSH` → `EnsureDocker`）和**容器管理**（`CreateNewContainer`/`AttachToContainer` → `DetachContainer`）。SSH 和容器生命周期解耦，支持在同一 SSH 连接上创建、切换、分离多个容器。所有公共方法均通过 `sync.Mutex` 保证并发安全。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `config.AppConfig`（包含 SSH 和 Docker 配置）、`context.Context`、进度回调函数 `onProgress func(string, int)`、可选的排除容器 ID 列表 `excludeDockerIDs []string`、已有容器的 `dockerID` 和 `containerName`
- 输出结果: 返回 `error` 表示操作成败；`CreateNewContainer` 额外返回 `dockerID` 和 `containerName`；通过 `onProgress` 回调向前端发送进度通知；通过 `Operator()`、`Transfer()`、`Docker()`、`SSH()` 方法暴露子系统实例供外部使用

## 4. 关键实现细节
- 结构体/接口定义:
  - `SandboxManager` — 顶层沙箱管理器，持有 `SSHClient`、`RemoteDockerManager`、`RemoteOperator`、`FileTransfer`、`EnvironmentSetup` 子系统引用及 `config.AppConfig` 和 `sync.Mutex`
- 导出函数/方法:
  - `NewSandboxManager(cfg config.AppConfig) *SandboxManager` — 根据配置创建管理器
  - **SSH 阶段方法**:
    - `ConnectSSH(ctx, onProgress) error` — 仅建立 SSH 连接并创建 FileTransfer 子系统
    - `EnsureDocker(ctx, onProgress) error` — 创建 Docker 管理器并确保 Docker 安装运行（必须在 ConnectSSH 之后调用）
  - **容器阶段方法**:
    - `CreateNewContainer(ctx, excludeIDs, onProgress) (dockerID, containerName, error)` — 创建全新容器（清理旧容器 → 拉镜像 → 创建容器 → 安装包 → 健康检查 → 创建 Operator）
    - `AttachToContainer(ctx, dockerID, name, onProgress) error` — 连接已有容器（inspect → start if stopped → 健康检查 → 创建 Operator）
    - `DetachContainer()` — 清除 operator 和 docker containerID/Name，**不关闭 SSH**，容器保持运行
  - **状态查询**:
    - `SSHConnected() bool` — 仅 SSH 连接状态
    - `HasActiveContainer() bool` — 是否有活跃容器（operator 存在且 docker 运行中）
    - `IsConnected() bool` — SSH 已连接 **且** 有活跃容器
  - **子系统访问**:
    - `Operator() *RemoteOperator` — 返回远程操作器子系统
    - `Transfer() *FileTransfer` — 返回文件传输子系统
    - `Docker() *RemoteDockerManager` — 返回 Docker 管理子系统
    - `SSH() *SSHClient` — 返回 SSH 客户端子系统
  - **完整生命周期方法**:
    - `Disconnect(ctx) error` — 关闭 SSH 连接但保留容器
    - `DisconnectAndDestroy(ctx) error` — 停止并删除容器后关闭 SSH
    - `StopContainer(ctx) error` — 仅停止容器不删除
  - **向后兼容 Legacy 方法**（内部组合新方法实现）:
    - `Connect(ctx, onProgress) error` — ConnectSSH + EnsureDocker + CreateNewContainer
    - `ConnectWithExclusions(ctx, onProgress, excludeDockerIDs) error` — 同上但支持排除列表
    - `Reconnect(ctx, dockerID, containerName, onProgress) error` — ConnectSSH + EnsureDocker + AttachToContainer
- Wails 绑定方法: 无（被上层 service 层调用）
- 事件发射: 无（通过 `onProgress` 回调通知进度）

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config` — 使用 `AppConfig`（含 `SSHConfig` 和 `DockerConfig`）
  - 同包引用: `SSHClient`、`RemoteDockerManager`、`RemoteOperator`、`FileTransfer`、`EnvironmentSetup`
- 外部依赖:
  - `context`、`fmt`、`sync`（标准库）
- 关键配置: `config.AppConfig`（SSH 主机/端口/认证、Docker 镜像/资源限制/工作目录）

## 6. 变更影响面
- `internal/service/sandbox_svc.go` — 服务层通过各阶段方法编排 SSH 和容器生命周期
- `internal/agent/` — Agent 层通过 `SandboxManager.Operator()` 获取命令执行能力
- `app.go` — 回调连接层间接依赖 Manager 状态
- `internal/sandbox/` 包内所有子系统文件 — 编排逻辑变更可能影响调用顺序和错误处理

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- SSH 和容器生命周期已解耦，新增方法时须明确属于哪个阶段。
- 新增生命周期方法时须确保加锁顺序一致，防止死锁。
- 错误处理需保证资源清理：SSH 连接失败时清空已创建的子系统引用。
- Legacy 方法仅为向后兼容，新代码应使用分阶段方法。
