# runtime_tools_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/runtime_tools_test.go`
- 文档文件: `doc/src/internal/tools/runtime_tools_test.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 覆盖 Runtime V2 core tools 的 catalog、alias、plan-mode gate、编辑执行、路径守卫、大输出持久化和 worktree-aware path 解析。

## 3. 输入与输出
- 输入来源: fake runtime operator、fake task manager、Runtime V2 catalog entries
- 输出结果: catalog state、tool invocation result、path guard result

## 4. 关键测试覆盖
- legacy aliases 映射到 canonical runtime tool names。
- plan mode current loaded 只保留 read-only trusted runtime tools。
- `TaskCreate` / `TaskGet` / `TaskUpdate` / `TaskList` 作为 safe runtime metadata tools 在 plan mode 可用。
- `Edit` tool 能通过 invokable contract 修改文件并返回 patch。
- `Edit` tool 替换大段旧文本时仍会在 patch 中保留 replacement 内容。
- `Write` tool 覆盖已有文件时返回 created=false、增删行和 patch metadata。
- `Write` tool 覆盖大型已有文件时使用 bounded preview，不调用全量 `ReadFile`。
- `Write` tool 的 bounded patch 会为写入的新内容保留预算，大文件覆盖小文件时仍能看到新增内容预览。
- bounded patch helper 会在生成预览时按 limit 截断并标记，不依赖先拼完整 patch。
- bounded patch helper 对超长单行保留可容纳的内容前缀，不退化成只有截断标记。
- `safeSearchPath` 拒绝 workspace 外 absolute path 和 `..` traversal。
- `workspaceFilePath` 在 worktree mode 下把 relative path 和 `/workspace/...` 映射到 active worktree，并拒绝 parent workspace absolute path。
- `workspaceFilePath` 允许 active worktree 自身的 absolute path，避免 `/workspace/.starxo/worktrees/...` 被重复映射。
- 大型 Bash 输出会写入持久化结果并在 inline stdout 中截断。
- `Read` tool 会通过 fake workspace manager 使用当前 session active workspace。
- deferred runtime tools 的 metadata 覆盖 LSP/LSPEdit/Skill/NotebookEdit/WebFetch/WebSearch 的 defer/read-only 语义。
- `LSPEdit` 和 `NotebookEdit` 必须保持 writable deferred，避免绕过 runtime permission queue。
- `WorktreeDiff` 是 read-only deferred；`WorktreeMerge` 是 writable deferred，plan mode 不直接加载。

## 5. 维护建议
- 新增 Runtime V2 core tool 时同步补 alias/permission/path 或执行语义测试。
