# runtime_tools.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/runtime_tools.go`
- 文档文件: `doc/src/internal/tools/runtime_tools.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 定义 Runtime V2 core tool 的 DTO、catalog entry 和执行逻辑。
- 把文件、搜索、编辑、shell、后台任务工具纳入统一 catalog/permission/ToolSearch 体系。

## 3. 输入与输出
- 输入来源: remote sandbox `commandline.Operator`、workspace path、runtime task manager
- 输出结果: catalog entries 和结构化 tool result

## 4. 关键实现细节
- always-load 工具：
  - `Bash`
  - `Read`
  - `Write`
  - `Edit`
  - `Glob`
  - `Grep`
  - `TaskOutput`
  - `TaskStop`
  - `ExitPlanMode`
- legacy aliases：
  - `shell_execute`
  - `read_file`
  - `write_file`
  - `list_files`
  - `str_replace_editor`
- `Read`/`Glob`/`Grep`/`TaskOutput`/`ExitPlanMode` 标记为 read-only trusted，plan mode 可见。
- `Bash`/`Write`/`Edit`/`TaskStop` 是 writable/destructive surface，plan mode 下不加载。
- 所有 workspace path 都走 guard：拒绝空 workspace、`..` traversal 和 workspace 外 absolute path。
- `Bash` 支持 foreground/background。background 通过 task manager 持久化输出。
- `Read` 支持 line offset/limit。
- `Edit` 使用精确字符串替换并返回 patch 摘要、行数变化和是否替换成功。
- `Glob` 通过远端 `find` 稳定排序；`Grep` 通过远端 `rg` 并支持 `content/count/files_with_matches`。

## 5. 依赖关系
- 内部依赖: `catalog.go`、`permissions.go`
- 外部依赖: `github.com/cloudwego/eino/components/tool`

## 6. 变更影响面
- 顶层 agent prompt 现在可以直接看到 Runtime V2 core tools。
- 后续 LSP/worktree/web/notebook tools 应通过相同 metadata contract 接入。

## 7. 维护建议
- 新增 runtime tool 时先定义 metadata、permission 和 read-only 语义，再接入实现。
- 任何文件类工具都必须复用 workspace guard，不要在工具内部各写一套 path 清洗。
