# store.go 技术说明

## 文件定位
- 源文件: `internal/config/store.go`
- 负责 `~/.starxo/config.json` 的读取、保存和更新。

## 核心职责
- `Load` 从磁盘读取配置，缺失时使用 `DefaultConfig`。
- 读取旧配置时，如没有 `sandbox` 块但存在 `docker` 块，会执行 Docker 到 Sandbox 的兼容迁移。
- `Update` 保存前统一调用 `NormalizeAppConfig`，避免继续写出运行时已废弃的 Docker 配置。
