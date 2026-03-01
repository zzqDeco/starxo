# settings_svc.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/settings_svc.go
- 文档文件: doc/src/internal/service/settings_svc.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件实现了 `SettingsService`，负责管理应用配置的读取、保存和连接测试。提供 SSH 和 LLM 连接的测试功能，让用户在保存配置前验证连接可用性。设置保存后通过回调通知 ChatService 使缓存的 runner 失效，确保下次消息使用最新配置。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - 前端 Wails 绑定调用: `GetSettings()`、`SaveSettings(cfg)`、`TestSSHConnection(sshCfg)`、`TestLLMConnection(llmCfg)`
  - 依赖注入: `config.Store`
- 输出结果:
  - `GetSettings`: 返回当前 `*config.AppConfig`
  - `SaveSettings`: 持久化配置并触发 `onSettingsSave` 回调
  - `TestSSHConnection`: 尝试 SSH 连接并关闭，返回 error
  - `TestLLMConnection`: 发送最小请求验证 LLM API 可达性，15 秒超时

## 4. 关键实现细节
- 结构体/接口定义:
  - `SettingsService`: 设置服务结构体，包含 Wails 上下文、配置存储、保存后回调
- 导出函数/方法:
  - `NewSettingsService(store) *SettingsService`: 构造函数
  - `SetOnSettingsSave(fn)`: 注册设置保存后回调
  - `SetContext(ctx)`: 设置 Wails 上下文
  - `GetSettings() *config.AppConfig`: 获取当前配置
  - `SaveSettings(cfg AppConfig) error`: 保存配置，使用 `store.Update` 的函数式更新
  - `TestSSHConnection(sshCfg SSHConfig) error`: 测试 SSH 连接，使用 `sandbox.NewSSHClient` 创建客户端并连接
  - `TestLLMConnection(llmCfg LLMConfig) error`: 测试 LLM 连接，创建模型并发送 "hi" 测试消息
- Wails 绑定方法: `GetSettings`、`SaveSettings`、`TestSSHConnection`、`TestLLMConnection`
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config`: Store、AppConfig、SSHConfig、LLMConfig
  - `starxo/internal/llm`: NewChatModel
  - `starxo/internal/sandbox`: NewSSHClient
- 外部依赖:
  - `github.com/cloudwego/eino/schema`: Message、User 角色
- 关键配置: LLM 测试超时 15 秒

## 6. 变更影响面
- `SaveSettings` 的回调触发 ChatService 的 `InvalidateRunner`，影响代理重建
- `TestSSHConnection` 使用 `sandbox.NewSSHClient`，受沙箱模块 SSH 实现影响
- `TestLLMConnection` 使用 `llm.NewChatModel`，受 LLM 模块实现影响
- 配置结构体 `AppConfig` 的字段变更需同步此文件和前端设置页面

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `TestLLMConnection` 发送的测试消息 "hi" 会消耗少量 token，可考虑使用更经济的验证方式（如仅检查认证头）。
- LLM 测试的 15 秒超时对于某些慢速端点可能不够，可考虑使其可配置。
- `SaveSettings` 的函数式更新 `store.Update(func(current *config.AppConfig) { *current = cfg })` 是线程安全的，应保持此模式。
