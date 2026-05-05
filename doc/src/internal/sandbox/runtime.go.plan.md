# runtime.go 技术说明

## 文件定位
- 源文件: `internal/sandbox/runtime.go`
- 负责远端轻量沙箱运行时能力，替代原 Docker manager/setup。

## 核心职责
- `RemoteRuntimeManager` 通过 SSH 检测、安装、创建、激活、销毁和执行 bwrap/Seatbelt 沙箱。
- `runtime=auto` 按远端 `uname -s` 选择 Linux `bwrap` 或 macOS `seatbelt`。
- 每个 sandbox 是远端持久目录：`rootDir/<id>/<workDirName>`，不是后台常驻容器进程。

## 关键接口
- `Detect(ctx) (RuntimeCheckResult, error)`：检测 runtime、Python 和隔离能力。
- `Install(ctx) (RuntimeInstallResult, error)`：Linux 上显式安装 bubblewrap/Python 依赖。
- `CreateSandbox(ctx, excludeIDs)`：创建 workspace/tmp/.venv 并按需初始化 Python 包。
- `AttachSandbox(ctx, id, name, workspacePath)`：绑定已有 workspace。
- `ExecInSandbox(ctx, command)`：通过 bwrap 或 sandbox-exec 执行命令。
- `DestroySandbox(ctx, id, workspacePath)`：删除远端 workspace 根目录。

## 维护要点
- Docker fallback 不保留，`RuntimeDocker` 只用于标记旧数据不可用。
- `network=false` 使用 bwrap `--unshare-net` 或 Seatbelt network deny。
- 内存限制为 `ulimit` best-effort，不等价于 Docker/cgroup 硬配额。
