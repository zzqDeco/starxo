# Dev UI Layout Regression Fix

## Summary
- 修复 dev 上的主界面布局退化：左侧大面积空白、右侧 workspace/runtime 面板互相挤压、任务轨道遮挡输入、workspace 文件树无法跟随 terminal/agent 写入刷新。
- 保留现有组件体系，不重做完整视觉系统；本次只收敛布局状态和跨组件同步。

## Changes
- `MainLayout` 改为桌面端单一 inspector 模式：`runtime` 与 `workspace` 互斥显示。
- Workspace 打开时 sidebar 自动进入 compact rail，释放聊天和 inspector 空间。
- 窄屏继续使用 workspace sheet / runtime dock overlay，且两者互斥。
- `TaskRailFloating` 改为输入区上方的 native inline status bar，展开列表使用 popover。
- 新增 `workspace:changed` Wails 事件，terminal、agent runtime 写入、worktree 切换和上传完成后通知前端刷新 workspace。
- `WorkspacePanel` 对 `workspace:changed` 做 session/container 过滤和 debounce 刷新；当前预览文件变更后自动重载。
- Workspace panel 增加 container query，窄 inspector 下文件树和预览自动堆叠，避免控件 offscreen。
- macOS production/dev plist 补充 Local Network 用途声明和 local networking ATS 例外，避免打包 app 访问局域网 SSH 主机时报 `no route to host`。
- 修复 scoped CSS 中 macOS selector 写法：统一使用 `:global(:root[data-platform="macos"] .selector)`，避免把组件样式误编译到 `:root` 导致白屏/根节点污染。
- 收敛 macOS native 视觉：侧栏 source list、toolbar search field、右侧 inspector、composer segmented control、runtime tasks sheet 和工具时间线均使用更接近 macOS 的低对比度面板、细分隔线和中性按钮。
- 修复 i18n 回退：`zh-CN`/`en-US` 归一化为 `zh`/`en`，缺失 key 先查语言包再 fallback，最后才 humanize，避免 `Title`、`Mode Label`、`Placeholder` 这类变量名式文案出现在 UI。
- Agent 和工具时间线文案本地化，避免在中文界面里露出 `Coding Agent` 等内部英文标签。
- 处理 GitHub review 指出的 workspace change 事件一致性：上传前固定 active sandbox id；不可解析工具结果不触发刷新；LSPEdit 多文件改动触发 broad refresh；前端 debounce 保留预览重载意图。

## Verification
- `cd frontend && npm run build`
- `go test ./...`
- `wails build -skipbindings -trimpath`
- Computer Use 检查 1224px 宽度：左侧无 240px 空白、右侧无双重重型面板、任务轨道不遮挡输入。
- 用 `192.168.31.59` 回归 sandbox 创建、terminal 写入文件、workspace 自动刷新、agent 写入文件、销毁后清空状态。
