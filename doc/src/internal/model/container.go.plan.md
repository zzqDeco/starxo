# container.go 技术说明

## 文件定位
- 源文件: `internal/model/container.go`
- 持久化 sandbox registry 条目。类型名仍为 `Container` 是为了兼容现有 Wails/service/frontend 命名。

## 核心职责
- `ContainerStatus` 增加 `unavailable`，用于旧 Docker 记录或无法由 dockerless runtime 管理的记录。
- `Container` 新增 `RuntimeID`、`Runtime`、`WorkspacePath`。
- `DockerID` 保留为旧 JSON 兼容字段，新记录会写入与 `RuntimeID` 相同的值以平滑前端过渡。
