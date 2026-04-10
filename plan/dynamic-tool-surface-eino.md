# `starxo` 基于 Eino 的 Dynamic Tool Surface + ToolSearch + MCP Resources 技术方案

## 1. 背景与目标

`starxo` 当前已经具备多会话、Plan Mode、MCP、interrupt/resume、远程 SSH + Docker sandbox、基础文件和命令执行能力，但工具面仍然是“构建时一次性注入、运行期整体暴露”的形态。这个形态在 MCP server 数量增加后会出现三个直接问题：

1. 模型首轮可见工具过多，`WithTools(...)` 的上下文成本快速膨胀。
2. MCP tools 没有统一 metadata 层，无法做按 server、风险、读写属性的选择和治理。
3. `starxo` 只接入了 MCP tool call，没有 MCP resources 层，很多“先发现资源、再读取资源”的工作流只能退化成外部工具实现。

本设计文档的目标是给出一套可直接落地到 `starxo` 的实现方案，围绕三件事展开：

- 动态工具面：模型默认只看到少量核心工具，按需加载动态工具。
- `ToolSearch`：通过自然语言检索激活动态工具，而不是把整个工具库直接暴露给模型。
- MCP resources：让 agent 能显式列出资源、列出资源模板、读取资源。

本文档面向 `starxo` 工程实现，不是产品概念文档。所有关键决策都在文中固定，不把核心架构选择留给实现阶段。

## 2. 当前 `starxo` 现状与问题

### 2.1 当前工具装配路径

当前工具装配主路径在 `internal/service/chat.go`：

1. 创建 `ToolRegistry`
2. 注册 built-in tools
3. 依次连接 `cfg.MCP.Servers`
4. 通过 `LoadMCPTools(...)` 拉取 MCP tools
5. `registry.GetAll()`
6. 整体作为 `extraTools` 传入 `BuildDeepAgentForMode(...)`

当前实现的结果是：默认模式下，顶层 deep agent 会直接拿到 registry 里的全部额外工具；plan mode 顶层则不拿这些工具。

### 2.2 当前注册层的问题

`internal/tools/registry.go` 当前只维护：

- `map[string]tool.BaseTool`
- `map[string][]tool.BaseTool`

它缺少以下能力：

- 工具来源 metadata
- 原始远端 tool name 与本地 canonical name 的映射
- `read-only / destructive / open-world` 等风控属性
- `always-loaded / dynamic / hidden` 暴露策略
- 按 server 聚合的资源与模板能力

结论：当前 `ToolRegistry` 不能支持动态工具面，必须升级为 metadata-aware 的 `ToolCatalog`。

### 2.3 当前 MCP 接入的问题

`internal/tools/mcp.go` 当前只做两件事：

- `ConnectMCPServer(...)`
- `LoadMCPTools(...)`

其中 `LoadMCPTools(...)` 直接依赖 `github.com/cloudwego/eino-ext/components/tool/mcp/officialmcp`。该扩展包只封装了：

- `ListTools`
- `CallTool`

它没有封装：

- `ListResources`
- `ListResourceTemplates`
- `ReadResource`
- tools/resources 的分页遍历
- tool annotations 的 catalog 化保留

结论：MCP resources 必须由 `starxo` 自己实现 adapter 层，不能依赖 `officialmcp` 直接完成。

### 2.4 当前 prompt 与真实工具面的失配

`internal/agent/prompts.go` 当前把顶层 agent 和子 agent 的工具说明写死在 prompt 里。问题在于：

- 顶层 prompt 是固定文案，不随工具集变化
- 子 agent prompt 里写了自有工具，顶层 prompt 里没有动态 catalog 认知
- 一旦引入 deferred tools，模型可见工具和 prompt 声明会立刻失配

结论：prompt 必须改成“固定骨架 + tool surface summary 注入”模式。

### 2.5 当前 Eino 版本的结构限制

`starxo` 当前依赖 `github.com/cloudwego/eino v0.7.36`。该版本只提供旧的 `AgentMiddleware` 机制，核心限制是：

1. `AdditionalTools` 在 `NewChatModelAgent(...)` 时一次性合并进 `ToolsConfig.Tools`
2. 没有 `WrapModel(...)` 可以在每轮模型调用前重写 `WithTools(...)`
3. 没有官方 `dynamictool/toolsearch` middleware

这意味着 `v0.7.36` 只能做“提示词级别的工具搜索”和“执行层兜底”，做不了真正的 deferred dynamic tool surface。

结论：本方案明确采用 `Eino 0.8.x` 路线，不在 `v0.7.36` 上做伪动态实现。

## 3. Eino / Eino-ext / MCP SDK 调研结论

### 3.1 Eino 0.8.x 已具备动态工具所需接入面

基于 `cloudwego/eino@v0.8.1` 源码，动态工具面所需的关键能力已经具备：

- `adk.ChatModelAgentMiddleware`
- `BeforeAgent(...)`
- `BeforeModelRewriteState(...)`
- `AfterModelRewriteState(...)`
- `WrapModel(...)`
- `WrapInvokableToolCall(...)`
- `WrapStreamableToolCall(...)`

其中 `WrapModel(...)` 是动态工具面成立的关键，因为它允许在每轮模型调用时基于历史消息裁剪传给模型的 tool list。

### 3.2 Eino 官方已有 `toolsearch` 中间件原型

`cloudwego/eino@v0.8.1/adk/middlewares/dynamictool/toolsearch/toolsearch.go` 已给出官方原型：

- 注入 `tool_search`
- 默认隐藏 dynamic tools
- 当模型调用 `tool_search` 后，根据历史消息把匹配工具重新加入模型可见工具集

这个实现证明两点：

1. 动态工具面不需要重写 agent graph，可以通过 middleware 挂到现有 `deep.New(...)`
2. `tool_search` 的正确落点是 model-side tool filtering，而不是只在 prompt 中引导模型

### 3.3 Eino examples 已给出 unknown tool 和 plan-execute 参考

`cloudwego/eino-examples` 里有两个直接相关的官方例子：

- `flow/agent/react/unknown_tool_handler_example`
- `flow/agent/multiagent/plan_execute`

前者说明 `compose.ToolsNodeConfig.UnknownToolsHandler` 可以在模型 hallucinate 未知工具时提供可恢复反馈；后者说明 planner / executor / replanner 的标准组合无需为动态工具做结构性重写。

### 3.4 `officialmcp` 的能力边界

`cloudwego/eino-ext/components/tool/mcp/officialmcp/mcp.go` 的真实能力边界是：

- 从 `ClientSession.ListTools(...)` 拉取 tools
- 把 MCP tool schema 转成 `schema.ToolInfo`
- 调用 `ClientSession.CallTool(...)`

它不负责：

- 分页拿全量 tools
- 资源层
- 模板层
- 统一命名
- annotations 治理

因此，`officialmcp` 最多只能作为“参考实现”，不是本方案的核心依赖。

### 3.5 MCP Go SDK 已提供完整资源 API

`github.com/modelcontextprotocol/go-sdk/mcp` 的 `ClientSession` 已提供：

- `ListResources(...)`
- `ListResourceTemplates(...)`
- `ReadResource(...)`

协议层对象还提供了：

- `Resource`
- `ResourceTemplate`
- `ResourceContents`
- `ToolAnnotations`

这意味着 `starxo` 没有协议级障碍，MCP resources 可以直接落地。

## 4. 总体架构决策

### 4.1 固定决策

本方案固定采用以下决策：

1. 升级到 `Eino 0.8.x`，推荐锁定 `v0.8.1` 作为第一实现版本。
2. 动态工具面基于 `adk.ChatModelAgentMiddleware` 实现。
3. `tool_search` 对外接口使用“自然语言检索 + 结构化过滤”，不使用 regex-only 接口。
4. MCP action tools 统一使用 `mcp__<server>__<tool>` canonical name。
5. MCP resources 单独实现 `list_mcp_resources`、`list_mcp_resource_templates`、`read_mcp_resource`。
6. 顶层 agent 不直接暴露代码编辑/执行 builtin；这些能力继续由 sub-agent 持有。
7. plan mode 的 executor 只允许 `read-only` dynamic tools + MCP resources。

### 4.2 总体架构

整体新增 5 个核心层：

1. `ToolCatalog`
2. `MCP runtime / adapter`
3. `DynamicToolSurfaceMiddleware`
4. `MCP resource tools`
5. `prompt surface renderer`

工具面划分为四类：

- `always-loaded orchestration tools`
- `always-loaded meta/resource tools`
- `dynamic MCP action tools`
- `subagent-private execution/code tools`

### 4.3 为什么不改写现有 runner / agent 架构

`starxo` 当前已经是：

- default mode: `deep.New(...)`
- plan mode: `planexecute.New(...)`，其中 deep agent 作为 executor

这个结构本身没有问题。动态工具面真正要解决的是“工具如何进入模型上下文”，不是“agent graph 如何编排”。

因此本方案明确不做以下事情：

- 不重写 `BuildPlanRunner(...)`
- 不重写 planner / replanner
- 不把 tool search 迁移到外部 workflow graph

动态工具面只挂在 deep agent 上即可。

## 5. 动态工具面设计

### 5.1 工具分类

顶层工具严格分为三层：

#### A. Always-loaded orchestration tools

这类工具默认始终暴露给顶层 agent：

- `ask_user`
- `ask_choice`
- `notify_user`
- `write_todos`
- `update_todo`

#### B. Always-loaded meta / resource tools

这类工具同样始终暴露：

- `tool_search`
- `list_mcp_resources`
- `list_mcp_resource_templates`
- `read_mcp_resource`

#### C. Dynamic MCP action tools

这类工具默认不暴露，必须通过 `tool_search` 激活：

- 各个 MCP server 暴露的 action tools
- 本地会以 canonical name 存储
- model-side 默认隐藏
- execution-side 必须校验“已激活”状态

### 5.2 顶层 agent 与 sub-agent 的边界

当前 `starxo` 顶层 agent 通过 `extraTools` 可以直接拿到 registry 中的 builtin 文件/执行工具。这与 `prompts.go` 中“代码与执行由 sub-agent 完成”的定位不一致。

本方案固定如下边界：

- 顶层 agent：只负责 orchestration、tool discovery、MCP resource access、受控 MCP action 调用
- `code_writer`：代码读取/编辑
- `code_executor`：命令/脚本执行
- `file_manager`：非代码批量文件操作

这意味着当前 registry 里的 builtin 执行/文件工具不能继续整体注入顶层。

### 5.3 Dynamic surface 的运行语义

本方案采用“替换式激活”而不是“并集式激活”：

- 每次 `tool_search` 都会产生一个新的 dynamic selection
- 新 selection 替换旧 selection
- 下一轮模型只看到新的 selected dynamic tools

选择替换而不是并集，有三个原因：

1. 保持 tool surface 始终有界
2. 避免历史搜索结果越积越多，重新回到大工具面
3. 让模型更容易建立“当前工具箱”的稳定心智模型

### 5.4 Model-side 与 execution-side 双重限制

本方案不只做 model-side 隐藏，还要做 execution-side 强约束：

- model-side：`WrapModel(...)` 裁剪 `WithTools(...)`
- execution-side：`WrapInvokableToolCall(...)` / `WrapStreamableToolCall(...)` 校验 dynamic tool 是否在当前 selection 里

原因是：

- prompt 泄漏时模型仍可能幻想出隐藏工具名
- unknown tool handler 只能兜底“完全未知”的工具名
- 对“catalog 中存在但当前未激活”的工具，必须返回明确错误并要求先 `tool_search`

### 5.5 Unknown tool 处理策略

保留 `compose.ToolsNodeConfig.UnknownToolsHandler`，但分成两类响应：

1. 完全未知工具名：
   - 返回“该工具不存在，请重新选择有效工具”
2. catalog 内存在但当前未激活：
   - 返回“该工具当前未加载，请先调用 tool_search 激活”

这两类错误都必须是“可恢复”的，不应直接中断整个 agent run。

## 6. ToolSearch 设计

### 6.1 设计目标

`tool_search` 的职责不是直接执行工具，而是：

- 帮模型理解当前系统有哪些按需能力
- 在不暴露完整大工具面的前提下，把需要的动态工具激活
- 给模型一个稳定、结构化、低 token 成本的 discovery 接口

### 6.2 对外输入接口

`tool_search` 的输入固定为自然语言检索，不使用 regex-only 接口。

建议输入结构：

```go
type ToolSearchInput struct {
	Query        string   `json:"query"`
	Server       string   `json:"server,omitempty"`
	Sources      []string `json:"sources,omitempty"`
	ReadOnlyOnly bool     `json:"read_only_only,omitempty"`
	Limit        int      `json:"limit,omitempty"`
}
```

字段约束：

- `Query` 必填
- `Server` 可选，用于多 server 场景精确过滤
- `Sources` 可选，允许值固定为 `builtin`、`mcp`、`custom`
- `ReadOnlyOnly` 可选
- `Limit` 可选，默认 12，最大 20

### 6.3 输出接口

```go
type ToolSearchMatch struct {
	Name          string   `json:"name"`
	Title         string   `json:"title,omitempty"`
	Description   string   `json:"description"`
	Source        string   `json:"source"`
	Server        string   `json:"server,omitempty"`
	ReadOnly      bool     `json:"read_only"`
	Destructive   bool     `json:"destructive"`
	OpenWorld     bool     `json:"open_world"`
	AlreadyLoaded bool     `json:"already_loaded"`
	Tags          []string `json:"tags,omitempty"`
}

type ToolSearchOutput struct {
	ActivatedTools []string          `json:"activated_tools"`
	Matches        []ToolSearchMatch `json:"matches"`
	Replaced       bool              `json:"replaced"`
}
```

语义：

- `Matches` 用于解释搜索命中结果
- `ActivatedTools` 是本轮真正被激活的 tool canonical names
- `Replaced=true` 明确说明新结果替换旧 selection

### 6.4 排序策略

v1 不做 embedding 或向量召回，只做确定性的 lexical ranking：

1. exact canonical name match
2. exact remote name / title match
3. prefix match
4. substring match
5. tag match
6. description token hit

如果分数相同，则按以下顺序稳定排序：

1. `already_loaded=true`
2. `read_only=true`
3. server name
4. tool name

### 6.5 激活规则

`tool_search` 返回结果后，中间件行为固定为：

1. 记录 `ActivatedTools`
2. 替换当前 dynamic selection
3. 下一轮模型调用只看到新的 selected dynamic tools

如果没有命中：

- 不清空旧 selection
- 返回空的 `ActivatedTools`
- 要求模型改写 query 或放宽 filters

### 6.6 为什么不用 regex-only 接口

官方 `toolsearch` 中间件采用 regex-only 接口，适合作为 framework 原型，但不适合作为 `starxo` 的外部 agent 接口，原因是：

1. 模型需要先自己猜测命名模式，交互不自然
2. 多 server、多 canonical name 情况下，regex 对模型不友好
3. `starxo` 目标是工程型 agent，而不是只做 name-based selection demo

因此这里固定采用自然语言 query。

## 7. MCP 资源层设计

### 7.1 资源层目标

为顶层 agent 新增资源发现和读取能力：

- 列出资源
- 列出资源模板
- 读取资源

这样 MCP server 不需要把所有内容都包装成 action tool，很多“读取知识、配置、文档、索引”的场景可以走更轻量的 resources 通道。

### 7.2 工具清单

固定新增三个工具：

- `list_mcp_resources`
- `list_mcp_resource_templates`
- `read_mcp_resource`

这三个工具始终是 always-loaded meta tools。

### 7.3 `list_mcp_resources`

输入：

```go
type ListMCPResourcesInput struct {
	Server string `json:"server,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
```

输出：

```go
type MCPResourceSummary struct {
	Server      string `json:"server"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mime_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
	URI         string `json:"uri"`
}

type ListMCPResourcesOutput struct {
	Resources  []MCPResourceSummary `json:"resources"`
	NextCursor string               `json:"next_cursor,omitempty"`
}
```

规则：

- 多 server 场景默认要求显式传 `server`
- 单 server 场景允许省略
- 必须支持分页 cursor

### 7.4 `list_mcp_resource_templates`

输入：

```go
type ListMCPResourceTemplatesInput struct {
	Server string `json:"server,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
```

输出：

```go
type MCPResourceTemplateSummary struct {
	Server      string `json:"server"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mime_type,omitempty"`
	URITemplate string `json:"uri_template"`
}

type ListMCPResourceTemplatesOutput struct {
	Templates  []MCPResourceTemplateSummary `json:"templates"`
	NextCursor string                      `json:"next_cursor,omitempty"`
}
```

v1 只做列出模板，不做模板展开或变量补全。

### 7.5 `read_mcp_resource`

输入：

```go
type ReadMCPResourceInput struct {
	Server            string `json:"server,omitempty"`
	URI               string `json:"uri"`
	IncludeBlobBase64 bool   `json:"include_blob_base64,omitempty"`
}
```

输出：

```go
type MCPResourceContent struct {
	URI            string `json:"uri"`
	MIMEType       string `json:"mime_type,omitempty"`
	Text           string `json:"text,omitempty"`
	BlobBase64     string `json:"blob_base64,omitempty"`
	Truncated      bool   `json:"truncated"`
	OriginalBytes  int64  `json:"original_bytes,omitempty"`
	ReturnedBytes  int64  `json:"returned_bytes,omitempty"`
}

type ReadMCPResourceOutput struct {
	Server   string               `json:"server"`
	Contents []MCPResourceContent `json:"contents"`
}
```

规则：

- 文本内容按 `MaxResourceReadBytes` 截断
- 二进制内容默认不返回 base64，仅返回 metadata
- 只有显式 `IncludeBlobBase64=true` 时才返回 base64，并且仍受大小限制
- 如果资源内容超过阈值，必须显式标记 `Truncated=true`

### 7.6 capability 探测与错误策略

每个 `MCPServerHandle` 连接后进行一次 capability probe，并缓存以下能力：

- `supports_tools`
- `supports_resources`
- `supports_resource_templates`

若 server 不支持某项能力：

- tool 不从 catalog 中注册对应 resource meta tool
- 或者统一 meta tool 返回明确错误：
  - 指明哪个 server 不支持
  - 列出当前支持该能力的 server

推荐采用第二种做法，这样用户和模型接口稳定，工具名不因 server 能力波动而变化。

## 8. `ChatService / deep agent / plan runner` 集成方案

### 8.1 `ChatService.buildRunnersLocked()` 改造

当前流程：

- registry builtins
- connect MCP
- load MCP tools
- `GetAll()`
- 注入 deep agent

改造后流程固定为：

1. 创建 `MCPServerHandle` 列表
2. 分页拉取 raw MCP tools / resources / templates
3. 构建 `ToolCatalog`
4. 生成 default mode 的 `ToolSurfacePolicy`
5. 生成 plan executor 的 `ToolSurfacePolicy`
6. 构建 default deep agent
7. 构建 plan deep executor
8. 构建 default runner / plan runner

### 8.2 `BuildDeepAgentForMode(...)` 改造

`BuildDeepAgentForMode(...)` 需要从“直接接收 `extraTools []tool.BaseTool`”改成：

- 接收 `alwaysLoadedTools`
- 接收 `dynamicTools`
- 接收 `handlers []adk.ChatModelAgentMiddleware`
- 接收 `prompt surface summary`

deep agent 仍然基于 `deep.New(...)` 创建，不改框架。

### 8.3 default mode 接入策略

default mode 的顶层 agent：

- always-loaded：orchestration + meta/resource tools
- dynamic：MCP action tools
- 不直接拿 subagent 私有 builtin

### 8.4 plan mode 接入策略

plan mode 里的 planner / replanner 不接动态工具。

只有作为 executor 的 deep agent 接入 dynamic surface，但 policy 更严格：

- always-loaded：orchestration + meta/resource tools
- dynamic：只允许 `read_only` 的 MCP action tools

### 8.5 runner 生命周期与 session 回收

当前 `ChatService.invalidateRunners()` 只做：

- `s.deepAgent = nil`
- `s.defaultRunner = nil`
- `s.planRunner = nil`

本方案要求这里额外做：

- 关闭旧的 `MCPServerHandle`
- 关闭旧 `ClientSession`
- 释放 stdio transport 对应的子进程资源

否则每次重建 runner 都会累积 MCP 连接泄漏。

## 9. Prompt 与 Tool Surface Policy 方案

### 9.1 Prompt 生成方式

`internal/agent/prompts.go` 从硬编码工具说明改为：

- 固定角色边界说明
- 固定工作流约束
- 注入动态生成的 tool surface summary

### 9.2 顶层 default mode prompt

必须明确告诉模型：

- 当前始终可用的核心工具有哪些
- 可以通过 `tool_search` 激活更多工具
- `tool_search` 会替换当前动态工具集合
- 如果需要外部知识或文档类内容，优先使用 MCP resources

但 prompt 不应泄漏所有隐藏动态工具的具体名字，只给出类别和 server 级摘要。

### 9.3 顶层 plan mode prompt

计划模式的 executor prompt 需要额外声明：

- 动态工具只允许读取型工具
- 修改型、外部 open-world 高风险工具不在 plan executor surface 内

### 9.4 Tool surface policy

建议定义：

```go
type ToolSurfacePolicy struct {
	AlwaysLoaded         []string
	DynamicCandidates    []string
	AllowReadOnlyOnly    bool
	AllowOpenWorld       bool
	AllowDestructive     bool
	ShowHiddenToolNames  bool
}
```

默认值：

- default mode:
  - `AllowReadOnlyOnly=false`
  - `AllowOpenWorld=true`
  - `AllowDestructive=true`
- plan executor:
  - `AllowReadOnlyOnly=true`
  - `AllowOpenWorld=false`
  - `AllowDestructive=false`

## 10. 数据结构与接口草案

### 10.1 `ToolCatalog`

```go
type ToolCatalog interface {
	All() []CatalogEntry
	Get(name string) (CatalogEntry, bool)
	MustGet(name string) CatalogEntry
	Search(q ToolSearchInput) []CatalogEntry
	BuildSurface(policy ToolSurfacePolicy) BuiltToolSurface
	Close() error
}
```

### 10.2 `CatalogEntry`

```go
type CatalogEntry struct {
	Name        string
	RemoteName  string
	Title       string
	Description string
	Source      string // builtin | mcp | custom | meta
	Server      string
	Exposure    string // always | dynamic | hidden

	ReadOnly    bool
	Destructive bool
	OpenWorld   bool

	Tags          []string
	ParamsSummary []string

	Tool tool.BaseTool
}
```

### 10.3 `MCPServerHandle`

```go
type MCPServerHandle struct {
	Name      string
	Session   *mcp.ClientSession

	SupportsTools             bool
	SupportsResources         bool
	SupportsResourceTemplates bool

	RawTools             []*mcp.Tool
	RawResources         []*mcp.Resource
	RawResourceTemplates []*mcp.ResourceTemplate

	CloseFunc func() error
}
```

### 10.4 `BuiltToolSurface`

```go
type BuiltToolSurface struct {
	AlwaysLoaded []tool.BaseTool
	Dynamic      []tool.BaseTool

	AlwaysLoadedEntries []CatalogEntry
	DynamicEntries      []CatalogEntry

	Summary string
}
```

### 10.5 `ToolSearchInput / Output`

```go
type ToolSearchInput struct {
	Query        string   `json:"query"`
	Server       string   `json:"server,omitempty"`
	Sources      []string `json:"sources,omitempty"`
	ReadOnlyOnly bool     `json:"read_only_only,omitempty"`
	Limit        int      `json:"limit,omitempty"`
}

type ToolSearchMatch struct {
	Name          string   `json:"name"`
	Title         string   `json:"title,omitempty"`
	Description   string   `json:"description"`
	Source        string   `json:"source"`
	Server        string   `json:"server,omitempty"`
	ReadOnly      bool     `json:"read_only"`
	Destructive   bool     `json:"destructive"`
	OpenWorld     bool     `json:"open_world"`
	AlreadyLoaded bool     `json:"already_loaded"`
	Tags          []string `json:"tags,omitempty"`
}

type ToolSearchOutput struct {
	ActivatedTools []string          `json:"activated_tools"`
	Matches        []ToolSearchMatch `json:"matches"`
	Replaced       bool              `json:"replaced"`
}
```

### 10.6 MCP resource tools

```go
type ListMCPResourcesInput struct {
	Server string `json:"server,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type ListMCPResourcesOutput struct {
	Resources  []MCPResourceSummary `json:"resources"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

type ListMCPResourceTemplatesInput struct {
	Server string `json:"server,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type ListMCPResourceTemplatesOutput struct {
	Templates  []MCPResourceTemplateSummary `json:"templates"`
	NextCursor string                      `json:"next_cursor,omitempty"`
}

type ReadMCPResourceInput struct {
	Server            string `json:"server,omitempty"`
	URI               string `json:"uri"`
	IncludeBlobBase64 bool   `json:"include_blob_base64,omitempty"`
}

type ReadMCPResourceOutput struct {
	Server   string               `json:"server"`
	Contents []MCPResourceContent `json:"contents"`
}
```

### 10.7 `DynamicToolSurfaceMiddleware`

```go
type DynamicToolSurfaceMiddleware struct {
	*adk.BaseChatModelAgentMiddleware

	Catalog        ToolCatalog
	AlwaysLoaded   []CatalogEntry
	DynamicEntries []CatalogEntry
	MaxResults     int
}
```

职责固定为：

- `BeforeAgent(...)` 注入 raw tools
- `BeforeModelRewriteState(...)` 解析最后一次 `tool_search`
- `WrapModel(...)` 裁剪模型可见工具
- `WrapInvokableToolCall(...)` / `WrapStreamableToolCall(...)` 校验 dynamic tool 是否已激活

## 11. 实施步骤

### 阶段 1：依赖与接入基线

1. 升级 `go.mod` 到 `Eino 0.8.x`
2. 修正 `deep.New(...)`、`adk.NewChatModelAgent(...)` 的新参数适配
3. 确认 current runner / interrupt / checkpoint 行为不回退

### 阶段 2：构建 `ToolCatalog`

1. 用 metadata-aware catalog 替换当前 `ToolRegistry` 的直出模式
2. 增加 canonical name、server、source、risk flags
3. 增加 surface builder 和 lexical search

### 阶段 3：重构 MCP runtime

1. 分页拉取 raw tools
2. 构建自有 MCP tool adapter
3. 增加 `MCPServerHandle` 生命周期管理
4. 在 runner invalidate 时释放旧 session

### 阶段 4：加入 MCP resource tools

1. capability probe
2. 实现 `list_mcp_resources`
3. 实现 `list_mcp_resource_templates`
4. 实现 `read_mcp_resource`

### 阶段 5：加入 `DynamicToolSurfaceMiddleware`

1. 注入 `tool_search`
2. 实现 selection 解析
3. 实现 model-side tool filtering
4. 实现 execution-side activation 校验
5. 配置 unknown tool handler

### 阶段 6：接入 prompt 和 policy

1. prompt 改为 summary 注入模式
2. default mode / plan executor 使用不同 policy
3. 顶层 agent 移除 subagent 私有 builtin 暴露

### 阶段 7：测试与验收

1. 单测 catalog、adapter、resource tools、middleware
2. 集成测试 default mode / plan mode
3. 观察首轮 tool 数量、name 冲突、session 回收

## 12. 测试方案

### 12.1 `ToolCatalog` 单测

- builtin、MCP、custom 三类 entry 的注册和读取
- canonical name 唯一性
- metadata 正确继承
- lexical ranking 顺序稳定
- `read_only_only` / `server` / `sources` filters 正确

### 12.2 MCP adapter 单测

- `ListTools` 多页拉取
- remote tool name 到 canonical name 的映射
- adapter `Info()` 返回本地 canonical name
- adapter `InvokableRun()` 实际调用远端原始 name

### 12.3 MCP resource tools 单测

- `ListResources` 分页
- `ListResourceTemplates` 分页
- `ReadResource` 文本截断
- `ReadResource` blob metadata 输出
- `IncludeBlobBase64=true` 时 base64 返回受大小限制
- 不支持 resources 的 server 返回明确错误

### 12.4 Middleware 单测

- 初始模型只看到 always-loaded + meta tools
- 调用 `tool_search` 后下一轮模型看到 selected dynamic tools
- 第二次 `tool_search` 替换第一次 selection
- 未激活 dynamic tool 被执行层拒绝
- unknown tool name 走 `UnknownToolsHandler`

### 12.5 集成测试

- default mode 下通过 `tool_search` 激活并调用 MCP action tool
- plan mode executor 只能看到 `read-only` dynamic tools
- `read_mcp_resource` 可用于资源读取工作流
- `invalidateRunners()` 后旧 MCP session 被释放，runner 可重建

## 13. 风险与降级策略

### 13.1 Eino 升级兼容风险

风险：

- `deep.New(...)` 与 `adk.NewChatModelAgent(...)` API 变化
- callback / middleware 行为变化

策略：

- 第一阶段先做依赖升级与最小编译通过
- 升级后先验证 default runner 和 plan runner 的现有行为，再继续功能开发

### 13.2 MCP server 能力不一致

风险：

- 有些 server 没有 resources
- 有些 server 的 tool annotations 不完整
- 有些 server 的 tools/resources list 有分页

策略：

- capability probe + 缓存
- 所有 meta tools 接口稳定，能力不足时返回结构化错误

### 13.3 Prompt 泄漏隐藏工具名

风险：

- prompt 如果直接列出全部工具名，model-side 隐藏就失效

策略：

- prompt 只展示 categories / server summaries，不展示 hidden tool names
- execution-side 再做一次激活校验

### 13.4 资源内容过大

风险：

- 资源文本过长
- 二进制资源直接灌进上下文

策略：

- 强制 `MaxResourceReadBytes`
- blob 默认只给 metadata
- 需要 base64 时显式 opt-in

### 13.5 降级策略

如果 `Eino 0.8.x` 升级阶段阻塞，可以接受的唯一降级路径是：

1. 先交付 `ToolCatalog`
2. 先交付 namespaced MCP adapter
3. 先交付 MCP resource tools

但该降级路径不交付真正的 deferred dynamic tool surface，只能作为临时过渡，不是目标方案。

## 14. 最终实施建议

建议按照以下优先级推进：

### P0

- 升级到 `Eino 0.8.x`
- 构建 `ToolCatalog`
- 实现 namespaced MCP adapter
- 实现 MCP resource tools
- 实现 `DynamicToolSurfaceMiddleware`

### P1

- prompt summary 注入
- default / plan executor policy 分离
- runner invalidate 时的 MCP session 释放

### P2

- 更精细的 ranking
- 模板展开与参数补全
- resources subscribe / updated 通知接入

最终验收必须满足以下条件：

1. 首轮模型可见 tool list 显著缩小
2. MCP tool name 不再冲突
3. 资源层支持分页与截断
4. prompt 与真实 tool surface 一致
5. runner 重建时旧 MCP session 被正确释放

如果以上 5 项未同时满足，则本方案不视为完成。
