# file_svc.go 技术说明

## 文件定位
- 源文件: `internal/service/file_svc.go`
- Wails 绑定的本地文件上传、下载、预览服务。

## 核心职责
- 通过 `SandboxService.Manager()` 获取当前 sandbox manager。
- 上传/下载使用 `FileTransfer` 直接 SFTP 到当前 sandbox workspace。
- 文件列表和预览通过 `RemoteOperator` 在 sandbox runtime 内执行。
- `GetWorkspaceInfo` 返回当前 SSH、sandbox、runtime、workspace 路径、active container registry ID、文件数量和大小。
- `CleanupSandboxTmp` 只清理当前 active sandbox 的 `tmp` 目录。
- 文件浏览和 workspace metadata 会通过 ChatService 的 runtime worktree state 解析当前 active session workspace；如果当前 session 进入 worktree，文件树会跟随 worktree。

## 维护要点
- `workspacePath` 先取 active runtime 的真实 workspace 作为默认值，再按 active session 的 runtime worktree state 覆盖；旧 `/workspace` 会由 transfer/operator 映射。
- tmp 清理必须走 runtime 的路径守卫，不能复用 workspace 文件删除逻辑。
