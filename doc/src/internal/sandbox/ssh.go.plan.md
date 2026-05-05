# ssh.go 技术说明

## 文件定位
- 源文件: `internal/sandbox/ssh.go`
- 远端 SSH 通信基础层。

## 核心职责
- 支持密码、私钥、SSH Agent 和默认密钥认证。
- `RunCommand` 在远端主机执行 shell 命令并返回 stdout/stderr/exitCode。
- `GetClient` 提供底层 SSH client 给 SFTP 使用。

## 变更影响面
- `RemoteRuntimeManager` 使用 SSH 执行 bwrap/Seatbelt 检测、安装、命令运行和 workspace 管理。
- `FileTransfer` 使用 SSH client 创建 SFTP 连接。

## 维护要点
- 该层不感知 Docker 或具体 runtime。
- `HostKeyCallback` 当前仍为 `InsecureIgnoreHostKey()`，生产化需要补主机密钥校验。
