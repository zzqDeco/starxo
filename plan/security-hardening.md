# 安全加固

## 现状分析

当前安全状况：

- **凭证明文存储**：`config.json` 中 SSH 密码（`password`）、LLM API Key（`apiKey`）均为明文
- **SSH 密钥处理粗糙**：`PrivateKey` 字段直接存储原始私钥字符串或文件路径（`ssh.go:170-173`）
- **Host Key 不校验**：使用 `ssh.InsecureIgnoreHostKey()`（`ssh.go:53`），易受中间人攻击
- **容器安全依赖配置**：`DockerConfig` 有 `MemoryLimit`/`CPULimit` 但无强制执行验证
- **MCP 服务器无沙箱**：MCP 服务器以 stdio/SSE 运行，继承宿主进程权限
- **无操作审计**：Agent 执行的所有命令、文件修改无审计记录

---

## 改进方向

### 1. 凭证存储加密（优先级 P0）

**当前问题**：`config.go` 定义的 `SSHConfig.Password`、`LLMConfig.APIKey` 直接序列化为 JSON 明文。

#### 改进方案

- **操作系统凭证管理器集成**：
  - Windows：Windows Credential Manager（`wincred`）
  - macOS：Keychain（`security` 命令）
  - Linux：libsecret / GNOME Keyring
  - Go 库：`github.com/zalando/go-keyring`

- **实现架构**：
  ```go
  type SecureConfigStore struct {
      store   *ConfigFileStore  // 非敏感配置
      keyring keyring.Keyring   // 敏感凭证
  }

  // config.json 中存储引用
  // "apiKey": "keyring://starxo/llm-api-key"
  // 实际值存入系统凭证管理器
  ```

- **迁移策略**：首次启动检测明文凭证，自动迁移到凭证管理器，清除 config.json 中的明文

---

### 2. SSH 密钥管理增强（优先级 P0）

**当前问题**：

- `SSHConfig.PrivateKey` 可能包含完整的 PEM 私钥字符串（`ssh.go:229-235`）
- 不支持加密私钥（passphrase-protected keys）
- Host key 验证被禁用

#### 改进方案

- **禁止存储原始私钥**：只存储密钥文件路径，不在配置中保存私钥内容
- **支持加密密钥**：
  ```go
  // 提示用户输入 passphrase
  signer, err := ssh.ParsePrivateKeyWithPassphrase(keyData, passphrase)
  ```
- **Host Key 验证**：
  - 首次连接时保存服务器指纹到 `~/.starxo/known_hosts`
  - 后续连接校验指纹，变更时警告用户（Trust on First Use 模型）
  - 替换 `ssh.InsecureIgnoreHostKey()` 为自定义回调
- **SSH Agent 优先**：鼓励用户使用 SSH Agent，避免密钥直接暴露（当前已支持 `trySSHAgent()`）

---

### 3. 容器逃逸防护和资源限制强制执行（优先级 P1）

#### 当前容器安全配置

`DockerConfig` 设有 `MemoryLimit` 和 `CPULimit`，但需要验证：
- 容器创建时是否正确传递了这些限制参数
- 容器运行时是否受限于这些参数

#### 改进方案

- **安全容器配置**：
  ```bash
  docker run \
    --memory=${MemoryLimit}m \
    --cpus=${CPULimit} \
    --pids-limit=256 \
    --read-only --tmpfs /tmp:rw,noexec,nosuid \
    --security-opt=no-new-privileges \
    --cap-drop=ALL \
    --cap-add=CHOWN,DAC_OVERRIDE,FOWNER,SETGID,SETUID \
    --network=${networkMode} \
    ${image}
  ```
- **网络限制**：
  - 默认禁用网络（`--network=none`），需要时由用户显式开启
  - 开启网络时限制可访问的域名/IP（egress 防火墙规则）
- **文件系统限制**：
  - `/workspace` 设置磁盘配额
  - 只读挂载系统目录
- **运行时验证**：定期检查容器实际资源限制与配置一致

---

### 4. MCP 服务器沙箱和权限模型（优先级 P1）

**当前问题**：MCP 服务器（`MCPServerConfig`）以 stdio 子进程运行，继承 Starxo 的全部权限。

#### 改进方案

- **进程级隔离**：
  - 使用受限用户运行 MCP 服务器进程
  - 限制文件系统访问范围
  - 限制网络访问

- **权限声明**：
  ```json
  {
    "name": "web-search",
    "transport": "stdio",
    "command": "npx",
    "args": ["@mcp/web-search"],
    "permissions": {
      "network": ["https://*.googleapis.com"],
      "filesystem": "none",
      "env_vars": ["GOOGLE_API_KEY"]
    }
  }
  ```

- **工具调用审批**：对高风险 MCP 工具调用（如文件写入、命令执行）增加用户确认步骤

---

### 5. API Key 轮换和多用户支持（优先级 P2）

#### API Key 管理

- **多 Key 支持**：配置多个 API Key，自动轮换使用
- **Key 健康检查**：定期验证 Key 有效性，失效时自动切换
- **用量跟踪**：记录每个 Key 的调用次数和 token 消耗
- **Key 过期提醒**：接近额度限制时提醒用户

#### 多用户支持（远期）

- **用户配置隔离**：每个用户独立的配置文件和凭证
- **会话权限**：不同用户可配置不同的沙箱访问权限
- **审计关联**：操作日志关联到具体用户

---

### 6. Agent 操作审计日志（优先级 P1）

**目标**：记录 Agent 执行的所有关键操作，用于事后审查和合规。

#### 审计事件

| 事件类型 | 记录内容 |
|----------|---------|
| `command.exec` | 执行的命令、容器 ID、退出码、耗时 |
| `file.write` | 文件路径、修改内容摘要 |
| `file.read` | 文件路径 |
| `agent.transfer` | 源 Agent、目标 Agent、原因 |
| `llm.request` | 模型、token 数、耗时 |
| `ssh.connect` | 目标主机、认证方式 |
| `container.create` | 镜像、资源限制 |
| `mcp.tool_call` | 服务器名、工具名、参数 |

#### 存储方案

```go
type AuditEntry struct {
    Timestamp time.Time         `json:"timestamp"`
    SessionID string            `json:"sessionId"`
    EventType string            `json:"eventType"`
    Agent     string            `json:"agent"`
    Details   map[string]any    `json:"details"`
}
```

- 存储到 `logs/audit/` 目录下的 JSON Lines 文件
- 按日期轮转，保留最近 30 天
- 前端提供审计日志查看界面

---

## 实施优先级总结

| 改进 | 优先级 | 安全影响 | 预估工作量 |
|------|--------|---------|-----------|
| 凭证存储加密 | P0 | 防止凭证泄露 | 中 |
| SSH 密钥管理增强 | P0 | 防止中间人攻击 | 中 |
| 容器逃逸防护 | P1 | 防止沙箱逃逸 | 中 |
| MCP 服务器沙箱 | P1 | 限制第三方工具权限 | 大 |
| Agent 操作审计 | P1 | 可追溯性 | 中 |
| API Key 轮换 | P2 | 可用性保障 | 小 |
