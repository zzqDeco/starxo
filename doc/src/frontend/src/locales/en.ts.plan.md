# en.ts 技术说明

## 文件定位
- 源文件: `frontend/src/locales/en.ts`
- 英文 UI 文案。

## 核心职责
- Docker/Container 用户可见文案迁移为 Sandbox。
- 新增 `settings.sandbox` 运行时配置、检测和安装文案。
- 新增 sandbox diagnostics、remote fix guide、workspace metadata/tmp cleanup 文案。
- 新增 Runtime Tasks panel、task refresh、output copy、task stop 文案。
- 新增 `permissions` copy for the tool approval modal and Settings / Permissions grant management panel.
- 新增 `workspace.worktree` 和 worktree tool labels，用于工作区抽屉审阅/合并 Runtime V2 worktree。
- 新增 `terminal` command runner 文案，用于 active sandbox 命令输入、禁用状态和失败提示。
- 新增 file/edit diff review labels，用于 Runtime V2 `Write` / `Edit` timeline 结构化结果。
- 新增 `message.agent.*` labels，用于 assistant timeline 中的 agent 名称，避免 UI 暴露内部变量式名称。
- macOS native design pass tightened high-traffic copy to shorter product language such as “New”, “Ready when you are”, “Run commands”, and “Browse files”, reducing implementation-like labels in the workbench.
- Figma v0.5 alignment adds `chat.agentRunning`, `chat.taskProgress`, `chat.statusRunning`, and `chat.statusReady` for the inline task status above the composer; `chat.modePlanHint` no longer describes the old main-agent/subagent split.
- `containers.unavailable` 用于旧 Docker 记录。
