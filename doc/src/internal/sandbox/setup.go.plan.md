# setup.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/sandbox/setup.go
- 文档文件: doc/src/internal/sandbox/setup.go.plan.md
- 文件类型: Go 源码
- 所属模块: sandbox

## 2. 核心职责
- `EnvironmentSetup` 负责沙箱环境的自动化初始化。职责拆分为两个独立阶段：`EnsureDockerAvailable`（Docker 安装和运行检测，在 SSH 连接后调用）和 `SetupNewContainer`（容器创建全流程，在需要新容器时调用）。保留 `InitializeExisting` 用于重连已有容器。通过 `onProgress` 回调向前端报告初始化进度。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `SSHClient`（远程命令执行）、`RemoteDockerManager`（Docker 操作）、进度回调 `onProgress func(string, int)`、`context.Context`、可选的排除容器 ID 列表、已有容器的 dockerID
- 输出结果: 返回 `error` 表示初始化成败；通过 `onProgress` 回调发送步骤名称和百分比进度（0-100）；副作用包括在远程主机安装 Docker、拉取镜像、创建/启动容器、安装 Python 包

## 4. 关键实现细节
- 结构体/接口定义:
  - `EnvironmentSetup` — 环境初始化器，持有 `*SSHClient`、`*RemoteDockerManager` 和 `onProgress` 回调
- 导出函数/方法:
  - `NewEnvironmentSetup(ssh, docker, onProgress) *EnvironmentSetup` — 创建环境初始化器
  - **阶段 1 — Docker 就绪**:
    - `EnsureDockerAvailable(ctx) error` — 确保 Docker 已安装、守护进程已运行、检测 sudo 权限（ensureDockerInstalled + ensureDockerRunning + DetectSudo，进度 0-40%）
  - **阶段 2 — 容器创建**:
    - `SetupNewContainer(ctx, excludeDockerIDs) error` — 完整容器创建流程（cleanupOldContainers + EnsureImageExists + CreateContainer + installPythonPackages + createWorkspaceDir + healthCheck，进度 0-100%）
  - **重连模式**:
    - `InitializeExisting(ctx, dockerID) error` — 重连已有容器（5 步流程：Docker 检测 → 守护进程启动 → 容器状态检查 → 按需启动 → 健康检查）
  - **向后兼容**:
    - `InitializeFresh(ctx, excludeDockerIDs) error` — 组合调用 EnsureDockerAvailable + SetupNewContainer，进度重映射
    - `Initialize(ctx) error` — 委托给 `InitializeFresh(ctx, nil)`
- 私有方法:
  - `cleanupOldContainers(ctx, excludeDockerIDs)` — 清理旧的 eino-sandbox 容器（跳过排除列表和已注册容器）
  - `ensureDockerInstalled(ctx) error` — 检测并自动安装 Docker
  - `ensureDockerRunning(ctx) error` — 确保 Docker 守护进程运行
  - `installPythonPackages(ctx) error` — 在容器内安装 Python 包
  - `createWorkspaceDir(ctx) error` — 在容器内创建工作目录
  - `healthCheck(ctx) error` — 验证 Python3 可用性和工作目录存在性

## 5. 依赖关系
- 内部依赖:
  - 同包引用: `SSHClient`、`RemoteDockerManager`
- 外部依赖:
  - `context`、`fmt`、`strings`（标准库）
- 关键配置: `docker.cfg.WorkDir`（工作目录，默认 `/workspace`）、预装 Python 包列表

## 6. 变更影响面
- `internal/sandbox/manager.go` — SandboxManager 在 `EnsureDocker` 中调用 `EnsureDockerAvailable`，在 `CreateNewContainer` 中调用 `SetupNewContainer`，在 `AttachToContainer` 中调用 `InitializeExisting`
- `internal/sandbox/docker.go` — 初始化流程依赖 RemoteDockerManager 的多个方法
- 前端进度条 — `onProgress` 回调的步骤名称和百分比变更会影响前端显示

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `EnsureDockerAvailable` 和 `SetupNewContainer` 是独立的两个阶段，分别由 SandboxManager 的不同方法调用，不应直接耦合。
- Docker 自动安装脚本（`get.docker.com`）需要远程主机有网络访问能力。
- 预装 Python 包列表硬编码在 `installPythonPackages` 中，如需可配置化应提取到 `config.DockerConfig`。
