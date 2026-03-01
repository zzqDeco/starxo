# ssh.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/sandbox/ssh.go
- 文档文件: doc/src/internal/sandbox/ssh.go.plan.md
- 文件类型: Go 源码
- 所属模块: sandbox

## 2. 核心职责
- `SSHClient` 封装了与远程服务器的 SSH 连接，支持密码认证、私钥认证、SSH Agent 认证以及默认密钥文件自动检测。提供命令执行（`RunCommand`）、连接管理（`Connect`/`Close`）和底层客户端访问（`GetClient`/`NewSFTPClient`）等能力。所有方法通过 `sync.Mutex` 保证线程安全。该客户端是整个沙箱子系统（Docker 管理、文件传输、环境初始化）的基础通信层。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `config.SSHConfig`（主机、端口、用户名、密码、私钥路径或内容）、`context.Context`、要执行的 shell 命令字符串
- 输出结果: `RunCommand` 返回 stdout、stderr、exitCode 和 error；`Connect`/`Close` 返回 error；`GetClient` 返回底层 `*ssh.Client`；`IsConnected` 返回 bool

## 4. 关键实现细节
- 结构体/接口定义:
  - `SSHClient` — SSH 客户端包装器，持有 `config.SSHConfig`、`*ssh.Client` 和 `sync.Mutex`
- 导出函数/方法:
  - `NewSSHClient(cfg config.SSHConfig) *SSHClient` — 创建 SSH 客户端
  - `Connect(ctx) error` — 建立 SSH 连接，支持多种认证方式（优先级：SSH Agent > 显式私钥 > 密码 > 默认密钥文件）
  - `RunCommand(ctx, cmd) (stdout, stderr, exitCode, error)` — 通过 SSH 执行远程命令，支持 context 取消
  - `NewSFTPClient() (*ssh.Client, error)` — 返回底层 SSH 客户端供 SFTP 使用
  - `Close() error` — 关闭 SSH 连接
  - `IsConnected() bool` — 检查连接状态
  - `GetClient() *ssh.Client` — 获取底层 SSH 客户端
- 私有方法:
  - `buildAuthMethods()` — 构建认证方法列表
  - `trySSHAgent()` — 尝试连接 SSH Agent（兼容 Windows OpenSSH Agent 命名管道和 Unix ssh-agent）
  - `tryParseKey(key)` — 解析私钥（支持文件路径和 PEM 内容）
  - `tryDefaultKeys()` — 自动检测 `~/.ssh/id_ed25519`、`id_rsa`、`id_ecdsa`
- 工具函数:
  - `isExitError(err, target)` — 判断是否为 SSH ExitError 并提取退出码
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config` — 使用 `SSHConfig`
- 外部依赖:
  - `golang.org/x/crypto/ssh` — SSH 协议实现
  - `golang.org/x/crypto/ssh/agent` — SSH Agent 协议
  - `bytes`、`context`、`fmt`、`net`、`os`、`path/filepath`、`runtime`、`strconv`、`strings`、`sync`、`time`（标准库）
- 关键配置: `config.SSHConfig`（Host、Port、User、Password、PrivateKey）

## 6. 变更影响面
- `internal/sandbox/manager.go` — SandboxManager 创建并管理 SSHClient 实例
- `internal/sandbox/docker.go` — RemoteDockerManager 依赖 SSHClient 执行远程 Docker 命令
- `internal/sandbox/transfer.go` — FileTransfer 依赖 SSHClient 的 `GetClient()` 创建 SFTP 客户端
- `internal/sandbox/setup.go` — EnvironmentSetup 依赖 SSHClient 执行远程环境检查/安装命令
- `internal/sandbox/operator.go` — 间接依赖（通过 RemoteDockerManager）

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `HostKeyCallback` 当前使用 `InsecureIgnoreHostKey()`，生产环境应考虑主机密钥验证。
- 认证方式优先级变更需同步更新文档和用户指南。
- Windows 平台 SSH Agent 连接使用命名管道 `\\.\pipe\openssh-ssh-agent`，如需支持其他 Agent（如 Pageant）需在 `trySSHAgent` 中扩展。
