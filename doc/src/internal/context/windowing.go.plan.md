# windowing.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/context/windowing.go`
- 文档文件: `doc/src/internal/context/windowing.go.plan.md`
- 文件类型: Go 源码
- 所属模块: `agentctx`

## 2. 核心职责
- 对消息做窗口化裁剪与内容截断。
- 现在支持显式 pinned prefix：始终保留 prefix，再裁剪普通 history。

## 3. 输入与输出
- 输入来源:
  - `WindowMessages(messages, cfg)`
  - `WindowMessagesWithPinnedPrefix(pinnedPrefix, history, cfg)`
- 输出结果: 裁剪后的 `[]*schema.Message`

## 4. 关键实现细节
- `WindowMessagesWithPinnedPrefix(...)` 是新的通用入口：
  - prefix 永不裁掉
  - history 按原有窗口规则裁剪
  - tool-call group 保留逻辑继续生效
- 不再依赖“保留前两条消息”之类的硬编码特殊 case。

## 5. 依赖关系
- 外部依赖:
  - `github.com/cloudwego/eino/schema`
  - `fmt`

## 6. 变更影响面
- deferred MCP announcement 可以稳定地位于 system prompt 之后、windowed history 之前。
- 为未来额外的 pinned meta hint 留出扩展空间。

## 7. 维护建议
- 若后续新增 pinned 内容，优先扩展 prefix 机制，不要再次引入按位置硬编码的保留规则。
