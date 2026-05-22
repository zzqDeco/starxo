# internal/service/platform_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/platform_svc.go
- 文件类型: Go Wails service

## 2. 核心职责
- 向前端暴露当前运行平台的 UI 能力信息。
- 用于前端选择平台化设计 token 和系统主题策略。

## 3. 关键接口
- `PlatformUIInfo`
  - `platform`: `macos` / `windows` / `linux` / fallback GOOS
  - `goos`: 原始 `runtime.GOOS`
  - `appearance`: 当前为 `system`
  - `supportsTranslucency`: 前端是否可尝试 translucent/material 样式
  - `supportsMica`: Windows 是否可尝试 Mica 风格
- `GetPlatformUIInfo() PlatformUIInfo`
  - 无副作用、无配置依赖，可在前端启动时调用。

## 4. 维护建议
- 新增字段后需重新生成 Wails bindings，并同步前端读取逻辑。
- 不要在该服务中读取用户敏感配置；它只描述平台 UI 能力。
