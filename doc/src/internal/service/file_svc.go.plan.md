# file_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/file_svc.go
- 文档文件: doc/src/internal/service/file_svc.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `FileService`，负责处理本地机器与沙箱容器之间的文件上传和下载操作。提供原生文件选择对话框（通过 Wails runtime）、文件上传到容器、从容器下载文件、列出工作区文件和文件预览等功能。所有文件操作都通过 SandboxService 获取底层的 SandboxManager、Transfer 和 Docker 组件。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `SelectAndUploadFile()`、`UploadFile(localPath)`、`DownloadFile(containerPath)`、`ListWorkspaceFiles()`、`ReadFilePreview(containerPath)`
  - 依赖注入: `SandboxService`、`SessionService`（用于获取工作区路径）
- 输出结果:
  - `SelectAndUploadFile`/`UploadFile`: 返回 `FileInfoDTO`（文件名、容器路径、大小、是否输出文件）
  - `DownloadFile`: 弹出保存对话框并下载到本地
  - `ListWorkspaceFiles`: 返回 `[]FileInfoDTO`，通过 `find` + `stat` 命令获取文件名和大小
  - `ReadFilePreview`: 返回文件内容预览，限制 4KB

## 4. 关键实现细节
- 结构体/接口定义:
  - `FileService`: 文件服务结构体，包含 Wails 上下文、SandboxService 引用、SessionService 引用
- 导出函数/方法:
  - `NewFileService(sandbox) *FileService`: 构造函数
  - `SetContext(ctx)`: 设置 Wails 上下文
  - `SetSessionService(ss)`: 注入会话服务
  - `SelectAndUploadFile() (FileInfoDTO, error)`: 打开文件选择对话框并上传，内部调用 `UploadFile`
  - `UploadFile(localPath) (FileInfoDTO, error)`: 上传本地文件到容器工作区根目录，使用 `transfer.UploadToContainer`
  - `DownloadFile(containerPath) error`: 打开保存对话框并从容器下载文件，使用 `transfer.DownloadFromContainer`
  - `ListWorkspaceFiles() ([]FileInfoDTO, error)`: 列出工作区文件（最深 3 级），通过 `find` + `stat -c "%s %n"` 获取大小和路径
  - `ReadFilePreview(containerPath) (string, error)`: 读取文件内容预览，限制 4096 字节并添加截断提示
- 内部方法:
  - `workspacePath() string`: 从 SessionService 获取工作区路径，默认 "/workspace"
- Wails 绑定方法: `SelectAndUploadFile`、`UploadFile`、`DownloadFile`、`ListWorkspaceFiles`、`ReadFilePreview`
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - 同包 `service`: `SandboxService`（获取 Manager、Transfer、Docker）、`SessionService`（获取工作区路径）、`FileInfoDTO`（事件类型定义在 events.go）
- 外部依赖:
  - `github.com/wailsapp/wails/v2/pkg/runtime` (wailsruntime): OpenFileDialog、SaveFileDialog
- 关键配置: 文件预览限制 4096 字节，文件列表最深 3 级

## 6. 变更影响面
- 文件上传/下载逻辑依赖 `sandbox.SandboxManager` 的 `Transfer()` 和 `Docker()` 接口
- `ListWorkspaceFiles` 使用 Linux `stat` 命令格式，仅适用于 Linux 容器
- 工作区路径通过 `SessionService.GetWorkspacePath()` 获取，受会话绑定影响
- `FileInfoDTO` 定义在 `events.go` 中，字段变更需同步

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `ListWorkspaceFiles` 使用的 `stat -c "%s %n"` 是 GNU stat 格式，macOS/BSD 容器可能不兼容，如需支持需条件处理。
- 文件上传默认放在工作区根目录，大文件上传可能需要进度回调支持。
- `ReadFilePreview` 的 4KB 限制对于大文件可能不足，可考虑支持分页或按需加载。
- 文件选择对话框通过 Wails runtime 调用，仅在桌面端可用。
