# setup.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/sandbox/setup.go
- 文档文件: doc/src/internal/sandbox/setup.go.plan.md
- 文件类型: Go 源码
- 所属模块: sandbox

## 2. 核心职责
- `EnvironmentSetup` 负责沙箱环境的自动化初始化，包括 Docker 安装检测与自动安装、Docker 守护进程启动、旧容器清理、镜像拉取、容器创建、Python 包安装、工作目录创建和健康检查。支持两种初始化模式：`InitializeFresh`（全新环境初始化，执行完整流程）和 `InitializeExisting`（重连已有容器，跳过包安装等步骤）。通过 `onProgress` 回调向前端报告初始化进度。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `SSHClient`（远程命令执行）、`RemoteDockerManager`（Docker 操作）、进度回调 `onProgress func(string, int)`、`context.Context`、可选的排除容器 ID 列表、已有容器的 dockerID
- 输出结果: 返回 `error` 表示初始化成败；通过 `onProgress` 回调发送步骤名称和百分比进度（0-100）；副作用包括在远程主机安装 Docker、拉取镜像、创建/启动容器、安装 Python 包

## 4. 关键实现细节
- 结构体/接口定义:
  - `EnvironmentSetup` — 环境初始化器，持有 `*SSHClient`、`*RemoteDockerManager` 和 `onProgress` 回调
- 导出函数/方法:
  - `NewEnvironmentSetup(ssh, docker, onProgress) *EnvironmentSetup` — 创建环境初始化器
  - `InitializeFresh(ctx, excludeDockerIDs) error` — 全新环境初始化（8 步流程：Docker 检测 -> 守护进程启动 -> 旧容器清理 -> 镜像拉取 -> 容器创建 -> Python 包安装 -> 工作目录创建 -> 健康检查）
  - `InitializeExisting(ctx, dockerID) error` — 重连已有容器（5 步流程：Docker 检测 -> 守护进程启动 -> 容器状态检查 -> 按需启动 -> 健康检查）
  - `Initialize(ctx) error` — 向后兼容方法，委托给 `InitializeFresh(ctx, nil)`
- 私有方法:
  - `cleanupOldContainers(ctx, excludeDockerIDs)` — 清理旧的 eino-sandbox 容器（跳过排除列表和已注册容器，支持短/长 ID 匹配）
  - `ensureDockerInstalled(ctx) error` — 检测并自动安装 Docker（使用 `get.docker.com` 脚本）
  - `ensureDockerRunning(ctx) error` — 确保 Docker 守护进程运行（尝试 systemctl/service 两种启动方式）
  - `installPythonPackages(ctx) error` — 在容器内安装 Python 包（pandas、numpy、matplotlib、openpyxl）
  - `createWorkspaceDir(ctx) error` — 在容器内创建工作目录
  - `healthCheck(ctx) error` — 验证 Python3 可用性和工作目录存在性
- Wails 绑定方法: 无
- 事件发射: 通过 `onProgress` 回调报告初始化进度

## 5. 依赖关系
- 内部依赖:
  - 同包引用: `SSHClient`（远程命令执行）、`RemoteDockerManager`（Docker 操作：`DetectSudo`、`EnsureImageExists`、`CreateContainer`、`ExecInContainer`、`InspectContainer`、`StartContainer`、`dockerCmd`）
- 外部依赖:
  - `context`、`fmt`、`strings`（标准库）
- 关键配置: `docker.cfg.WorkDir`（工作目录，默认 `/workspace`）、预装 Python 包列表

## 6. 变更影响面
- `internal/sandbox/manager.go` — SandboxManager 在 `Connect`/`Reconnect` 中调用 EnvironmentSetup
- `internal/sandbox/docker.go` — 初始化流程依赖 RemoteDockerManager 的多个方法
- 前端进度条 — `onProgress` 回调的步骤名称和百分比变更会影响前端显示

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- Docker 自动安装脚本（`get.docker.com`）需要远程主机有网络访问能力。
- 预装 Python 包列表硬编码在 `installPythonPackages` 中，如需可配置化应提取到 `config.DockerConfig`。
- `cleanupOldContainers` 采用尽力清理策略（错误不中断流程），确保旧容器不会无限积累。
- 健康检查目前仅验证 Python3 和工作目录，如新增运行时依赖需同步扩展检查项。
