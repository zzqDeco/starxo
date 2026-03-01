# docker.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/sandbox/docker.go
- 文档文件: doc/src/internal/sandbox/docker.go.plan.md
- 文件类型: Go 源码
- 所属模块: sandbox

## 2. 核心职责
- `RemoteDockerManager` 通过 SSH 在远程主机上管理 Docker 容器。负责镜像拉取、容器创建（含内存/CPU 限制和网络隔离配置）、容器内命令执行、文件复制（宿主机与容器之间双向传输）、容器启停与销毁，以及 sudo 权限自动检测。容器命名格式为 `eino-sandbox-<uuid前8位>`，默认以 `sleep infinity` 保持运行。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `SSHClient`（用于执行远程命令）、`config.DockerConfig`（镜像名、资源限制、工作目录、网络开关）、`context.Context`、容器内执行的命令 `[]string`、文件路径参数
- 输出结果: 容器操作返回 `error`；`ExecInContainer` 返回 stdout、stderr、exitCode、error；`ContainerID()`/`ContainerName()` 返回容器标识；`IsRunning()` 返回运行状态；`InspectContainer` 返回 exists、running、error

## 4. 关键实现细节
- 结构体/接口定义:
  - `RemoteDockerManager` — 远程 Docker 管理器，持有 `SSHClient`、`config.DockerConfig`、containerID、containerName、useSudo 标志和 `sync.Mutex`
- 导出函数/方法:
  - `NewRemoteDockerManager(ssh, cfg) *RemoteDockerManager` — 创建 Docker 管理器
  - `DetectSudo(ctx)` — 检测 docker 命令是否需要 sudo
  - `EnsureImageExists(ctx) error` — 确保镜像存在，不存在则拉取
  - `CreateContainer(ctx) error` — 创建并启动新容器（含资源限制和网络配置）
  - `ExecInContainer(ctx, cmd) (stdout, stderr, exitCode, error)` — 在容器内执行命令
  - `CopyToContainer(ctx, hostPath, containerPath) error` — 从宿主机复制文件到容器
  - `CopyFromContainer(ctx, containerPath, hostPath) error` — 从容器复制文件到宿主机
  - `StopAndRemove(ctx) error` — 停止并删除容器
  - `StopContainer(ctx) error` — 仅停止容器不删除
  - `StartContainer(ctx, dockerID) error` — 启动已停止的容器
  - `InspectContainer(ctx, dockerID) (exists, running, error)` — 检查容器状态
  - `ContainerID() string` — 返回当前容器 ID
  - `ContainerName() string` — 返回当前容器名
  - `IsRunning() bool` — 判断容器是否运行中
  - `SetContainerID(id, name)` — 直接设置容器 ID 和名称（用于重连）
- 工具函数:
  - `shellQuote(s) string` — 为 shell 命令安全转义字符串（单引号包裹）
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config` — 使用 `DockerConfig`
  - 同包引用: `SSHClient`（通过其 `RunCommand` 方法执行所有远程 Docker 命令）
- 外部依赖:
  - `github.com/google/uuid` — 生成容器名称的唯一标识
  - `context`、`fmt`、`strings`、`sync`（标准库）
- 关键配置: `config.DockerConfig`（Image、MemoryLimit、CPULimit、WorkDir、Network）

## 6. 变更影响面
- `internal/sandbox/manager.go` — SandboxManager 创建并编排 RemoteDockerManager
- `internal/sandbox/operator.go` — RemoteOperator 依赖 RemoteDockerManager 执行容器内操作
- `internal/sandbox/setup.go` — EnvironmentSetup 使用 RemoteDockerManager 进行镜像拉取、容器创建和包安装
- `internal/sandbox/transfer.go` — FileTransfer 的 `UploadToContainer`/`DownloadFromContainer` 调用 `CopyToContainer`/`CopyFromContainer`

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `shellQuote` 函数是安全关键点，修改时需考虑命令注入风险。
- 容器资源限制参数变更需同步更新 `config.DockerConfig` 和前端配置界面。
- `useSudo` 标志由 `DetectSudo` 在连接时设置，不支持运行时变更。
