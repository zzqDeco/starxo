# filecontext.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/context/filecontext.go
- 文档文件: doc/src/internal/context/filecontext.go.plan.md
- 文件类型: Go 源码
- 所属模块: agentctx

## 2. 核心职责
- 该文件实现了文件上下文管理器 `FileContext`，负责跟踪三类文件：用户上传的文件（`uploadedFiles`）、工作区文件（`workspaceFiles`）和 Agent 生成的文件（`generatedFiles`）。提供文件的注册、查询和格式化功能，将当前文件信息注入到系统提示词中，使 AI Agent 能感知工作区的文件状态。所有操作通过读写锁保证线程安全。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `FileInfo` 结构体（文件名、路径、大小、修改时间、预览内容、是否为输出文件）
- 输出结果: 各类文件列表的副本；`FormatForSystemMessage()` 返回适合注入系统提示词的文本描述

## 4. 关键实现细节
- 结构体/接口定义:
  - `FileInfo` — 文件信息结构体，包含 Name, Path, Size, Modified, Preview, IsOutput 字段（支持 JSON 序列化）
  - `FileContext` — 文件上下文管理器，持有读写锁和三个文件列表
- 导出函数/方法:
  - `NewFileContext() *FileContext` — 创建空的文件上下文
  - `(fc *FileContext) AddUploadedFile(info FileInfo)` — 记录用户上传文件
  - `(fc *FileContext) AddGeneratedFile(info FileInfo)` — 记录 Agent 生成文件（自动设置 IsOutput=true）
  - `(fc *FileContext) SetWorkspaceFiles(files []FileInfo)` — 替换工作区文件列表
  - `(fc *FileContext) GetUploadedFiles() []FileInfo` — 获取上传文件列表副本
  - `(fc *FileContext) GetWorkspaceFiles() []FileInfo` — 获取工作区文件列表副本
  - `(fc *FileContext) GetGeneratedFiles() []FileInfo` — 获取生成文件列表副本
  - `(fc *FileContext) GetAllFiles() []FileInfo` — 获取所有文件合并列表
  - `(fc *FileContext) FormatForSystemMessage() string` — 格式化文件信息为系统提示词文本
- 未导出函数:
  - `formatSize(bytes int64) string` — 将字节数格式化为可读字符串 (B/KB/MB)
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `fmt`, `strings` (格式化)
  - `sync` (读写锁)
  - `time` (FileInfo.Modified 字段类型)
- 关键配置: 无

## 6. 变更影响面
- `FormatForSystemMessage()` 输出格式变更会影响 AI Agent 对工作区文件的感知方式
- `FileInfo` 结构体字段变更会影响前端文件展示和序列化兼容性
- 该类被 `Engine.PrepareMessages()` 和 `Engine.SessionValues()` 使用
- `SetWorkspaceFiles` 的全量替换策略意味着调用方需提供完整列表

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 文件列表仅追加不删除（上传文件和生成文件），长时间会话可能累积大量文件记录，可考虑添加清理机制。
- `FormatForSystemMessage()` 的输出会占用系统提示词的 Token 预算，文件数量过多时应考虑截断策略。
- 当前文件三分类（uploaded/workspace/generated）的边界在代码中未做强制校验，依赖调用方正确分类。
