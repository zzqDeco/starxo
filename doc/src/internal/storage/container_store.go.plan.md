# container_store.go 技术说明

## 文件定位
- 源文件: `internal/storage/container_store.go`
- 沙箱注册表持久化存储。类型名保留 ContainerStore 以兼容现有服务层。

## 核心职责
- 主存储文件改为 `~/.starxo/sandboxes.json`。
- 首次读取时若新文件不存在，会读取旧 `containers.json`，把 Docker 记录迁移为 `unavailable`，并保存到 `sandboxes.json`。
- `RegisteredDockerIDs` 保留方法名，但返回 runtime IDs，用于避免清理当前已注册 sandbox。
