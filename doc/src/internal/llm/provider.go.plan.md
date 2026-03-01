# provider.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/llm/provider.go
- 文档文件: doc/src/internal/llm/provider.go.plan.md
- 文件类型: Go 源码
- 所属模块: llm

## 2. 核心职责
- 该文件实现了 LLM 提供商的工厂函数 `NewChatModel`，根据配置中的提供商类型（openai/deepseek、ark、ollama）创建对应的 Eino `ToolCallingChatModel` 实例。同时实现了自定义 HTTP 传输层 `headerTransport`，支持为 LLM API 请求注入自定义 HTTP 头（如认证头、路由头等）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `context.Context`；`config.LLMConfig` 配置（Type, BaseURL, APIKey, Model, Headers）
- 输出结果: `model.ToolCallingChatModel` 接口实例；错误信息（不支持的提供商类型时）

## 4. 关键实现细节
- 结构体/接口定义:
  - `headerTransport` — 未导出结构体，实现 `http.RoundTripper` 接口，用于注入自定义 HTTP 头
- 导出函数/方法:
  - `NewChatModel(ctx context.Context, cfg config.LLMConfig) (model.ToolCallingChatModel, error)` — LLM 模型工厂函数
- 未导出函数:
  - `httpClientWithHeaders(headers map[string]string) *http.Client` — 创建带自定义头的 HTTP 客户端
- 支持的提供商:
  - `"openai"` / `"deepseek"` — 使用 eino-ext OpenAI 适配器
  - `"ark"` — 使用 eino-ext 火山引擎 Ark 适配器
  - `"ollama"` — 使用 eino-ext Ollama 适配器（默认地址 `http://localhost:11434`）
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config` (LLMConfig 配置类型)
- 外部依赖:
  - `context`, `fmt`, `net/http` (Go 标准库)
  - `github.com/cloudwego/eino-ext/components/model/ark` (火山引擎 Ark 模型适配器)
  - `github.com/cloudwego/eino-ext/components/model/ollama` (Ollama 模型适配器)
  - `github.com/cloudwego/eino-ext/components/model/openai` (OpenAI 模型适配器)
  - `github.com/cloudwego/eino/components/model` (ToolCallingChatModel 接口)
- 关键配置:
  - Ollama 默认 BaseURL: `http://localhost:11434`

## 6. 变更影响面
- 新增提供商类型需在 `switch` 语句中添加 case 分支
- 修改 `headerTransport` 逻辑会影响所有提供商的 HTTP 请求行为
- 返回的 `ToolCallingChatModel` 被 ChatService 中的 Eino ADK Runner 使用
- `LLMConfig` 结构体变更（`config.go`）需同步更新此文件的字段引用

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增 LLM 提供商时需引入对应的 eino-ext 适配器包，并在 `switch` 中添加处理分支。
- `"openai"` 和 `"deepseek"` 共享同一处理逻辑（均使用 OpenAI 兼容协议），后续若 DeepSeek 需要特殊处理可拆分。
- 自定义 Headers 功能主要用于 API 网关场景（如 API 路由、自定义认证），应在用户文档中说明其用途。
