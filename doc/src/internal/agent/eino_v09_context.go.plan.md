# eino_v09_context.go 技术说明

## 1. 文件定位
- 源文件: `internal/agent/eino_v09_context.go`
- 所属模块: agent

## 2. 核心职责
- 集中创建 Eino v0.9 context middlewares。
- 将远端 sandbox workspace 包装成 Eino filesystem backend。
- 接入 summarization、reduction、skill、agentsmd middleware。

## 3. 关键实现细节
- `remoteWorkspaceBackend` 通过 `commandline.Operator` 读取和写入当前 sandbox workspace 文件。
- `remoteWorkspaceBackend.resolve(...)` 会拒绝 `..` 逃逸和 workspace 外绝对路径；middleware 文件读写也必须遵守 workspace guard。
- `reduction` 将大工具结果写入 `.starxo/tool-results`，并使用 `Read` 作为恢复工具名。
- `skill` 搜索 `.starxo/skills/<name>/SKILL.md` 和 `.claude/skills/<name>/SKILL.md`。
- `agentsmd` 读取 `AGENTS.md` 和 `.starxo/AGENTS.md`，作为 transient context 注入。
- summarization 只处理模型上下文压缩；Starxo 的 runtime sidecar compact 仍由 service 层维护。

## 4. 安全边界
- middleware 只负责上下文和结果管理。
- workspace guard、permission、sandbox 隔离和 private web access 仍由 Starxo runtime tools/service 层处理。
- 本文件的 backend guard 是第二层防线，不能替代 runtime tool 的路径校验。
