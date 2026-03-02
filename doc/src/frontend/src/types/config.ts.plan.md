# config.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/types/config.ts
- 文档文件: doc/src/frontend/src/types/config.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/types (类型定义)

## 2. 核心职责
- 定义应用配置和沙盒状态相关的 TypeScript 接口类型。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 无（纯类型定义）
- 输出结果: 导出类型接口供 stores 和 components 使用

## 4. 关键实现细节
- **导出类型/接口**:
  - `SSHConfig` — SSH 连接配置: host, port, user, password?(可选), privateKey?(可选)
  - `DockerConfig` — Docker 容器配置: image, memoryLimit, cpuLimit, workDir, network
  - `LLMConfig` — 大模型配置: type (openai/deepseek/ark/ollama), baseURL, apiKey, model, headers?(可选 Record)
  - `MCPServerConfig` — MCP 服务器配置: name, transport (stdio/sse), command?, args?, url?, enabled
  - `AppSettings` — 顶层应用配置，聚合 ssh/docker/llm/mcp/agent 子配置
  - `FileInfo` — 文件信息: name, path, size, modified, preview, isOutput
  - `SandboxStatus` — 沙盒状态: sshConnected, dockerRunning, containerID, **activeContainerID**, **activeContainerName**, **dockerAvailable**

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: 无（纯类型定义）

## 6. 变更影响面
- `AppSettings` 修改影响 settingsStore 和所有设置表单组件 (SSHConfig, DockerConfig, LLMConfig, MCPConfig)
- `LLMConfig.type` 枚举扩展需同步 Go 后端的 LLM 提供商支持
- `FileInfo` 修改影响 FileExplorer 组件
- `SandboxStatus` 修改影响 connectionStore 的 refreshStatus

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 类型结构需与 Go 后端的配置结构体保持一一对应。
- 新增 LLM 提供商时需同步更新 type 联合类型和设置面板的下拉选项。
