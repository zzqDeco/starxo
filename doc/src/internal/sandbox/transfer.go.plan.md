# transfer.go 技术说明

## 文件定位
- 源文件: `internal/sandbox/transfer.go`
- 提供本地和远端 sandbox workspace 之间的 SFTP 文件传输。

## 核心职责
- `UploadFile` / `DownloadFile` 负责本地与远端路径的 SFTP 传输。
- `ListFiles` 使用 SFTP 直接枚举远端 workspace 文件，供 WorkspacePanel 刷新文件树使用。
- workspace 根目录读取失败会返回错误；子目录在刷新期间被删除或不可读时跳过该子树，避免一个 volatile 目录清空整个文件列表。
- `UploadToContainer` / `DownloadFromContainer` 保留旧方法名，但实际上传/下载到当前 runtime workspace。
- 传输路径会映射 `/workspace/...` 到真实 workspace，并拒绝 workspace 外路径。

## 维护要点
- 不再使用 `docker cp` 或宿主机 `/tmp` 中转。
- 大文件、小文件和 workspace 列表统一走 SFTP，避免依赖 sandbox 内 Python/bwrap 命令枚举文件。
