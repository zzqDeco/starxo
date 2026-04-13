# `starxo` Deferred Tool Surface Phase 2 计划

## 1. Summary

当前 `starxo` 已经完成 deferred MCP surface 的 phase-1 落地：

- `SessionData.DiscoveredTools` 作为 discovery 唯一持久化状态源
- `tool_search` / announcement / visible tool list / execution gating 共用同一套 deferred helper
- per-model-call late binding、mode-aware searchable/loadable pool、startup cancel、freshness rebuild、resume-safe bundle lifecycle 已收敛

下一阶段不再解决“能不能工作”的问题，而是对齐 `claude-code` 已经证明有效的下一层能力：

1. 把“每次模型调用都注入完整 deferred MCP 名字列表”升级成增量同步，减少 prompt churn 和 cache bust。
2. 把 MCP runtime 状态提示从全量重述升级成 instructions delta，只在状态变化时提示模型。
3. 把 deferred 体系从“仅 MCP 子集”扩到更通用的 `shouldDefer / alwaysLoad` 框架，为后续 builtin/system tools 的按需暴露打基础。

本文件是 phase-2 的可执行计划，不回滚或重写 phase-1。

## 2. Reference Targets

`claude-code` 中本阶段最值得参考的实现点：

- [`/Users/zhaoziqian/claude-code/src/tools/ToolSearchTool/ToolSearchTool.ts`](/Users/zhaoziqian/claude-code/src/tools/ToolSearchTool/ToolSearchTool.ts)
  - `tool_search` 的 exact-name / keyword / `select:` 语义与搜索评分。
- [`/Users/zhaoziqian/claude-code/src/tools/ToolSearchTool/prompt.ts`](/Users/zhaoziqian/claude-code/src/tools/ToolSearchTool/prompt.ts)
  - `isDeferredTool`、`alwaysLoad`、`shouldDefer` 的统一判定。
- [`/Users/zhaoziqian/claude-code/src/utils/toolSearch.ts`](/Users/zhaoziqian/claude-code/src/utils/toolSearch.ts)
  - `deferred_tools_delta` announced/discovered 集合重建。
- [`/Users/zhaoziqian/claude-code/src/utils/attachments.ts`](/Users/zhaoziqian/claude-code/src/utils/attachments.ts)
  - deferred tools delta 的生成与增量注入。
- [`/Users/zhaoziqian/claude-code/src/utils/mcpInstructionsDelta.ts`](/Users/zhaoziqian/claude-code/src/utils/mcpInstructionsDelta.ts)
  - MCP instructions 的增量提示。
- [`/Users/zhaoziqian/claude-code/src/services/api/claude.ts`](/Users/zhaoziqian/claude-code/src/services/api/claude.ts)
  - 每轮请求前基于当前状态生成 deferred announcement / delta，而不是把完整列表静态写死在 prompt 里。

`starxo` 中对应的落点：

- [`/Users/zhaoziqian/starxo/internal/tools/dynamic_mcp_surface.go`](/Users/zhaoziqian/starxo/internal/tools/dynamic_mcp_surface.go)
- [`/Users/zhaoziqian/starxo/internal/tools/tool_search.go`](/Users/zhaoziqian/starxo/internal/tools/tool_search.go)
- [`/Users/zhaoziqian/starxo/internal/service/chat.go`](/Users/zhaoziqian/starxo/internal/service/chat.go)
- [`/Users/zhaoziqian/starxo/internal/context/engine.go`](/Users/zhaoziqian/starxo/internal/context/engine.go)
- [`/Users/zhaoziqian/starxo/internal/agent/prompts.go`](/Users/zhaoziqian/starxo/internal/agent/prompts.go)
- [`/Users/zhaoziqian/starxo/internal/model/session_data.go`](/Users/zhaoziqian/starxo/internal/model/session_data.go)

## 3. Current Gaps

phase-1 之后，`starxo` 还存在三个明确的“比 `claude-code` 更粗糙”的点：

### 3.1 Full announcement on every model call

当前 [`/Users/zhaoziqian/starxo/internal/tools/dynamic_mcp_surface.go`](/Users/zhaoziqian/starxo/internal/tools/dynamic_mcp_surface.go) 仍然在每次模型调用前注入完整 `<available-deferred-mcp-tools>` 名单。

直接后果：

- searchable pool 轻微变化就会重发完整名字列表
- prompt cache 更容易抖动
- compaction/windowing 后没有“之前已公告哪些 deferred tools”的稳定重建语义

### 3.2 No MCP instructions delta

当前 `starxo` 只有“可搜索的 deferred MCP canonical names announcement”，没有“pending/auth/runtime state 变化”的增量提示层。

直接后果：

- server 从 `pending -> connected`、`connected -> needs_auth` 等状态变化时，只能靠下一轮完整 announcement 间接体现
- 模型不知道“为什么某个之前能搜到的工具现在不可用了”，只能依赖 tool call error 文案补救

### 3.3 Deferred framework still MCP-only

phase-1 的范围固定为 cc 的 MCP deferred 子集，不覆盖 builtin `shouldDefer`。

这在 phase-1 是正确的，但 phase-2 如果继续参考 `claude-code`，下一步就要把：

- `alwaysLoad`
- `shouldDefer`
- MCP deferred

收成一套统一规则，而不是维持“只有 MCP 可以 deferred”的特殊通道。

## 4. Design Principles

phase-2 固定遵守以下设计原则：

1. 不把 `claude-code` 的 attachment 机制原样照搬到 `starxo`。  
   `starxo` 没有同构的 attachment/message protocol；phase-2 采用 `starxo` 原生实现：
   - session-persisted delta state
   - synthetic delta message injection
   - compaction/reload 时基于 state 重建

2. 不回退 phase-1 的边界。  
   以下语义保持不变：
   - `SessionData.DiscoveredTools` 仍是 discovery 唯一持久化状态源
   - shared runner / bundle 不保存 session discovery
   - startup cancel、freshness rebuild、resume-safe bundle lifecycle 不重做

3. phase-2 先解决“增量同步”，再解决“泛化 deferral”。  
   也就是先做：
   - deferred tools delta
   - MCP instructions delta
   再做：
   - 通用 `shouldDefer / alwaysLoad`

4. 所有增量提示都必须可重建。  
   不允许依赖“内存里曾经发过什么”作为唯一真相；reload / save / compact 后必须仍能恢复正确语义。

## 5. Phase 2A: Deferred Tools Delta

### 5.1 Goal

把当前“每轮全量 `<available-deferred-mcp-tools>`”升级成“只在 searchable pool 变化时发送 delta；无变化时不重发”。

### 5.2 Starxo-native state model

新增 session 级 announcement state：

- `SessionData.DeferredAnnouncementState`
- `SessionRun.deferredAnnouncementState`

字段固定为：

- `AnnouncedSearchableCanonicalNames []string`

规则固定为：

- 这份 state 只记录“已经向模型公告过的 searchable deferred canonical names”
- 持久化前先去重并稳定排序
- 空 state 固定表示为稳定排序后的空切片，不混用 `nil`
- 与 `DiscoveredTools` 分离，不混用
- 会话恢复时从 `SessionData` hydrate
- 旧 session 缺该字段时按“无 prior announcement state”处理，不报错不中断 restore
- compaction/reload 后允许基于这份 state 继续生成 delta，而不是退回每轮全量 announcement

### 5.3 Delta semantics

每次模型调用前基于：

- 当前 `searchablePoolForMode`
- 当前 `DeferredAnnouncementState`

计算：

- `addedNames`
- `removedNames`
- `isBootstrap`

注入规则固定为：

- 第一次 announcement：发送 full snapshot delta
- 后续仅当 `addedNames` 或 `removedNames` 非空时才注入
- 若无变化，则不注入 deferred tools announcement

消息形式不照搬 `claude-code` attachment，而是 `starxo` synthetic `schema.UserMessage`。wire 固定为：

```text
<deferred-tools-delta>
mode: bootstrap|delta
added:
<canonical-name-per-line>
removed:
<canonical-name-per-line>
</deferred-tools-delta>
```

固定规则：

- bootstrap：当前 searchable 全量写入 `added`，`removed` 为空
- delta：只写真实变化集合
- `added` / `removed` 一律稳定排序
- 固定段落标签必须保留，不省略空段落

### 5.4 Persistence / rebuild semantics

delta state 必须满足：

- SaveSessionByID 时随 session snapshot 一起原子落盘
- snapshot 中 `DeferredAnnouncementState`、`DiscoveredTools`、timeline / `ctxEngine` / streaming 都来自同一份 session snapshot
- compact / reload 后可基于 `AnnouncedSearchableCanonicalNames` 重建“之前已公告集合”
- 若旧 session 缺 state 且当前 view 为空，则本轮不发消息；待本轮模型调用成功建立后写入规范化空 state，避免后续反复 bootstrap

### 5.5 Execution points

实现入口固定为：

- [`/Users/zhaoziqian/starxo/internal/tools/dynamic_mcp_surface.go`](/Users/zhaoziqian/starxo/internal/tools/dynamic_mcp_surface.go)
  - 当前全量 announcement 改成 deferred tools delta synthetic message
  - searchable canonical names 的规范化与 delta 计算固定收成单点 helper，不允许不同调用点各算各的
- [`/Users/zhaoziqian/starxo/internal/service/chat.go`](/Users/zhaoziqian/starxo/internal/service/chat.go)
  - session hydrate/save 时带上 announcement state
  - synthetic message 只在 `Generate(...)` 成功返回消息、或 `Stream(...)` 成功返回 stream reader 后推进 state
- [`/Users/zhaoziqian/starxo/internal/model/session_data.go`](/Users/zhaoziqian/starxo/internal/model/session_data.go)
  - 新增持久化字段

### 5.6 Acceptance

完成后必须满足：

- searchable pool 不变时，不再每轮重发完整 deferred tool 名单
- searchable pool 变化时，只发送变化部分
- reload / compact 后不会丢失“已公告集合”
- `tool_search` / execution gating / visible tool list 仍共用同一 deferred helper，不引入第二套工具可见性逻辑

## 6. Phase 2B: MCP Instructions Delta

### 6.1 Goal

把 runtime-dependent MCP state 变化从“隐含在 tool visibility 里”提升成显式、增量、可重建的 instructions delta。

### 6.2 What counts as an instruction change

至少覆盖：

- pending server 进入 connected
- connected server 进入 failed / needs_auth / disabled
- pending server 获得 cached tool metadata，因此首次贡献具体 searchable names
- plan mode / default mode 切换导致 searchable pool 显著变化

### 6.3 State model

新增 session 级 `MCPInstructionsDeltaState`，固定包含：

- `LastAnnouncedSearchableServers []string`
- `LastAnnouncedPendingServers []string`
- `LastAnnouncedUnavailableServers []string`
- `LastInstructionsFingerprint string`

语义固定为：

- 只记录对模型有意义的 instructions summary
- 不持久化原始错误文本
- 不持久化完整 tool schema
- 三组 server 集合都使用稳定排序后的规范化表示
- fingerprint 只能基于规范化后的三组集合计算

### 6.4 Delta content

只允许提示这三类信息：

- 哪些 MCP servers 现在可搜索
- 哪些 servers 仍在 pending
- 哪些 servers 当前不可用，以及简短原因类别（如 `needs_auth` / `failed`）

禁止：

- 泄漏 schema
- 泄漏长错误栈
- 把具体资源内容塞进 instructions delta

### 6.5 Execution points

实现落点固定为：

- [`/Users/zhaoziqian/starxo/internal/tools/dynamic_mcp_surface.go`](/Users/zhaoziqian/starxo/internal/tools/dynamic_mcp_surface.go)
  - 在 deferred tools delta 之外，再注入一个 MCP instructions delta synthetic message
  - server summary 规范化与 fingerprint 计算必须收成单点 helper
- [`/Users/zhaoziqian/starxo/internal/agent/prompts.go`](/Users/zhaoziqian/starxo/internal/agent/prompts.go)
  - 把提示词里的静态描述改成“模型会收到 deferred tools delta + MCP instructions delta”的骨架文案

### 6.6 Acceptance

完成后必须满足：

- MCP runtime state 改变时，模型能收到增量解释，而不是只能靠失败后的 tool call 错误理解
- 无状态变化时不重复发送
- reload / compact 后仍能正确重建下一次 delta

## 7. Phase 2C: General Deferred Framework

### 7.1 Goal

把 deferred 框架从“仅 MCP 子集”推进到统一的 `alwaysLoad / shouldDefer / MCP deferred` 模型。

### 7.2 Rule model

统一规则固定为：

- `alwaysLoad == true`：永不 deferred
- `tool_search`：永不 deferred
- MCP tools：默认 deferred，除非显式 `alwaysLoad`
- 其他工具：按 `shouldDefer` 判断

### 7.3 Catalog changes

`ToolCatalog` 新增或固定以下 metadata：

- `AlwaysLoad bool`
- `ShouldDefer bool`
- `DeferReason string`
- `ToolClass`，至少区分：
  - builtin
  - mcp_action
  - mcp_resource

### 7.4 ToolSearch scope expansion

`tool_search` 从“只搜索 deferred MCP tools”扩展到“搜索所有 deferred tools”，但 phase-2 实施顺序要求：

1. 这一批只做 framework-first
2. 不修改任何真实生产顶层非 MCP 工具的默认暴露状态
3. 非 MCP deferred 只通过 test-only / hidden sample entries 验证
4. 默认生产 registry 不注册这些 sample entries

输出兼容要求：

- MCP canonical name 规则不变
- 对非 MCP deferred tools，名称直接使用 tool name
- exact-name / `select:` / partial-hit 语义不回退

### 7.5 Acceptance

完成后必须满足：

- `alwaysLoad` 真正成为统一的 opt-out 规则
- hidden/test-only 的非 MCP deferred sample 也能被 `tool_search` 激活
- plan mode 对只读约束仍只影响 MCP deferred pool，不误伤非 MCP builtin tools
- prompt / runtime wording 不再错误宣称“只有 deferred MCP tools”

## 8. Branch / Commit Plan

phase-2 建议拆成三条实现分支，按顺序进入 `dev`：

1. `feat/deferred-tools-delta`
   - 目标：Phase 2A
   - 提交建议：
     - `feat(session): persist deferred announcement state`
     - `feat(agent): emit deferred tool deltas instead of full announcements`
     - `test(docs): cover deferred tools delta persistence and compaction rebuild`

2. `feat/mcp-instructions-delta`
   - 目标：Phase 2B
   - 提交建议：
     - `feat(agent): emit MCP instructions deltas from runtime state changes`
     - `test(docs): cover MCP instructions delta rebuild and mode-aware state changes`

3. `feat/general-deferred-framework`
   - 目标：Phase 2C
   - 提交建议：
     - `refactor(tools): generalize alwaysLoad and shouldDefer metadata`
     - `feat(tools): extend tool_search to non-MCP deferred tools`
     - `test(docs): cover generic deferred tool loading semantics`

集成顺序固定为：

- feature branch -> `dev`
- `dev` 稳定后再统一 fast-forward 到 `master`

## 9. Test Plan

### 9.1 Deferred tools delta

- searchable pool 不变时，不发送新的 delta
- searchable pool 增加一项时，只发送 added delta
- searchable pool 删除一项时，只发送 removed delta
- reload 后基于 persisted announcement state 继续正确生成 delta
- compact 后不会退回“每轮 full announcement”

### 9.2 MCP instructions delta

- pending -> connected 时，发送 runtime state delta
- connected -> needs_auth / failed 时，发送 runtime state delta
- pending 且无 cached metadata 时，只提示 server pending，不提示具体工具名
- 纯 tool-level 变化但 server-summary 不变时，只发 deferred tools delta
- 纯 server-summary 变化但 tool-level 不变时，只发 instructions delta
- 同一 server 原始错误文本变化但 reason-class 不变时，不重复发送 instructions delta

### 9.3 General deferred framework

- `alwaysLoad == true` 的工具永不进入 deferred pool
- 非 MCP hidden/test-only sample entry 可被 `tool_search` 激活
- exact-name / `select:` / partial-hit 语义对非 MCP deferred tools 同样成立
- `tool_search` 的 canonical output / pending server 语义对 MCP 不回退
- plan mode 不会错误过滤非 MCP hidden/test-only sample entry

### 9.4 Regression

- `SessionData.DiscoveredTools` 语义不回退
- startup cancel / detached bundle task lifecycle 不回退
- freshness fallback loop fix 不回退
- `go test ./internal/... -count=1`

### 9.5 Manual smoke

- `wails dev` 下 searchable pool 变化时，不再每轮重发完整 deferred tool 名单
- MCP server 从 pending 变 connected 后，模型能收到增量提示而不是只能靠 tool call 报错理解
- 2C 合入后，真实生产顶层非 MCP 工具行为保持不变

## 10. Non-Goals

phase-2 明确不做：

- 重写 startup cancel / freshness / bundle lifecycle 主线
- 重写 pinned-prefix 注入架构
- 把 `starxo` 改造成与 `claude-code` 完全同构的 attachment pipeline
- 一次性把所有 builtin/system tools 都改造成 deferred

## 11. Exit Criteria

phase-2 结束的判断标准是：

1. `starxo` 不再每轮全量重发 deferred MCP searchable names。
2. MCP runtime state 变化对模型是可见的，并且是增量可重建的。
3. deferred framework 不再是 MCP-only 特例，而是统一的 `alwaysLoad / shouldDefer` 规则。
4. 以上三点都在 `dev` 上稳定通过自动测试和最小手工 smoke 后，再进入 `master`。
