# app.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: app.go
- 文档文件: doc/src/app.go.plan.md
- 文件类型: Go 源码
- 所属模块: main

## 2. 核心职责
- 该文件定义了应用的核心结构体 `App`，负责创建和组装所有后端服务实例、存储层和上下文引擎。`startup` 方法在 Wails 启动时被调用，完成服务间的依赖注入和事件回调连接。`shutdown` 方法在应用关闭时保存会话并断开 SSH 连接。该文件是整个后端的服务编排中心。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Wails 运行时提供的 `context.Context`；各存储层和配置的初始化结果
- 输出结果: 完全初始化的 `App` 实例，所有服务已就绪并相互连接

## 4. 关键实现细节
- 结构体/接口定义:
  - `App` — 主应用结构体，持有 `context.Context`、`config.Store`、`storage.SessionStore`、`storage.ContainerStore`、六个服务实例和 `agentctx.Engine`
- 导出函数/方法:
  - `NewApp() *App` — 创建并初始化所有服务和存储
- 关键回调连接:
  - `sandboxService.SetOnConnect` → 更新 chatService 的沙箱管理器
  - `sandboxService.SetOnContainerBound` → 绑定容器到当前会话
  - `sandboxService.SetOnContainerDeactivated` → **chatService.UpdateSandbox(nil)**（容器分离时清除 sandbox 引用）
  - `sessionService.SetOnSessionSwitch` → **ActivateContainer/DeactivateContainer**（不再断开 SSH，仅切换容器）
  - `sessionService.SetOnDestroyContainer` → 级联删除时调用 containerService.DestroyContainer
  - `settingsService.SetOnSettingsSave` → chatService runner 失效重建
  - `chatService.SetOnAgentDone` → 自动保存当前会话
- **关键行为变更**:
  - 会话切换不再断开/重连 SSH，仅通过 `ActivateContainer`/`DeactivateContainer` 切换活跃容器
  - 新增 `onContainerDeactivated` 回调确保容器分离时 ChatService 的 sandbox 引用被清除

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config`、`starxo/internal/context`、`starxo/internal/logger`、`starxo/internal/sandbox`、`starxo/internal/service`、`starxo/internal/storage`
- 外部依赖:
  - `context`、`os`、`path/filepath`（标准库）

## 6. 变更影响面
- 修改 `App` 结构体字段会影响 `main.go` 中的 Bind 列表
- 修改 `startup` 中的回调连接逻辑会影响服务间的事件响应
- 会话切换回调的行为变更直接影响用户切换会话时的连接体验

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增服务时需在 `App` 结构体中添加字段，在 `NewApp` 中初始化，在 `startup` 中设置上下文和依赖。
- 会话切换回调中使用 `go sandboxService.ActivateContainer()` 异步执行，避免阻塞会话切换流程。
