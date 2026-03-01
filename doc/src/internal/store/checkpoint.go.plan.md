# checkpoint.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/store/checkpoint.go
- 文档文件: doc/src/internal/store/checkpoint.go.plan.md
- 文件类型: Go 源码
- 所属模块: store

## 2. 核心职责
- 该文件实现了一个线程安全的内存键值存储 `inMemoryStore`，作为 Eino 框架 `compose.CheckPointStore` 接口的实现。该存储用于 ADK Runner 的中断/恢复（interrupt/resume）工作流，在 Agent 执行过程中保存检查点状态数据。数据仅存储在内存中，不做磁盘持久化。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `Set` 方法接收 key (string) 和 value ([]byte)
- 输出结果: `Get` 方法返回 value ([]byte)、存在标志 (bool) 和错误 (error)

## 4. 关键实现细节
- 结构体/接口定义:
  - `inMemoryStore` — 未导出结构体，实现 `compose.CheckPointStore` 接口，持有读写锁 `mu` 和内存 map `mem map[string][]byte`
- 导出函数/方法:
  - `NewInMemoryStore() compose.CheckPointStore` — 创建内存检查点存储实例
- 未导出方法:
  - `(s *inMemoryStore) Set(_ context.Context, key string, value []byte) error` — 存储键值对
  - `(s *inMemoryStore) Get(_ context.Context, key string) ([]byte, bool, error)` — 获取键值对
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `context` (Go 标准库，接口方法签名要求)
  - `sync` (读写锁)
  - `github.com/cloudwego/eino/compose` (CheckPointStore 接口定义)
- 关键配置: 无

## 6. 变更影响面
- 该存储被 ChatService 的 Eino ADK Runner 使用，用于支持 Agent 执行流程的中断和恢复
- 修改接口实现可能影响 Runner 的状态管理行为
- 当前为纯内存实现，应用重启后检查点数据丢失；若需持久化检查点，需替换此实现

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 当前实现不包含数据过期或大小限制机制，长时间运行的 Agent 可能累积大量检查点数据。
- `context.Context` 参数在当前实现中未使用（以 `_` 忽略），后续若需支持超时或取消可利用此参数。
- 若需持久化检查点以支持应用重启后恢复 Agent 执行，可将此实现替换为基于文件系统或数据库的版本。
