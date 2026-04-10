# `claude-code -> starxo` 功能迁移调研报告

日期：2026-04-10  
调研目标：补齐 `starxo` 的核心能力，同时优先提炼可迁移的算法、交互范式与工作流模式，而不是照搬 `claude-code` 的 Bun/Ink/Anthropic 私有实现。

## 1. 调研范围与方法

- 对象一：`claude-code` 当前仓库，重点查看 `README.md`、`src/tools.ts`、`src/commands.ts`、`src/skills/`、`src/plugins/`、`src/bridge/`、`src/services/compact/`、`src/services/SessionMemory/`、`src/memdir/`、`doc/` 中的工具与架构文档。
- 对象二：`starxo` 当前仓库，重点查看 `README.md`、`internal/agent/`、`internal/tools/`、`internal/context/`、`internal/service/`、`internal/storage/`、`internal/store/`、`plan/` 与前端对应 UI。
- 分析口径：按 8 个能力域统一比较，分别标注 `已有 / 部分具备 / 缺失 / 不适配`。
- 迁移类型定义：
  - `直接能力迁移`：能力本身可在 `starxo` 中重建。
  - `交互范式迁移`：不搬原语法或 UI，但搬交互模式。
  - `算法/策略迁移`：搬调度、窗口管理、权限判定、恢复策略等。
  - `工作流迁移`：搬任务分发、自动化、扩展点设计。
- 评分方法：6 维各 1-5 分，越高越值得做，总分 30。
  - 核心能力补齐度
  - 用户价值
  - 架构无关可迁移性
  - 对闭源/私有基础设施依赖度低
  - 实现复杂度低
  - 与 `starxo` 定位适配度

## 2. 执行摘要

- `starxo` 已经有一个不错的远程执行底座：SSH + Docker sandbox、多会话、Plan Mode、中断恢复、MCP、基础文件与命令工具、可视化 todo DAG。这些不应重造，应该继续增强。
- `claude-code` 的优势不在“单一模型更强”，而在一整层围绕 agent 的工程化能力：动态工具面、权限治理、上下文压缩与记忆、技能/插件、任务系统、IDE/bridge、自动化触发、工作流命令层。
- 对 `starxo` 来说，最值得迁移的不是 CLI/TUI 表层，而是 5 组基础设施能力：
  - 动态工具面与资源访问层
  - 权限治理与审计层
  - 会话记忆与上下文压缩层
  - 工作流封装层（skills/plugins）
  - 持久化任务编排层
- `starxo` 当前最明显的结构性短板有 4 个：
  - 工具面太静态，无法优雅承载更多能力与 MCP 资源。
  - 缺少细粒度权限模式、预批准规则与审计。
  - 上下文管理只有窗口裁剪，没有真正的会话摘要、长期记忆和自动压缩。
  - 缺少工作流复用层，很多复杂能力只能继续塞进 system prompt 或 hardcode tool。
- 建议优先级：
  - `P0`：动态工具面、权限治理与审计、Session Memory + Auto Compact、Skills/Plugins、Task V2。
  - `P1`：长期记忆、LSP、自动化调度、Git worktree。
  - `P2`：IDE/Bridge、通用 swarm/跨 session 消息协议。

## 3. 当前能力基线

### 3.1 `claude-code` 的高层轮廓

- `README.md` 明确其核心结构包含 `tools/`、`commands/`、`coordinator/`、`bridge/`、`plugins/`、`skills/`、`memdir/`、`remote/`、`server/`、`services/` 等大层级，且工具和命令规模已明显超出“简单 coding agent”。
- `src/tools.ts` 暴露的基础工具池包含 `AgentTool`、`SkillTool`、`BashTool`、`FileRead/Edit/Write`、`TodoWrite`、`Task*`、`LSPTool`、`EnterWorktree/ExitWorktree`、`ScheduleCronTool`、`RemoteTriggerTool`、`ReadMcpResourceTool`、`ToolSearchTool` 等。
- `src/commands.ts` 体现了独立的命令层，除调试/配置类命令外，还提供 `review`、`memory`、`mcp`、`session`、`skills`、`tasks`、`permissions`、`plugin`、`files`、`branch`、`share`、`resume` 等 workflow 入口。

### 3.2 `starxo` 的高层轮廓

- `README.md` 与 `internal/agent/` 说明其主体是 Go + Wails + Eino 的桌面远程 coding agent，核心价值在 SSH + Docker 远程沙箱、可视化桌面 UI、多会话与实时事件流。
- `internal/tools/builtin.go` 的基础 builtin 主要是 `shell_execute`、`list_files`、`read_file`、`write_file`、`str_replace_editor`、`python_execute`。
- `internal/tools/` 额外提供 `ask_user`、`ask_choice`、`notify_user`、`write_todos`、`update_todo`、`mcp` 接入与 recoverable tool error。
- `internal/agent/runner.go`、`internal/agent/deep_agent.go`、`internal/service/chat.go` 表明它已经具备 default/plan 双模式、多 session run state、interrupt/resume、timeline event stream。
- `internal/context/windowing.go` 与 `internal/context/engine.go` 目前只实现了基础窗口裁剪与文件上下文注入，还没有成熟的摘要/记忆体系。

## 4. 能力全景对比矩阵

| 能力域 | `claude-code` 现状 | `starxo` 现状 | 状态 | 结论 |
|---|---|---|---|---|
| 工具层 | `src/tools.ts` + `doc/src/tools/*` 形成大工具面，含 ToolSearch、Task V2、LSP、Worktree、Cron、RemoteTrigger、MCP 资源读取 | `internal/tools/builtin.go` + `internal/tools/mcp.go` + `internal/tools/registry.go`，以基础读写/执行/MCP 为主 | 部分具备 | 该层是 `starxo` 最大的核心能力缺口之一，应优先扩容并动态化 |
| 命令层 | `src/commands.ts` 构成独立 workflow 层 | 无独立命令/动作层，主要靠 prompt、按钮和工具 | 缺失 | 不建议照搬 slash command 语法，但应迁移“工作流封装”思想 |
| Agent 编排 | `AgentTool`、`coordinatorMode.ts`、`SendMessageTool`、`Task*` 支持 worker/swarms | `deep_agent.go` + 三个固定子 agent，Plan Mode 已存在 | 部分具备 | 基础已够，但还缺更通用的 worker 编排与共享任务模型 |
| 上下文与记忆 | `autoCompact.ts`、`SessionMemory/`、`memdir/`、`extractMemories/` 形成短期摘要 + 长期记忆双层机制 | `context/windowing.go` 只有简单裁剪；会话持久化仅保存消息与 display | 缺失 | 这是 `starxo` 最值得迁移的算法/策略层 |
| Skills / Plugins / MCP | `SkillTool`、`loadSkillsDir.ts`、`builtinPlugins.ts` 支持 file-based skills、插件化扩展、MCP skill builder | MCP 工具已接入，但无技能/插件/扩展工作流层 | 部分具备 | MCP 不该单独演化，应该与技能/插件统一成扩展体系 |
| IDE / Bridge | `src/bridge/*` 是完整 bridge 子系统，能管理远程 session 与外部入口 | Wails 桌面自身是主入口，没有 editor bridge | 缺失 | 有价值，但不是第一优先级 |
| 自动化 / 远程触发 | `ScheduleCronTool`、`RemoteTriggerTool`、`SleepTool`、相关 skills 形成自动化触发能力 | 暂无 cron / remote trigger；计划中也未覆盖 | 缺失 | 与远程桌面定位非常契合，建议在基础能力补齐后做 |
| 权限 / 治理 / 可观测性 | `doc/permissions.md`、Bash/File 权限、自定义 permission mode、MCP 审批、tool telemetry、audit 类能力成熟 | 只有 Docker 隔离与少量 tool wrapper/error policy；`plan/008-audit-log.md` 仍待实施 | 缺失 | 对远程执行产品尤其关键，应列入 `P0` |

## 5. 重点迁移候选 Top 列表

| 排名 | 候选能力 | 优先级 | 迁移类型 | 迁移分数 | 结论 |
|---|---|---|---|---|---|
| 1 | 动态工具面 + ToolSearch + MCP 资源层 | `P0` | 直接能力迁移 + 工作流迁移 | 28/30 | 应作为所有后续能力的承载层 |
| 2 | 权限治理模式 + 审计日志 | `P0` | 算法/策略迁移 + 直接能力迁移 | 28/30 | 远程执行场景的安全底线 |
| 3 | Session Memory + Auto Compact | `P0` | 算法/策略迁移 | 27/30 | 直接提升长对话稳定性和执行成功率 |
| 4 | Skills / Plugins / Workflow 封装层 | `P0` | 工作流迁移 | 26/30 | 解决功能增长后 prompt 膨胀与能力复用问题 |
| 5 | Task V2 持久化任务系统 | `P0` | 直接能力迁移 | 25/30 | 比当前 in-memory todo 更适合多 agent/多 session |
| 6 | 长期记忆系统（跨会话 memory） | `P1` | 算法/策略迁移 | 24/30 | 对个人开发助手和项目助手都很有价值 |
| 7 | LSP 语义代码智能 | `P1` | 直接能力迁移 | 23/30 | 提升精准度，但依赖远程语言服务器治理 |
| 8 | 自动化调度与远程触发 | `P1` | 直接能力迁移 + 工作流迁移 | 22/30 | 与远程 agent 定位契合，需先补权限与 checkpoint |
| 9 | Git worktree 隔离执行 | `P1` | 直接能力迁移 | 21/30 | 适合高风险任务与并发实验 |
| 10 | IDE/Bridge 与通用 swarm 协议 | `P2` | 交互范式迁移 + 工作流迁移 | 18/30 | 有价值，但不该早于前述基础层 |

## 6. 重点候选逐项分析

### Feature Card 1: 动态工具面 + ToolSearch + MCP 资源层

- `feature_name`: 动态工具暴露、延迟加载和资源型工具访问
- `source_evidence`: `/Users/zhaoziqian/claude-code/src/tools.ts`；`/Users/zhaoziqian/claude-code/doc/src/tools/ToolSearchTool/README.md`；`/Users/zhaoziqian/claude-code/doc/src/tools/ReadMcpResourceTool/README.md`；`/Users/zhaoziqian/starxo/internal/tools/registry.go`；`/Users/zhaoziqian/starxo/internal/tools/builtin.go`；`/Users/zhaoziqian/starxo/internal/tools/mcp.go`
- `current_starxo_state`: 已有静态 ToolRegistry、builtin 与 MCP tools 接入，但工具面固定且较小；没有 deferred tools、tool search、MCP resource list/read，也没有按 token 预算动态暴露工具。
- `transfer_type`: 直接能力迁移 + 工作流迁移
- `user_value`: 高。能力面扩充后，agent 才能在不塞爆 prompt 的前提下调用更多工具。
- `core_capability_gap`: 当前 `starxo` 只能依赖少量固定工具，随着 LSP、git、automation、memory 等能力加入，prompt 会迅速膨胀，MCP 也缺少资源读取层。
- `migration_score`: 28/30
- `recommended_priority`: `P0`
- `implementation_outline`: 1. 给 `ToolRegistry` 增加 metadata 层，区分 always-load、deferred、read-only、needs-approval、resource-like；2. 新增 `tool_search` 工具，对延迟工具做名称/关键词匹配；3. 在 MCP 侧补 `list_mcp_resources` 与 `read_mcp_resource` 封装；4. 在 prompt 构建阶段只暴露核心工具与 deferred 索引。
- `blocking_dependencies`: 需要先定义工具元数据模型与前端展示约定。
- `risk_notes`: 如果没有同步做权限层，动态暴露更多工具会直接放大风险面。

### Feature Card 2: 权限治理模式 + 审计日志

- `feature_name`: 分层权限模式、规则匹配、审批 UI 与审计日志
- `source_evidence`: `/Users/zhaoziqian/claude-code/doc/permissions.md`；`/Users/zhaoziqian/claude-code/src/tools.ts`；`/Users/zhaoziqian/claude-code/src/hooks/useInboxPoller.ts`；`/Users/zhaoziqian/starxo/README.md`；`/Users/zhaoziqian/starxo/internal/tools/builtin.go`；`/Users/zhaoziqian/starxo/internal/service/chat.go`；`/Users/zhaoziqian/starxo/plan/008-audit-log.md`
- `current_starxo_state`: 有 SSH + Docker 隔离，但 agent 调用 `shell_execute`、`write_file`、`python_execute` 时没有类似 `default / plan / acceptEdits / bypass` 的细粒度权限模式，也没有 per-tool 审批和规则系统；审计日志仍在 backlog。
- `transfer_type`: 算法/策略迁移 + 直接能力迁移
- `user_value`: 极高。`starxo` 是远程执行产品，权限确认和审计能力的重要性比本地 CLI 更高。
- `core_capability_gap`: 当前安全模型主要依赖“容器隔离”，但容器内破坏、误删文件、危险网络操作、自动化任务误触发等风险都没有更细粒度治理。
- `migration_score`: 28/30
- `recommended_priority`: `P0`
- `implementation_outline`: 1. 在 `internal/tools` 给工具定义 read-only / write / dangerous / network 类别；2. 在 `ChatService` 前插统一 permission middleware；3. 前端增加审批对话框与默认模式设置；4. 同步实现 `plan/008-audit-log.md`；5. 将 permission 结果与 timeline event 打通。
- `blocking_dependencies`: 需要工具元数据、前端审批交互、审计日志落盘方案。
- `risk_notes`: 若直接照搬 `claude-code` 的模式集合会过重，建议先做 `default / plan / accept_edits / bypass` 四档，再逐步加规则。

### Feature Card 3: Session Memory + Auto Compact

- `feature_name`: 会话摘要、自动压缩、上下文预算管理
- `source_evidence`: `/Users/zhaoziqian/claude-code/src/services/compact/autoCompact.ts`；`/Users/zhaoziqian/claude-code/src/services/SessionMemory/sessionMemory.ts`；`/Users/zhaoziqian/claude-code/doc/src/services/SessionMemory/sessionMemory.md`；`/Users/zhaoziqian/starxo/internal/context/engine.go`；`/Users/zhaoziqian/starxo/internal/context/windowing.go`；`/Users/zhaoziqian/starxo/internal/storage/session_store.go`
- `current_starxo_state`: 只有简单的 `MaxMessages + MaxContentLen` 裁剪和“Earlier conversation omitted”占位，没有真正的摘要文件、压缩触发阈值、post-sampling memory extraction。
- `transfer_type`: 算法/策略迁移
- `user_value`: 高。长会话下能显著降低上下文失真、减少 agent 在后半段失忆或重复探索。
- `core_capability_gap`: `starxo` 的窗口策略会直接丢弃中间语义链条，对长任务、连续调试、多轮修改非常不友好。
- `migration_score`: 27/30
- `recommended_priority`: `P0`
- `implementation_outline`: 1. 在 `internal/context` 上方新增 session memory manager；2. 通过后台轻量子 agent 或摘要流程更新 `session_memory.md`；3. 在接近 token 阈值时优先注入摘要，再做原始消息裁剪；4. 与 session persistence 一起保存 memory cursor。
- `blocking_dependencies`: 需要 token 预算估算、后台摘要调用策略，以及 checkpoint 持久化避免中断丢状态。
- `risk_notes`: 如果摘要质量不稳定，可能把错误结论固化进后续上下文；必须保留“原始上下文优先 + 摘要可重写”的策略。

### Feature Card 4: Skills / Plugins / Workflow 封装层

- `feature_name`: file-based skills、插件扩展与 workflow 复用层
- `source_evidence`: `/Users/zhaoziqian/claude-code/src/skills/loadSkillsDir.ts`；`/Users/zhaoziqian/claude-code/doc/src/tools/SkillTool/README.md`；`/Users/zhaoziqian/claude-code/src/plugins/builtinPlugins.ts`；`/Users/zhaoziqian/starxo/README.md`；`/Users/zhaoziqian/starxo/internal/tools/registry.go`
- `current_starxo_state`: 只有工具注册，没有“可组合工作流”层；新的复杂能力目前只能继续写死在 prompt、agent 或前端。
- `transfer_type`: 工作流迁移
- `user_value`: 高。能把 `review`、`verify`、`commit`、`analyze logs`、`setup env` 等高频工作流封装成可维护能力，而不是让主 agent 每次临场发挥。
- `core_capability_gap`: 当前 `starxo` 缺少“能力打包与复用”层，导致产品成长后 system prompt 与工具列表会越来越臃肿。
- `migration_score`: 26/30
- `recommended_priority`: `P0`
- `implementation_outline`: 1. 不照搬 slash command，而是在 `starxo` 中定义 `workflow specs` 或 `skills` 目录；2. 支持声明 prompt、允许工具、可选模型、是否 fork 执行；3. 后续把内建 workflow 与 MCP workflow 统一接入。
- `blocking_dependencies`: 需要先定义 workflow schema 和加载器。
- `risk_notes`: 如果没有权限边界和工具分类，skills 只会变成另一层 hardcode prompt。

### Feature Card 5: Task V2 持久化任务系统

- `feature_name`: 并发安全的持久化任务模型
- `source_evidence`: `/Users/zhaoziqian/claude-code/doc/tasks.md`；`/Users/zhaoziqian/claude-code/doc/src/tools/TaskCreateTool/README.md`；`/Users/zhaoziqian/claude-code/doc/src/tools/TodoWriteTool/README.md`；`/Users/zhaoziqian/starxo/internal/tools/todos.go`；`/Users/zhaoziqian/starxo/frontend/src/components/layout/TaskRailFloating.vue`
- `current_starxo_state`: 已有 `write_todos` / `update_todo` 和 DAG UI，但 todo 数据是会话内内存结构，不支持 owner、并发 agent 协同、持久化 hooks 与更细状态流转。
- `transfer_type`: 直接能力迁移
- `user_value`: 高。比当前 todo 更适合多 agent、长任务、后台任务和恢复场景。
- `core_capability_gap`: 现在的 todo 只能用来做视觉进度提示，还不足以承担“任务系统”的职责。
- `migration_score`: 25/30
- `recommended_priority`: `P0`
- `implementation_outline`: 1. 保留现有 DAG UI，但把后端从 in-memory todo 升级为持久化 task records；2. 支持 `create/get/list/update/stop`；3. 增加 owner、blockedBy、metadata；4. 后续与多 agent、automation、skills 对接。
- `blocking_dependencies`: 需要定义持久化目录结构与 session/team 维度。
- `risk_notes`: 不建议一次性复制 `claude-code` 的全部 swarm 语义，先实现单 session/单用户可恢复任务即可。

### Feature Card 6: 长期记忆系统（跨会话 memory）

- `feature_name`: file-based durable memory 与自动提取
- `source_evidence`: `/Users/zhaoziqian/claude-code/src/memdir/memdir.ts`；`/Users/zhaoziqian/claude-code/src/services/extractMemories/extractMemories.ts`；`/Users/zhaoziqian/claude-code/src/memdir/memoryTypes.ts`；`/Users/zhaoziqian/starxo/internal/storage/session_store.go`；`/Users/zhaoziqian/starxo/README.md`
- `current_starxo_state`: 会话数据会落盘，但没有“跨会话共享的用户/项目记忆”；每个 session 都像从零开始。
- `transfer_type`: 算法/策略迁移
- `user_value`: 中高。对个人桌面 coding assistant 很有价值，尤其适合保存偏好、项目约束、常用远程环境信息。
- `core_capability_gap`: `starxo` 有 session persistence，但没有 durable memory；二者用途不同。
- `migration_score`: 24/30
- `recommended_priority`: `P1`
- `implementation_outline`: 1. 在 `~/.starxo/` 下设计 `memory/` 层，区分 user/project/reference；2. 引入显式保存与后台提取双路径；3. 与权限层结合，避免错误记忆写入。
- `blocking_dependencies`: 需要先补 Session Memory 机制和文件级权限控制。
- `risk_notes`: 记忆是高敏感功能，必须先定义何种内容禁止保存，以及用户如何查看/删除。

### Feature Card 7: LSP 语义代码智能

- `feature_name`: 语言服务器驱动的语义检索与导航
- `source_evidence`: `/Users/zhaoziqian/claude-code/doc/lsp.md`；`/Users/zhaoziqian/claude-code/src/services/lsp/manager.ts`；`/Users/zhaoziqian/claude-code/src/tools.ts`；`/Users/zhaoziqian/starxo/internal/tools/builtin.go`；`/Users/zhaoziqian/starxo/internal/sandbox/ssh.go`
- `current_starxo_state`: 只有文本级读文件、列文件和 shell，没有 go-to-definition、find-references、hover、workspace symbol 等语义工具。
- `transfer_type`: 直接能力迁移
- `user_value`: 高。能显著提升代码理解质量，减少大仓库下纯文本搜索的误报。
- `core_capability_gap`: `starxo` 当前读代码仍偏“文本代理”，没有语义层。
- `migration_score`: 23/30
- `recommended_priority`: `P1`
- `implementation_outline`: 1. 在远程容器内启动语言服务器并做代理管理；2. 先支持 TS/Go/Python 三类常见语言；3. 以工具形式暴露 `definition / references / hover / symbols`，而非先做完整 IDE。
- `blocking_dependencies`: 需要远程语言服务器进程生命周期管理，以及文件变更同步。
- `risk_notes`: 如果没有与容器文件状态同步，LSP 结果会过期；必须绑定 `write_file/str_replace_editor` 后的 didOpen/didChange/didSave。

### Feature Card 8: 自动化调度与远程触发

- `feature_name`: cron / scheduled jobs / remote trigger
- `source_evidence`: `/Users/zhaoziqian/claude-code/doc/src/tools/ScheduleCronTool/README.md`；`/Users/zhaoziqian/claude-code/doc/src/tools/RemoteTriggerTool/README.md`；`/Users/zhaoziqian/claude-code/src/skills/bundled/loop.ts`；`/Users/zhaoziqian/starxo/internal/store/checkpoint.go`；`/Users/zhaoziqian/starxo/plan/005-checkpoint-persistence.md`
- `current_starxo_state`: 暂无调度与远程触发能力；checkpoint 仍是 in-memory store，重启无法恢复中断状态。
- `transfer_type`: 直接能力迁移 + 工作流迁移
- `user_value`: 中高。非常契合远程沙箱产品，可以做定时检查、批处理、夜间构建和主动提醒。
- `core_capability_gap`: `starxo` 目前所有任务都必须手动发起，不能把 agent 变成持续运行的工作助手。
- `migration_score`: 22/30
- `recommended_priority`: `P1`
- `implementation_outline`: 1. 先完成持久化 checkpoint；2. 设计 `automation` 数据模型与 UI；3. 第一阶段仅支持本地 cron + workspace task，不接外部云控制；4. 后续再扩为 remote triggers。
- `blocking_dependencies`: `plan/005-checkpoint-persistence.md`、权限治理、审计日志。
- `risk_notes`: 若没有权限审计和资源配额，自动化很容易成为失控执行面。

### Feature Card 9: Git worktree 隔离执行

- `feature_name`: 每任务独立 worktree / branch 隔离
- `source_evidence`: `/Users/zhaoziqian/claude-code/src/tools.ts`；`/Users/zhaoziqian/claude-code/doc/src/tools/ExitWorktreeTool/ExitWorktreeTool.md`；`/Users/zhaoziqian/claude-code/doc/architecture.md`；`/Users/zhaoziqian/starxo/internal/sandbox/docker.go`；`/Users/zhaoziqian/starxo/internal/service/session_svc.go`
- `current_starxo_state`: 已有容器级隔离和会话-容器绑定，但同一容器/工作区内的高风险修改仍会相互影响，没有 repo-level worktree 隔离。
- `transfer_type`: 直接能力迁移
- `user_value`: 中高。适合实验性改动、并发修复、PR 级别验证。
- `core_capability_gap`: 目前容器隔离粒度偏粗，成本较高；很多任务其实只需要 repo 内隔离。
- `migration_score`: 21/30
- `recommended_priority`: `P1`
- `implementation_outline`: 1. 在已有 container/workspace 模型上增加可选 worktree session；2. 在任务启动时选择“复用工作区 / 新建 worktree”；3. 与 session/container 层级一起管理回收。
- `blocking_dependencies`: 需要 git-aware workspace 管理和 UI 提示。
- `risk_notes`: 远程容器中的 git 状态复杂时，worktree 生命周期管理要谨慎，不然会制造更多脏状态。

### Feature Card 10: IDE/Bridge 与通用 swarm 协议

- `feature_name`: 外部 IDE/Session bridge 与通用 worker 协议
- `source_evidence`: `/Users/zhaoziqian/claude-code/src/bridge/bridgeMain.ts`；`/Users/zhaoziqian/claude-code/doc/src/tools/SendMessageTool/README.md`；`/Users/zhaoziqian/claude-code/src/coordinator/coordinatorMode.ts`；`/Users/zhaoziqian/starxo/internal/agent/deep_agent.go`；`/Users/zhaoziqian/starxo/internal/tools/todos.go`
- `current_starxo_state`: 有固定三子 agent 和桌面 UI 事件流，但没有通用 worker 池、agent 间消息协议、editor bridge。
- `transfer_type`: 交互范式迁移 + 工作流迁移
- `user_value`: 中等。长远看重要，但不直接决定 `starxo` 近期核心体验。
- `core_capability_gap`: 当前 agent 编排主要是固定结构，不便于扩展成更开放的 worker 协作网络。
- `migration_score`: 18/30
- `recommended_priority`: `P2`
- `implementation_outline`: 先不要做完整 CCR/bridge 复制；优先抽象 `worker task + message channel + task ownership`，等 Task V2 与权限层成熟后再扩到 IDE/外部入口。
- `blocking_dependencies`: Task V2、权限治理、automation、editor 集成需求。
- `risk_notes`: 这是“系统乘法器”，做得太早会把复杂度提前释放。

## 7. 哪些 feature 不建议直接迁移

- Anthropic 私有云与账号体系：
  - 证据：`RemoteTriggerTool` 直接依赖 claude.ai CCR API、OAuth token、组织 UUID、beta headers。
  - 结论：不应直接迁移 API 形态，应只借鉴“可调度远程 agent”这个产品能力。
- GrowthBook / Datadog / 大量 feature flag 基础设施：
  - 证据：`src/tools.ts`、`src/commands.ts`、`bridgeMain.ts`、`autoCompact.ts` 广泛依赖 `feature()` 与 GrowthBook gate。
  - 结论：可迁移“灰度发布与 kill switch”思想，但不值得在当前阶段复制整套实验平台。
- CLI/TUI 表层与 slash command 语法：
  - 证据：`src/commands.ts`、Ink 组件体系、`SkillTool` 对 slash command 的兼容。
  - 结论：`starxo` 是 Wails 桌面应用，不应为了“像 Claude Code”而复刻 `/command` 语法；应迁移为命令面板、workflow palette 或 skill launcher。
- KAIROS / BUDDY / TORCH / ULTRAPLAN 等内部实验特性：
  - 证据：`src/tools.ts`、`src/commands.ts` 中大量 feature gate。
  - 结论：这些不是迁移目标，应只提取底层模式，不复制具体 feature 名称和产品形态。

## 8. 建议优先迁移的 8 个能力

1. 动态工具面 + ToolSearch + MCP 资源层
2. 权限治理模式 + 审计日志
3. Session Memory + Auto Compact
4. Skills / Plugins / Workflow 封装层
5. Task V2 持久化任务系统
6. 长期记忆系统（跨会话 memory）
7. LSP 语义代码智能
8. 自动化调度与远程触发

## 9. 暂不建议优先推进的能力

- 直接复制 slash command 语法
- 直接复制 claude.ai CCR / OAuth / RemoteTrigger API 形态
- 过早建设通用 worker swarm/跨 session 协议
- 复制 Anthropic 内部实验 feature 与遥测基础设施

## 10. 推荐路线图

### Phase 0: 地基补齐

- 完成 `plan/005-checkpoint-persistence.md`
- 完成 `plan/008-audit-log.md`
- 给 `internal/tools` 建立统一 metadata：工具类型、风险级别、是否可延迟加载、是否只读

### Phase 1: 核心能力补齐

- 实现动态工具面与 `tool_search`
- 实现基础权限模式与审批 UI
- 把现有 in-memory todos 升级为持久化 Task V2
- 让 timeline、task rail、tool execution 共享统一任务/权限状态

### Phase 2: 上下文智能化

- 引入 Session Memory
- 引入 Auto Compact
- 在此基础上设计长期 memory 目录与显式记忆操作

### Phase 3: 生态与精准度

- 引入 skill/workflow 目录与插件加载器
- 补 LSP 语义工具
- 试点 Git worktree 隔离

### Phase 4: 自动化与开放入口

- 在权限与 checkpoint 稳定后引入 automation/cron
- 再评估 IDE bridge、外部 session 入口与通用 swarm 协议

## 11. 对 `starxo` 现有能力的结论

- `Plan Mode`：已经有较好的基础，不建议重写；建议与 Task V2、权限模式联动增强。
- `interrupt/resume`：基础路径已存在，但必须补 durable checkpoint，否则自动化、长任务与 crash recovery 都不可靠。
- `MCP`：工具接入已完成，但资源读取、审批、deferred loading 还很弱，应增强而非重造。
- `todo DAG UI`：已经是很好的前端资产，建议后端升级为持久化 task，而不是废弃。
- `远程容器隔离`：这是 `starxo` 相比 `claude-code` 的天然差异化优势，迁移工作应始终围绕这个定位展开。

## 12. 最终判断

- 如果只做一个结论：`starxo` 不该追求“做成另一个 Claude Code CLI”，而应该把 `claude-code` 中最成熟的 agent 工程化能力迁移到自己的远程桌面产品里。
- 最值得先搬的不是 UI，而是 4 层基础设施：`动态工具面`、`权限治理`、`上下文记忆/压缩`、`工作流封装`。
- 只要这四层补齐，`starxo` 再叠加它已有的 SSH + Docker + Wails 桌面交互，就能形成与 `claude-code` 不同但竞争力很强的产品形态。
