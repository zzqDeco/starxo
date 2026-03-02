# app.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: app.go
- 文档文件: doc/src/app.go.plan.md
- 文件类型: Go 源码
- 所属模块: main

## 2. 核心职责
- 该文件定义了应用的核心结构体 `App`，负责创建和组装所有后端服务实例、存储层和上下文引擎。`startup` 方法在 Wails 启动时被调用，完成服务间的依赖注入和事件回调连接（沙箱连接、容器绑定、会话切换、设置保存等）。`shutdown` 方法在应用关闭时保存会话并断开 SSH 连接。该文件是整个后端的服务编排中心。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Wails 运行时提供的 `context.Context`；各存储层和配置的初始化结果
- 输出结果: 完全初始化的 `App` 实例，所有服务已就绪并相互连接；应用关闭时保存会话状态

## 4. 关键实现细节
- 结构体/接口定义:
  - `App` — 主应用结构体，持有 `context.Context`、`config.Store`、`storage.SessionStore`、`storage.ContainerStore`、六个服务实例和 `agentctx.Engine`
- 导出函数/方法:
  - `NewApp() *App` — 创建并初始化所有服务和存储
- Wails 绑定方法: 无直接绑定（绑定在 `main.go` 中完成）
- 事件发射: 无直接事件发射（通过回调连接服务间事件）
- 关键回调连接:
  - `sandboxService.SetOnConnect` → 更新 chatService 的沙箱管理器
  - `sandboxService.SetOnContainerBound` → 绑定容器到当前会话
  - `sandboxService.SetSessionService` → 容器注册时获取活跃会话 ID
  - `sessionService.SetOnSessionSwitch` → 自动重连容器（空容器时调用 `Disconnect()`）
  - `sessionService.SetOnDestroyContainer` → 级联删除时调用 `containerService.DestroyContainer`
  - `containerService.SetSessionService` → 销毁容器时更新所属会话
  - `settingsService.SetOnSettingsSave` → 使 chatService runner 失效重建
  - `chatService.SetOnAgentDone` → 自动保存当前会话

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config` (配置存储)
  - `starxo/internal/context` (上下文引擎，别名 `agentctx`)
  - `starxo/internal/logger` (日志系统)
  - `starxo/internal/sandbox` (沙箱管理器类型)
  - `starxo/internal/service` (六个服务: Chat, Sandbox, File, Settings, Session, Container)
  - `starxo/internal/storage` (SessionStore, ContainerStore)
- 外部依赖:
  - `context` (Go 标准库)
  - `os`, `path/filepath` (Go 标准库，用于确定项目根目录)
- 关键配置:
  - 系统提示词: 硬编码的 AI 编码助手角色描述
  - 上下文 Token 预算: 8000

## 6. 变更影响面
- 修改 `App` 结构体字段会影响 `main.go` 中的 Bind 列表
- 修改 `startup` 中的回调连接逻辑会影响服务间的事件响应
- 修改 `NewApp` 中的服务初始化顺序可能导致依赖错误
- 修改 `shutdown` 逻辑会影响应用退出时的数据持久化
- 系统提示词变更会影响 AI Agent 的行为表现

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增服务时需在 `App` 结构体中添加字段，在 `NewApp` 中初始化，在 `startup` 中设置上下文和依赖，并在 `main.go` 的 Bind 列表中注册。
- 系统提示词和 Token 预算当前为硬编码值，后续可考虑迁移到配置文件中。
- `startup` 方法中的回调连接顺序需谨慎，确保依赖的服务已完成初始化。
