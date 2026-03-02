# 002 - 凭证加密存储

## 目标

替换 config.json 中的明文敏感字段，使用操作系统凭证管理器安全存储。

## 范围

- LLM API Key（OpenAI、Anthropic 等）
- SSH 密码 / 私钥密码

## 方案

1. 引入 `github.com/zalando/go-keyring` 作为跨平台系统凭证管理库
2. 在 `internal/config/store.go` 的 Load/Save 流程中拦截敏感字段
3. config.json 中存储引用格式 `keyring://starxo/<field-name>`，实际值存入系统 keyring
4. 首次启动自动迁移：检测到明文值时写入 keyring 并替换为引用

## 具体任务

- [ ] 添加 `github.com/zalando/go-keyring` 依赖到 go.mod
- [ ] 创建 `internal/config/keyring.go`: 实现 SaveSecret/LoadSecret/MigrateSecrets 函数
- [ ] 修改 `internal/config/store.go`: Load 时检测 `keyring://` 前缀并解引用、Save 时将敏感字段写入 keyring
- [ ] 定义敏感字段列表（ApiKey、SshPassword 等），集中管理
- [ ] 测试: 明文迁移流程、keyring 读写、`keyring://` 引用格式解析、keyring 不可用时的降级处理

## 涉及文件

- `go.mod`（添加 go-keyring 依赖）
- `internal/config/keyring.go`（新建）
- `internal/config/store.go`（修改 Load/Save）
- `internal/config/keyring_test.go`（新建）

## 预估时间

1-2 天

## 状态

待实施
