# file_svc.go 技术说明

## 文件定位
- 源文件: `internal/service/file_svc.go`
- Wails 绑定的本地文件上传、下载、预览服务。

## 核心职责
- 通过 `SandboxService.Manager()` 获取当前 sandbox manager。
- 上传/下载使用 `FileTransfer` 直接 SFTP 到当前 sandbox workspace。
- 上传成功后发出带 active sandbox registry ID、路径、source/action 和时间戳的 `workspace:changed`，让 workspace 文件树不依赖手动刷新。
- 文件列表通过 `FileTransfer.ListFiles` 走 SFTP 直接枚举 active workspace，预览仍通过 `RemoteOperator` 在 sandbox runtime 内读取并走 workspace guard。
- `GetWorkspaceInfo` 返回当前 SSH、sandbox、runtime、workspace 路径、active container registry ID、文件数量和大小。
- `CleanupSandboxTmp` 只清理当前 active sandbox 的 `tmp` 目录。
- 文件浏览和 workspace metadata 会通过 ChatService 的 runtime worktree state 解析当前 active session workspace；如果当前 session 进入 worktree，文件树会跟随 worktree。
- 上传、下载、列表、预览、tmp 清理和 workspace info 都要按 active session 的 sandbox binding 做 guard；未绑定或绑定与全局 active sandbox 不一致时，不得读取旧会话 sandbox。

## 维护要点
- `workspacePath` 先取 active runtime 的真实 workspace 作为默认值，再按 active session 的 runtime worktree state 覆盖；旧 `/workspace` 会由 transfer/operator 映射。
- tmp 清理必须走 runtime 的路径守卫，不能复用 workspace 文件删除逻辑。
- 上传前必须捕获 active sandbox registry ID，并在上传完成后用捕获值发出 `workspace:changed`，避免长上传期间切换 sandbox 后刷新错目标；事件必须携带 createdAt 便于前端处理枚举竞态。
- 文件列表错误必须带 workspace 路径上下文，前端据此显示可观测错误状态。
