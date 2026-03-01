# main.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: main.go
- 文档文件: doc/src/main.go.plan.md
- 文件类型: Go 源码
- 所属模块: main

## 2. 核心职责
- 该文件是 starxo 应用的入口点，负责嵌入前端静态资源（`frontend/dist`）并启动 Wails v2 桌面应用。它通过 `wails.Run` 配置应用窗口属性（标题、尺寸、背景色）、资源服务器、生命周期回调（`OnStartup`/`OnShutdown`），并将所有后端服务绑定到前端供 JavaScript 调用。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 嵌入的前端静态资源 (`frontend/dist`)；`NewApp()` 返回的 App 实例
- 输出结果: 启动 Wails 桌面应用窗口；错误时向标准输出打印错误信息

## 4. 关键实现细节
- 结构体/接口定义: 无（使用 `App` 结构体来自 `app.go`）
- 导出函数/方法: 无（`main` 函数为程序入口）
- Wails 绑定方法: 通过 `options.App.Bind` 绑定以下服务到前端:
  - `app.chatService` (ChatService)
  - `app.sandboxService` (SandboxService)
  - `app.fileService` (FileService)
  - `app.settingsService` (SettingsService)
  - `app.sessionService` (SessionService)
  - `app.containerService` (ContainerService)
- 事件发射: 无直接事件发射（生命周期回调委托给 `app.startup`/`app.shutdown`）

## 5. 依赖关系
- 内部依赖: 无直接 import（通过 `NewApp()` 间接依赖 `app.go`）
- 外部依赖:
  - `embed` (Go 标准库，嵌入前端资源)
  - `github.com/wailsapp/wails/v2` (Wails 框架运行时)
  - `github.com/wailsapp/wails/v2/pkg/options` (应用配置选项)
  - `github.com/wailsapp/wails/v2/pkg/options/assetserver` (静态资源服务器配置)
- 关键配置:
  - 窗口尺寸: 1400x900 (最小 1000x600)
  - 应用标题: "Starxo"
  - 背景色: RGBA(12, 14, 26, 1)

## 6. 变更影响面
- 修改 Bind 列表会影响前端可调用的后端服务范围
- 修改窗口配置会影响桌面应用的外观表现
- 修改 `OnStartup`/`OnShutdown` 回调会影响应用的初始化和关闭流程
- 修改嵌入路径 `frontend/dist` 需要同步前端构建输出目录

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增后端服务时，需在 `Bind` 切片中注册才能被前端调用。
- 前端构建输出目录变更时需同步修改 `//go:embed` 指令。
- 窗口属性修改应同时考虑不同操作系统的显示兼容性。
