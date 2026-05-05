# manager.go 技术说明

## 文件定位
- 源文件: `internal/sandbox/manager.go`
- 沙箱生命周期顶层编排器。

## 核心职责
- 管理 SSH 连接、轻量 runtime、文件传输和 `RemoteOperator`。
- 生命周期为 `ConnectSSH -> EnsureRuntime -> CreateNewSandbox/AttachToSandbox -> DetachContainer -> Disconnect`。
- 保留 `EnsureDocker`、`CreateNewContainer`、`AttachToContainer`、`Docker()` 等兼容方法名，但内部全部代理到 bwrap/Seatbelt runtime。

## 关键行为
- `ConnectSSH` 只建立 SSH 和 SFTP 能力。
- `EnsureRuntime` 检测远端 bwrap/Seatbelt 可用性，不自动安装。
- `CreateNewSandbox` 创建远端持久 workspace 并创建 operator。
- `AttachToSandbox` 激活已有 workspace。
- `SSHHostPort` 暴露当前远端地址给 workspace 元信息面板。
- `DestroySandbox` 删除指定 workspace 根目录；`Disconnect` 保留 workspace。

## 维护要点
- 新代码优先使用 runtime/sandbox 命名；兼容 Docker 命名仅用于旧调用和 Wails 过渡。
- 锁内只做状态切换，长耗时远端操作应避免扩大锁范围。
