# WorkspacePanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/files/WorkspacePanel.vue
- 文档文件: doc/src/frontend/src/components/files/WorkspacePanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/files

## 2. 核心职责
- 工作区主面板，提供文件树浏览、搜索、上传/下载、预览联动、sandbox 元信息、Runtime worktree 审阅和 tmp 清理。

## 3. 输入与输出
- 输入来源: FileService (`GetWorkspaceInfo`, `ListWorkspaceFiles`, `ReadFilePreview`, `DownloadFile`, `CleanupSandboxTmp`)、`useWorkspaceBridge` 路径打开事件、sandbox 生命周期 Wails 事件、`workspace:changed` 文件变化事件
- 输出结果: 渲染文件树与代码预览，触发上传下载行为

## 4. 关键实现细节
- 文件树:
  - 将 `FileInfo[]` 构造成目录/文件混合树节点
  - 目录优先排序，名称字典序排序
- 交互:
  - 顶部按钮：上传、下载、刷新、复制 workspace 路径、清理 tmp
  - 元信息栏以 inspector 密度展示 active sandbox、runtime、SSH host、workspace path；文件数量和大小仍由后端提供但不在窄右栏里抢占首屏视觉。
  - `WorktreeReviewPanel` 展示当前 session active worktree，并支持 review/merge/exit keep
  - 搜索过滤：按 `path/name` 匹配
  - 选择文件后加载预览内容
  - 收到工具时间线发来的 workspace path 后，自动选择路径并加载预览
  - 收到 `container:ready` / `container:activated` / `session:switched` 后自动刷新 workspace 信息与文件树
  - 收到 `container:deactivated` / `ssh:disconnected` 后清空文件树、选中路径、预览和搜索；`container:destroyed` 只在销毁当前 tracked registry container ID 时清空
  - 收到 `workspace:changed` 后按 session/container 过滤并 debounce 刷新；如果当前预览文件被更新则自动重载预览
- 分栏:
  - 左侧树 + 右侧 `CodePreview`
  - 中间 `SplitHandle` 拖拽宽度（`starxo-workspace-tree-width`）
  - 在窄 inspector 下通过 container query 自动改为树和预览上下布局，避免按钮和预览 offscreen
- 上传:
  - 通过 `FileTransfer` 上传弹窗
- 工作区桥接:
  - mounted 后消费 pending path，避免抽屉首次打开时丢失点击来源
  - mounted 期间监听 `starxo:workspace-open-path`
- worktree review 成功 merge 或 exit 后触发 `refreshFiles`，确保文件树跟随后端 active workspace 状态。
- 刷新和预览请求使用请求序号校验，避免 sandbox 销毁后旧响应写回 stale 文件内容。
- `currentWorkspaceContainerID` 只来自 lifecycle event payload 或 `GetWorkspaceInfo.activeContainerID`，避免从全局 store 读取到异步切换过程中的旧 active container。

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
- 工作区抽屉现在能展示 Runtime V2 worktree 状态，不再只显示原 sandbox workspace。
- macOS native pass 后，WorkspacePanel 更接近 Xcode/VS Code inspector：顶部工具栏低高度、元信息分组行、文件树选中态使用系统灰底，不使用强调色边条。

## 7. 维护建议
- 若加入大目录懒加载，优先在 `buildTree` 层做虚拟化或按需展开。
- 下载/预览失败建议后续统一接入 message 提示。
- tmp 清理只调用后端受保护方法，前端不得传入任意删除路径。
- lifecycle 清理必须同步清空预览内容，不能只清文件树。
