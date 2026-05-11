# LSPConfig.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/settings/LSPConfig.vue`
- 设置页 Runtime LSP 配置面板。

## 核心职责
- 管理 `settings.agent.lsp` 的启用状态、请求超时、最大结果字节数和自定义 language server 列表。
- 自定义 server 支持 language、executable、command args、extensions、disabled。
- 调用 `ChatService.GetRuntimeLSPStatus` 查看当前 session 的常驻 LSP server 状态。
- 状态面板同时展示 configured server mappings 和已启动的 server processes，方便区分“已配置”和“当前正在运行”。

## 维护建议
- 该面板只保存配置，不直接安装远端 language server。
- 可写 LSP 能力如 rename/format/code action 必须先接入 permission queue。
