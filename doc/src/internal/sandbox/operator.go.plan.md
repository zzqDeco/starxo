# operator.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/sandbox/operator.go
- 文档文件: doc/src/internal/sandbox/operator.go.plan.md
- 文件类型: Go 源码
- 所属模块: sandbox

## 2. 核心职责
- `RemoteOperator` 实现了 Eino 框架的 `commandline.Operator` 接口，将所有文件和命令操作委托到远程 Docker 容器中执行。它是 AI Agent 工具层与沙箱环境之间的桥梁，提供文件读写（`ReadFile`/`WriteFile`）、目录判断（`IsDirectory`）、文件存在性检查（`Exists`）和命令执行（`RunCommand`）等能力。写文件操作根据内容大小自动选择策略：小文件（<=64KB）使用 base64 编码通过 `docker exec` 写入，大文件通过宿主机临时文件中转后 `docker cp` 到容器。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `RemoteDockerManager`（容器操作能力）、`context.Context`、文件路径、文件内容、命令参数 `[]string`
- 输出结果: `ReadFile` 返回文件内容字符串和 error；`WriteFile` 返回 error；`IsDirectory`/`Exists` 返回 bool 和 error；`RunCommand` 返回 `*commandline.CommandOutput`（含 Stdout/Stderr/ExitCode）和 error；可选的 `onOutput` 回调用于将命令输出转发到前端终端

## 4. 关键实现细节
- 结构体/接口定义:
  - `RemoteOperator` — 远程操作器，持有 `*RemoteDockerManager` 和可选的 `onOutput` 回调
  - 编译时接口检查: `var _ commandline.Operator = (*RemoteOperator)(nil)`
- 导出函数/方法:
  - `NewRemoteOperator(docker) *RemoteOperator` — 创建远程操作器
  - `SetOnOutput(fn)` — 设置命令输出回调（用于前端终端展示）
  - `ReadFile(ctx, path) (string, error)` — 读取容器内文件
  - `WriteFile(ctx, path, content) error` — 写入文件（自动选择小/大文件策略）
  - `IsDirectory(ctx, path) (bool, error)` — 判断路径是否为目录
  - `Exists(ctx, path) (bool, error)` — 判断路径是否存在
  - `RunCommand(ctx, command) (*commandline.CommandOutput, error)` — 执行命令并返回输出
- 私有方法:
  - `writeFileSmall(ctx, path, content) error` — 小文件 base64 编码写入
  - `writeFileLarge(ctx, path, content) error` — 大文件通过宿主机临时文件中转写入
- 工具函数:
  - `parentDir(path) string` — 返回路径的父目录
- Wails 绑定方法: 无
- 事件发射: 通过 `onOutput` 回调向前端发送终端输出

## 5. 依赖关系
- 内部依赖:
  - 同包引用: `RemoteDockerManager`（容器命令执行和文件复制）、`shellQuote`（命令参数转义）
- 外部依赖:
  - `github.com/cloudwego/eino-ext/components/tool/commandline` — 实现 `Operator` 接口和使用 `CommandOutput` 类型
  - `context`、`encoding/base64`、`fmt`、`strings`（标准库）
- 关键配置: 大文件阈值常量 `largeThreshold = 64 * 1024`（64KB）

## 6. 变更影响面
- `internal/tools/builtin.go` — 内置工具通过 `commandline.Operator` 接口调用 RemoteOperator
- `internal/sandbox/manager.go` — SandboxManager 创建并暴露 RemoteOperator
- `internal/agent/` — Agent 层通过 Eino 框架间接使用 Operator 执行文件操作和命令
- Eino 内置工具（`str_replace_editor`、`python_execute`）依赖此接口

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 大文件写入阈值（64KB）可根据实际网络情况调整。
- `writeFileLarge` 中临时文件路径使用容器路径的下划线替换，需注意路径冲突风险。
- `IsDirectory` 的实现有两种检测路径（链式命令和简单 `test -d`），修改时两条路径都需测试。
- 确保 `commandline.Operator` 接口变更时同步更新所有方法实现。
