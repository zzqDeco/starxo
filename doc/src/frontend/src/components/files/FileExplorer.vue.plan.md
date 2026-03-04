# FileExplorer.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/files/FileExplorer.vue
- 文档文件: doc/src/frontend/src/components/files/FileExplorer.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/files (文件管理模块)

## 2. 核心职责
- 文件浏览器组件，以树形结构展示沙盒工作区的文件列表，支持文件选择、预览、上传和下载。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Go 后端 FileService 返回的文件列表（已移除 `files:updated` Wails 事件监听，该事件后端未实际发射）
- 输出结果: 渲染文件树 UI；调用 Wails 绑定执行文件操作

## 4. 关键实现细节
- **Wails 绑定调用**: ListWorkspaceFiles, DownloadFile, ReadFilePreview (来自 FileService)
- **Wails 事件**: 无（已移除 `files:updated` 幽灵监听器，待后端实现文件变更通知时再添加）
- **文件树构建** (`treeData` computed): 将 FileInfo[] 平铺列表转换为 NTree 的 TreeOption[] 树形结构，按路径分隔符分组到目录节点
- **文件预览**: 选中文件后调用 ReadFilePreview 加载预览内容，使用 `<pre>` 标签渲染
- **模板结构**:
  - 头部: 工作区标题 + 操作按钮 (上传/下载/刷新)
  - 内容: NTree 树形列表或 NEmpty 空状态
  - 文件信息栏: 选中文件名 + 文件大小
  - 文件预览: 预览代码或无法预览提示
  - FileTransfer 对话框: 上传功能

## 5. 依赖关系
- 内部依赖: `@/types/config` (FileInfo)、`./FileTransfer.vue`
- 外部依赖: `vue` (ref, onMounted, computed)、`naive-ui` (NTree, NButton, NIcon, NEmpty, NTag, NSpin, TreeOption)、`@vicons/ionicons5` (FolderOpen, Document, CloudUpload, CloudDownload, Refresh)、`vue-i18n` (useI18n)
- Wails 绑定: `wailsjs/go/service/FileService` (ListWorkspaceFiles, DownloadFile, ReadFilePreview)

## 6. 变更影响面
- 文件树结构修改影响文件浏览体验
- FileInfo 类型变更需同步 `@/types/config`
- 被 MainLayout 右侧面板 (Files Tab) 包含

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 文件树构建逻辑仅支持一级目录分组，深层嵌套目录需要改进算法。
- 预览功能使用纯文本渲染，后续可考虑针对特定文件类型添加语法高亮。
- 已移除 `useWailsEvent` 导入和 `files:updated` 监听；待后端实现文件变更通知事件时需重新添加。
