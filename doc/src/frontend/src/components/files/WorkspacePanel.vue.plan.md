# WorkspacePanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/files/WorkspacePanel.vue
- 文档文件: doc/src/frontend/src/components/files/WorkspacePanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/files

## 2. 核心职责
- 工作区主面板，提供文件树浏览、搜索、上传/下载、预览联动。

## 3. 输入与输出
- 输入来源: FileService (`ListWorkspaceFiles`, `ReadFilePreview`, `DownloadFile`)、`useWorkspaceBridge` 路径打开事件
- 输出结果: 渲染文件树与代码预览，触发上传下载行为

## 4. 关键实现细节
- 文件树:
  - 将 `FileInfo[]` 构造成目录/文件混合树节点
  - 目录优先排序，名称字典序排序
- 交互:
  - 顶部按钮：上传、下载、刷新
  - 搜索过滤：按 `path/name` 匹配
  - 选择文件后加载预览内容
  - 收到工具时间线发来的 workspace path 后，自动选择路径并加载预览
- 分栏:
  - 左侧树 + 右侧 `CodePreview`
  - 中间 `SplitHandle` 拖拽宽度（`starxo-workspace-tree-width`）
- 上传:
  - 通过 `FileTransfer` 上传弹窗
- 工作区桥接:
  - mounted 后消费 pending path，避免抽屉首次打开时丢失点击来源
  - mounted 期间监听 `starxo:workspace-open-path`

## 5. 依赖关系
- 内部依赖:
  - `SplitHandle.vue`, `FileTransfer.vue`, `CodePreview.vue`
  - `@/types/config` (`FileInfo`)
  - `@/composables/useWorkspaceBridge`
- 外部依赖:
  - `vue`, `naive-ui`, `@vicons/ionicons5`, `vue-i18n`
  - Wails `FileService`

## 6. 变更影响面
- 替代旧 FileExplorer 组合，提升工作区浏览与预览一体化体验。

## 7. 维护建议
- 若加入大目录懒加载，优先在 `buildTree` 层做虚拟化或按需展开。
- 下载/预览失败建议后续统一接入 message 提示。
