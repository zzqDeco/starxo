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
- 配置 macOS / Windows / Linux 平台专属 Wails shell options，使窗口壳跟随平台原生外观。

## 3. 输入与输出
- 输入来源: `NewApp()` 创建的 App 实例、嵌入资源 `frontend/dist`
- 输出结果: 启动桌面应用；异常时打印错误信息

## 4. 关键实现细节
- `wails.Run(options.App{...})` 负责应用初始化。
- 关键窗口参数:
  - `Width: 1400`, `Height: 900`
  - `MinWidth: 768`, `MinHeight: 600`
  - `WindowStartState: options.Maximised`（默认启动最大化）
  - macOS 使用 Wails/native default titlebar、system appearance、transparent/translucent window 和 About 信息
  - Windows 使用 system theme、Mica backdrop 和 light/dark titlebar custom theme
  - Linux 设置 app icon、program name 和 WebKit GPU policy
- 生命周期:
  - `OnStartup: app.startup`
  - `OnShutdown: app.shutdown`
- 前端可调用服务绑定:
  - `chatService`, `sandboxService`, `fileService`, `settingsService`, `sessionService`, `containerService`, `platformService`

## 5. 依赖关系
- 内部依赖: `app.go`（`NewApp()`）
- 外部依赖:
  - `github.com/wailsapp/wails/v2`
  - `github.com/wailsapp/wails/v2/pkg/options`
  - `github.com/wailsapp/wails/v2/pkg/options/assetserver`
  - `github.com/wailsapp/wails/v2/pkg/options/mac`
  - `github.com/wailsapp/wails/v2/pkg/options/windows`
  - `github.com/wailsapp/wails/v2/pkg/options/linux`
  - `embed`（标准库）

## 6. 变更影响面
- 修改 `Bind` 会影响前端 IPC 能力。
- 修改窗口参数会直接影响桌面端启动体验。
- `MinWidth: 768` 是 Figma v0.5 响应式验收的最小桌面断点，不能无意调回 1000，否则 sheet mode 无法进入。
- `WindowStartState` 变更会影响默认窗口状态（当前为最大化）。
- macOS titlebar 策略会影响系统级窗口行为；默认 titlebar 由系统处理交通灯、标题栏双击 zoom、拖拽和窗口菜单一致性。

## 7. 维护建议
- 新增后端服务时必须加入 `Bind`。
- 若调整前端产物目录，需同步修改 `//go:embed` 路径。
- 调整窗口策略时应验证 Windows/macOS 的一致性行为。
- 调整平台 shell 选项后应至少验证 macOS 本地 `wails build`，并确认 release workflow 三平台构建仍通过。
