# MacLocalNetworkFixCard.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/settings/MacLocalNetworkFixCard.vue`
- macOS Local Network 权限修复提示组件。

## 核心职责
- 在 SSH 连接错误符合 macOS 私网 `no route to host` 条件时显示用户可读修复步骤。
- 提供 `CheckMacLocalNetworkAccess` 重新检测按钮和 copy-only `tccutil reset LocalNetwork com.starxo.app`。

## 维护要点
- 不能自动执行 reset 命令，也不能自动打开或修改系统隐私设置。
- `compact` 模式用于侧栏连接错误区域；完整模式用于 SSH 设置页。
