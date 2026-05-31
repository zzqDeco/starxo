# zh.ts 技术说明

## 文件定位
- 源文件: `frontend/src/locales/zh.ts`
- 中文 UI 文案。

## 核心职责
- Docker/容器用户可见文案迁移为沙箱。
- 新增 `settings.sandbox` 运行时配置、检测和安装文案。
- 新增沙箱诊断、远端修复指南、工作区元信息和 tmp 清理文案。
- 新增 Runtime Tasks 面板、任务刷新、输出复制、停止任务文案。
- 新增 `permissions` 文案，用于工具审批 modal 和 Settings / Permissions 授权管理面板。
- 新增 `workspace.worktree` 和 worktree 工具文案，用于工作区抽屉审阅/合并 Runtime V2 worktree。
- 新增 `terminal` command runner 文案，用于 active sandbox 命令输入、禁用状态和失败提示。
- 新增 file/edit diff review 文案，用于 Runtime V2 `Write` / `Edit` timeline 结构化结果。
- 新增 `message.agent.*` 文案，用于 assistant timeline 中的 agent 名称，避免中文 UI 暴露内部英文变量名。
- macOS native 设计语言调整后，核心入口文案收敛为更短、更自然的产品语言，例如“新建”“准备开始”“执行命令”“浏览文件”，减少工作台里暴露实现名或变量式描述。
- Figma v0.5 对齐后，`chat.agentRunning`、`chat.taskProgress`、`chat.statusRunning`、`chat.statusReady` 服务 composer 上方 inline task status；`chat.modePlanHint` 避免继续表达“主 agent / subagent”旧范式。
- `containers.unavailable` 用于旧 Docker 记录。
