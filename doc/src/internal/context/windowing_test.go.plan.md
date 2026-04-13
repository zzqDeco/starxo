# windowing_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/context/windowing_test.go`
- 文档文件: `doc/src/internal/context/windowing_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: context

## 2. 核心职责
- 验证消息窗口化、内容截断和 tool-call 分组保护逻辑。

## 3. 输入与输出
- 输入来源: 构造的 `schema.Message` 历史和 `WindowConfig`
- 输出结果: `WindowMessages(...)`、`WindowMessagesWithPinnedPrefix(...)`、`TruncateContent(...)`

## 4. 关键测试覆盖
- 空消息、默认配置和 pinned prefix 的基础行为
- 超出窗口预算时保留系统提示、插入省略占位、保留最近消息
- 长内容截断保留头尾并插入 marker
- 工具调用与工具结果不会被窗口切断，`adjustForToolCallGroups` 会把 cut point 回退到完整 group 起点

## 5. 依赖关系
- 内部依赖: `windowing.go`
- 外部依赖: `github.com/cloudwego/eino/schema`、`stretchr/testify`

## 6. 变更影响面
- 保护模型上下文窗口规则，直接影响 token 使用、上下文完整性和 tool-call 恢复能力。

## 7. 维护建议
- 若修改窗口裁剪策略，必须保住 tool-call group 不被拆断这一约束。
