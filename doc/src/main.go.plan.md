# main.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: main.go
- 文档文件: doc/src/main.go.plan.md
- 文件类型: Go 源码
- 所属模块: main

## 2. 核心职责
- 程序入口，嵌入前端静态资源（`frontend/dist`），并启动 Wails 应用。
- 配置窗口尺寸/最小尺寸/启动状态、生命周期回调（`OnStartup` / `OnShutdown`）以及前后端绑定服务。

## 3. 输入与输出
- 输入来源: `NewApp()` 创建的 App 实例、嵌入资源 `frontend/dist`
- 输出结果: 启动桌面应用；异常时打印错误信息

## 4. 关键实现细节
- `wails.Run(options.App{...})` 负责应用初始化。
- 关键窗口参数:
  - `Width: 1400`, `Height: 900`
  - `MinWidth: 1000`, `MinHeight: 600`
  - `WindowStartState: options.Maximised`（默认启动最大化）
- 生命周期:
  - `OnStartup: app.startup`
  - `OnShutdown: app.shutdown`
- 前端可调用服务绑定:
  - `chatService`, `sandboxService`, `fileService`, `settingsService`, `sessionService`, `containerService`

## 5. 依赖关系
- 内部依赖: `app.go`（`NewApp()`）
- 外部依赖:
  - `github.com/wailsapp/wails/v2`
  - `github.com/wailsapp/wails/v2/pkg/options`
  - `github.com/wailsapp/wails/v2/pkg/options/assetserver`
  - `embed`（标准库）

## 6. 变更影响面
- 修改 `Bind` 会影响前端 IPC 能力。
- 修改窗口参数会直接影响桌面端启动体验。
- `WindowStartState` 变更会影响默认窗口状态（当前为最大化）。

## 7. 维护建议
- 新增后端服务时必须加入 `Bind`。
- 若调整前端产物目录，需同步修改 `//go:embed` 路径。
- 调整窗口策略时应验证 Windows/macOS 的一致性行为。
