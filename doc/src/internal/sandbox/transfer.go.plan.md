# transfer.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/sandbox/transfer.go
- 文档文件: doc/src/internal/sandbox/transfer.go.plan.md
- 文件类型: Go 源码
- 所属模块: sandbox

## 2. 核心职责
- `FileTransfer` 提供基于 SFTP 协议的文件传输能力，支持本地与远程主机之间的双向文件传输（`UploadFile`/`DownloadFile`），以及本地与 Docker 容器之间的双向文件传输（`UploadToContainer`/`DownloadFromContainer`）。容器文件传输采用两步中转策略：先通过 SFTP 将文件传至远程主机临时目录，再通过 `docker cp` 复制到容器内（反向同理）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `SSHClient`（SSH 连接）、`context.Context`、本地文件路径、远程文件路径、容器内文件路径、`RemoteDockerManager`（容器文件传输时需要）
- 输出结果: 所有方法返回 `error`；副作用为在本地/远程/容器之间创建或覆盖文件

## 4. 关键实现细节
- 结构体/接口定义:
  - `FileTransfer` — 文件传输器，持有 `*SSHClient`
- 导出函数/方法:
  - `NewFileTransfer(ssh) *FileTransfer` — 创建文件传输器
  - `UploadFile(ctx, localPath, remotePath) error` — SFTP 上传本地文件到远程主机
  - `DownloadFile(ctx, remotePath, localPath) error` — SFTP 下载远程文件到本地
  - `UploadToContainer(ctx, localPath, containerPath, docker) error` — 上传本地文件到容器（SFTP -> 远程主机 /tmp -> docker cp）
  - `DownloadFromContainer(ctx, containerPath, localPath, docker) error` — 从容器下载文件到本地（docker cp -> 远程主机 /tmp -> SFTP）
- 私有方法:
  - `newSFTP() (*sftp.Client, error)` — 基于 SSH 连接创建 SFTP 客户端
- 工具函数:
  - `sanitizeFileName(name) string` — 清理文件名中的路径分隔符和空格
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - 同包引用: `SSHClient`（`GetClient()` 方法获取底层 SSH 客户端）、`RemoteDockerManager`（`CopyToContainer`/`CopyFromContainer` 方法）、`shellQuote`（命令参数转义）
- 外部依赖:
  - `github.com/pkg/sftp` — SFTP 协议客户端
  - `context`、`fmt`、`io`、`os`、`path/filepath`、`strings`（标准库）
- 关键配置: 临时文件路径前缀 `/tmp/eino-upload-` 和 `/tmp/eino-download-`

## 6. 变更影响面
- `internal/sandbox/manager.go` — SandboxManager 创建并暴露 FileTransfer 实例
- `app.go` — 前端绑定层可能通过 `Transfer()` 方法访问文件传输功能
- `internal/sandbox/docker.go` — 容器文件传输依赖 `CopyToContainer`/`CopyFromContainer`

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 容器文件中转使用 `/tmp` 目录下的临时文件，传输完成后会清理，但异常中断时可能残留需注意。
- `UploadFile` 自动创建远程目录（`MkdirAll`），`DownloadFile` 自动创建本地目录（`os.MkdirAll`）。
- SFTP 客户端每次传输都重新创建，如有大量文件传输场景需考虑连接复用优化。
