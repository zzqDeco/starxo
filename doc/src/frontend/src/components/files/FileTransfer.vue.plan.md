# FileTransfer.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/files/FileTransfer.vue
- 文档文件: doc/src/frontend/src/components/files/FileTransfer.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/files (文件管理模块)

## 2. 核心职责
- 文件传输对话框组件，提供文件上传到沙盒工作区的模态界面。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Props (show: boolean, mode: 'upload' | 'download')
- 输出结果: Emits (update:show)；调用 Wails 绑定执行文件上传

## 4. 关键实现细节
- **Props 定义**:
  - `show: boolean` — 对话框可见性
  - `mode: 'upload' | 'download'` — 传输模式
- **Emits 定义**: `update:show` — 控制对话框显隐
- **Wails 绑定调用**: `SelectAndUploadFile` (来自 FileService) — 打开原生文件选择器并上传
- **上传流程**:
  1. 用户点击上传区域触发 handleNativeUpload
  2. 调用 SelectAndUploadFile 打开文件选择器
  3. 上传成功后显示文件名，800ms 后自动关闭对话框
  4. 上传失败显示错误信息
- **模板结构**: NModal → 上传区域 (可点击的 drop-zone) → 图标 + 提示文本 + 上传结果

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `vue` (ref)、`naive-ui` (NModal, NCard, NButton, NIcon, NProgress)、`@vicons/ionicons5` (CloudUpload)、`vue-i18n` (useI18n)
- Wails 绑定: `wailsjs/go/service/FileService` (SelectAndUploadFile)

## 6. 变更影响面
- 被 FileExplorer 通过 v-model:show 控制
- 上传逻辑修改需同步 Go 后端 FileService

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 下载模式 (mode='download') 目前未实现 UI，仅有 upload 模板分支。
- 上传成功的 800ms 延迟关闭为硬编码，可考虑使用消息提示替代。
