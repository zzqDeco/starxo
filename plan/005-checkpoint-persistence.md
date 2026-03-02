# 005 - 检查点持久化

## 目标

将内存中的 CheckPointStore 替换为文件持久化，防止重启丢失中断状态。

## 范围

- `internal/store/checkpoint.go`

## 方案

1. 实现 `FileCheckPointStore`，每个 checkpoint 存为 `~/.starxo/checkpoints/{key}.json`
2. 保持与 `compose.CheckPointStore` 接口兼容（Set/Get）
3. 启动时自动加载已有 checkpoint
4. 会话删除时级联清理对应 checkpoint

## 具体任务

- [ ] 创建 `internal/store/file_checkpoint.go`: 实现 FileCheckPointStore 结构体，满足 compose.CheckPointStore 接口
- [ ] 实现 Set 方法: 将 checkpoint 序列化为 JSON 写入 `~/.starxo/checkpoints/{key}.json`
- [ ] 实现 Get 方法: 从文件反序列化读取 checkpoint
- [ ] 修改 `app.go`: 使用 FileCheckPointStore 替换 InMemoryStore
- [ ] 在 `SessionService.DeleteSession()` 中添加 checkpoint 文件清理逻辑
- [ ] 添加单元测试: 读写往返、文件不存在处理、并发访问

## 涉及文件

- `internal/store/file_checkpoint.go`（新建）
- `internal/store/file_checkpoint_test.go`（新建）
- `app.go`（修改 checkpoint store 初始化）
- `internal/service/session.go`（修改 DeleteSession）

## 预估时间

1 天

## 状态

待实施
