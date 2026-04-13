# checkpoint_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/store/checkpoint_test.go`
- 文档文件: `doc/src/internal/store/checkpoint_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: store

## 2. 核心职责
- 验证内存型 `CheckPointStore` 的基本读写和并发安全语义。

## 3. 输入与输出
- 输入来源: `NewInMemoryStore()` 生成的 store、不同 key/value 组合
- 输出结果: `Set` / `Get` 的返回值与存在标志

## 4. 关键测试覆盖
- store 创建成功
- 基础 set/get、覆盖写、缺失 key 查询
- 空字节切片和 `nil` 值都能稳定存取
- 多 key 并发读写不会破坏正确性

## 5. 依赖关系
- 内部依赖: `checkpoint.go`
- 外部依赖: `context`、`sync`、`stretchr/testify`

## 6. 变更影响面
- 保护 interrupt / resume 所依赖的内存检查点语义。

## 7. 维护建议
- 若替换为持久化 checkpoint store，应保留这类基础正确性与并发测试。
