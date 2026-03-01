# 插件/扩展体系

## 目标

让用户能够扩展 Agent 的能力，超越当前内置工具和 MCP 协议的范围。提供一个安全、易用的插件框架，支持自定义工具注册、插件生命周期管理和配置界面。

## 现状分析

当前 Starxo 已有的扩展基础：

- `ToolRegistry` (`internal/tools/registry.go`) 已支持三类工具源：`builtins`、`mcpTools`、`custom`
- `RegisterCustom(name, tool)` 方法已存在但尚未对外暴露
- MCP 协议支持已实现（`internal/tools/mcp.go`），可连接外部 MCP 服务器
- 工具通过 Eino 的 `tool.BaseTool` 接口统一抽象

### 与 MCP 的关系和区别

| 维度 | MCP | 插件系统 |
|------|-----|---------|
| 协议 | 标准化的 Model Context Protocol | Starxo 自有插件 API |
| 运行方式 | 独立进程（stdio/SSE 传输） | 同进程加载或沙箱进程 |
| 开发成本 | 需要实现完整 MCP 服务器 | 仅需实现工具接口 |
| 适用场景 | 通用工具、跨应用复用 | Starxo 专属扩展、深度集成 |
| 安全模型 | 进程隔离 | 需要显式权限声明 |

**定位**：MCP 用于连接外部服务生态；插件系统用于轻量级、深度集成的自定义扩展。两者互补而非替代。

---

## Phase 1: 插件 API 设计（优先级 P2）

### 工具插件接口

```go
// Plugin 定义了一个 Starxo 插件的基本契约
type Plugin interface {
    // Metadata 返回插件元数据
    Metadata() PluginMetadata
    // Init 初始化插件，接收插件运行时上下文
    Init(ctx PluginContext) error
    // Tools 返回插件提供的工具列表
    Tools() []tool.BaseTool
    // Shutdown 清理资源
    Shutdown() error
}

type PluginMetadata struct {
    Name        string   // 唯一标识
    Version     string   // 语义化版本
    Description string   // 描述
    Author      string   // 作者
    Permissions []string // 所需权限声明
}

type PluginContext struct {
    Config    map[string]any       // 用户配置
    Operator  commandline.Operator // 沙箱命令执行（受限）
    Logger    *slog.Logger         // 日志
}
```

### 扩展 ToolRegistry

- 新增 `RegisterPlugin(plugin Plugin)` 方法，管理插件生命周期
- 插件工具通过 `plugin:<name>:<tool>` 命名空间隔离
- 插件卸载时自动清理已注册工具

---

## Phase 2: 插件发现与加载机制（优先级 P2）

### 加载方式

1. **本地目录扫描**：扫描 `~/.starxo/plugins/` 目录
2. **配置文件声明**：在 `config.json` 中声明插件路径/URL
3. **Go 插件（`plugin` 包）**：编译为 `.so`/`.dll`，运行时加载
4. **脚本插件**：通过 JSON Schema 描述工具，脚本实现执行逻辑

### 插件清单文件

```json
{
  "name": "my-custom-tool",
  "version": "1.0.0",
  "description": "自定义代码分析工具",
  "entry": "plugin.so",
  "permissions": ["sandbox:exec", "sandbox:read"],
  "config_schema": {
    "type": "object",
    "properties": {
      "timeout": { "type": "integer", "default": 30 }
    }
  }
}
```

### 插件生命周期

```
发现 -> 校验元数据 -> 加载 -> Init() -> 注册工具 -> 运行中 -> Shutdown() -> 卸载
```

---

## Phase 3: 插件配置 UI（优先级 P3）

### Settings Panel 扩展

- 在 `SettingsPanel.vue` 中新增"插件"标签页
- 展示已安装插件列表（名称、版本、状态、权限）
- 每个插件的配置表单根据 `config_schema` 动态渲染
- 支持插件启用/禁用/卸载操作
- 插件权限审批界面

### 前端组件

```
frontend/src/components/settings/
  PluginList.vue        -- 插件列表
  PluginConfig.vue      -- 单个插件配置
  PluginPermissions.vue -- 权限审批
```

---

## Phase 4: 安全考虑（优先级 P1）

### 威胁模型

| 威胁 | 缓解措施 |
|------|---------|
| 恶意插件执行任意代码 | 权限声明 + 用户审批 |
| 插件访问宿主文件系统 | 限制文件操作范围到沙箱内 |
| 插件泄露 API Key | 插件上下文不暴露 LLM 凭证 |
| 插件占用过多资源 | 执行超时 + 资源限制 |
| 插件间互相干扰 | 命名空间隔离 + 独立配置域 |

### 权限模型

```
sandbox:exec   -- 在沙箱中执行命令
sandbox:read   -- 读取沙箱文件
sandbox:write  -- 写入沙箱文件
network:http   -- 发起 HTTP 请求
config:read    -- 读取应用配置（脱敏）
```

插件必须在清单中声明所需权限，用户安装时显式授权。运行时超出权限范围的操作将被拒绝。
