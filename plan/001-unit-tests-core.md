# 001 - Go 核心包单元测试

## 目标

为纯逻辑包添加单元测试，建立测试基础，确保核心模块的正确性和可回归性。

## 范围

- `internal/config` — Config 默认值、序列化
- `internal/model` — 数据模型构造
- `internal/store/checkpoint` — Set/Get/Delete
- `internal/context/windowing` — 消息截断、窗口化逻辑

## 方案

1. 添加 `github.com/stretchr/testify` 到 go.mod 作为测试断言库
2. 为每个目标包创建 `_test.go` 文件
3. 使用表驱动测试覆盖正常路径、边界条件和错误场景

## 具体任务

- [ ] 添加 `github.com/stretchr/testify` 依赖到 go.mod
- [ ] `internal/config/config_test.go`: 测试 DefaultConfig() 返回值、JSON 序列化/反序列化往返一致性
- [ ] `internal/model/session_test.go`: 测试 Session 构造、字段验证、零值处理
- [ ] `internal/model/container_test.go`: 测试 Container 状态转换（Created → Running → Stopped）
- [ ] `internal/store/checkpoint_test.go`: 测试 Set/Get/Delete 基本操作、并发安全（goroutine 竞争）
- [ ] `internal/context/windowing_test.go`: 测试 WindowMessages 截断逻辑、空输入、超长消息、边界条件

## 涉及文件

- `go.mod`（添加 testify 依赖）
- `internal/config/config_test.go`（新建）
- `internal/model/session_test.go`（新建）
- `internal/model/container_test.go`（新建）
- `internal/store/checkpoint_test.go`（新建）
- `internal/context/windowing_test.go`（新建）

## 预估时间

1 天

## 状态

待实施
