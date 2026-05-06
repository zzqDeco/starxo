# file_svc.go 技术说明

## 文件定位
- 源文件: `internal/service/file_svc.go`
- Wails 绑定的本地文件上传、下载、预览服务。

## 核心职责
- 通过 `SandboxService.Manager()` 获取当前 sandbox manager。
- 上传/下载使用 `FileTransfer` 直接 SFTP 到当前 sandbox workspace。
- 文件列表和预览通过 `RemoteOperator` 在 sandbox runtime 内执行。
- `GetWorkspaceInfo` 返回当前 SSH、sandbox、runtime、workspace 路径、文件数量和大小。
- `CleanupSandboxTmp` 只清理当前 active sandbox 的 `tmp` 目录。

## 维护要点
- `workspacePath` 优先取 active runtime 的真实 workspace，旧 `/workspace` 会由 transfer/operator 映射。
- tmp 清理必须走 runtime 的路径守卫，不能复用 workspace 文件删除逻辑。
