# container_test.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `internal/model/container_test.go`
- 文档文件: `doc/src/internal/model/container_test.go.plan.md`
- 文件类型: Go 测试文件
- 所属模块: model

## 2. 核心职责
- 验证 `Container` 模型和 `ContainerStatus` 常量的基础语义。

## 3. 输入与输出
- 输入来源: `Container` 结构体和状态常量
- 输出结果: JSON round-trip 与状态值断言

## 4. 关键测试覆盖
- 状态常量文本值稳定
- 容器零值结构符合预期
- `Container` JSON 序列化和反序列化保持一致
- 常见状态转换写入后不丢失语义

## 5. 依赖关系
- 内部依赖: `container.go`
- 外部依赖: `encoding/json`、`stretchr/testify`

## 6. 变更影响面
- 保护容器状态持久化和前后端共享模型的兼容性。

## 7. 维护建议
- 若新增状态或字段，需要同步补 JSON 与状态值断言。
