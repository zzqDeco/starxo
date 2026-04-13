# config_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/config/config_test.go`
- 文档文件: `doc/src/internal/config/config_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: config

## 2. 核心职责
- 验证默认配置、JSON 序列化行为和配置对象的基础值语义。

## 3. 输入与输出
- 输入来源: `DefaultConfig()` 和手工构造的 `AppConfig`
- 输出结果: 对默认值、`json.Marshal` / `json.Unmarshal` 结果的断言

## 4. 关键测试覆盖
- 默认 SSH / Docker / LLM / Agent / MCP 配置值正确
- 配置经 JSON round-trip 后不会丢字段
- 空密码、空私钥、空 headers 会被 `omitempty` 正确省略
- `DefaultConfig()` 每次返回独立实例，避免共享引用污染

## 5. 依赖关系
- 内部依赖: `config.go`
- 外部依赖: `encoding/json`、`stretchr/testify`

## 6. 变更影响面
- 保护配置默认值与持久化格式，避免设置页和配置存储出现不兼容漂移。

## 7. 维护建议
- 若新增配置字段，应同步补默认值断言和 JSON 序列化断言。
