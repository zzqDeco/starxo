# config.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/config/config.go
- 文档文件: doc/src/internal/config/config.go.plan.md
- 文件类型: Go 源码
- 所属模块: config

## 2. 核心职责
- 该文件定义了 starxo 应用的全部配置数据模型，包括 SSH 连接、Docker 容器、LLM 大模型、MCP 服务器和 Agent 行为的配置结构体。同时提供了 `DefaultConfig` 函数生成合理的默认配置值。所有配置结构体均支持 JSON 序列化，用于持久化到磁盘和前端交互。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 无外部输入（纯数据类型定义）
- 输出结果: `DefaultConfig()` 返回带有默认值的 `*AppConfig` 实例

## 4. 关键实现细节
- 结构体/接口定义:
  - `AppConfig` — 根配置结构体，聚合 SSH、Docker、LLM、MCP、Agent 子配置
  - `SSHConfig` — SSH 连接配置 (Host, Port, User, Password, PrivateKey)
  - `DockerConfig` — Docker 容器配置 (Image, MemoryLimit, CPULimit, WorkDir, Network)
  - `LLMConfig` — LLM 提供商配置 (Type, BaseURL, APIKey, Model, Headers)
  - `MCPConfig` — MCP 服务器列表配置
  - `MCPServerConfig` — 单个 MCP 服务器配置 (Name, Transport, Command, Args, URL, Env, Enabled)
  - `AgentConfig` — Agent 行为配置 (MaxIterations)
- 导出函数/方法:
  - `DefaultConfig() *AppConfig` — 返回默认配置
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: 无（纯 Go 标准库类型）
- 关键配置:
  - 默认 SSH 端口: 22，用户: root
  - 默认 Docker 镜像: python:3.11-slim，内存: 2048MB，CPU: 1.0 核，工作目录: /workspace
  - 默认 LLM 类型: openai，模型: gpt-4o
  - 默认 Agent 最大迭代: 30

## 6. 变更影响面
- 修改配置结构体字段会影响 `config.Store` 的序列化/反序列化
- 修改配置字段会影响前端设置页面的数据绑定
- 修改默认值会影响首次启动时的应用行为
- `LLMConfig` 变更会影响 `internal/llm/provider.go` 的模型创建逻辑
- `SSHConfig` 变更会影响 `internal/sandbox/` 的连接逻辑
- `DockerConfig` 变更会影响容器创建参数

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增配置字段时需确保有合理的默认值（在 `DefaultConfig` 中设置），避免零值导致运行异常。
- 密码和 API Key 等敏感字段使用了 `omitempty` 标签，确保新增敏感字段也遵循此模式。
- 配置结构体变更后需同步更新前端 TypeScript 类型定义。
