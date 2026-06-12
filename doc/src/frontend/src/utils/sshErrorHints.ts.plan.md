# sshErrorHints.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `frontend/src/utils/sshErrorHints.ts`
- 文档文件: `doc/src/frontend/src/utils/sshErrorHints.ts.plan.md`
- 文件类型: TypeScript 工具函数
- 所属模块: frontend/src/utils

## 2. 核心职责
- 统一 SSH 错误文本提取。
- 针对 macOS app 访问局域网 SSH 主机时报 `no route to host` 的场景，提供 Local Network 权限风险判断。

## 3. 输入与输出
- 输入来源: Wails 调用抛出的 error/string，以及当前 SSH 配置中的 host。
- 输出结果: 原始错误文案，以及供 Sidebar/SSH 设置页决定是否显示 Local Network 修复卡片的布尔判断。

## 4. 关键实现细节
- 仅在 macOS 平台、错误包含 `no route to host`、host 属于 localhost/private/link-local/mDNS 等本地网络地址时返回 true。
- 不修改后端错误语义，不吞掉原始错误，避免掩盖真实网络或认证失败。
- 不自动执行 `tccutil` 或系统设置变更，权限恢复仍由用户在系统设置中完成。

## 5. 变更影响面
- 左侧连接按钮失败提示。
- 设置页 SSH 测试失败提示。
- 不影响 Windows/Linux 或公网 SSH 主机错误文案。

## 6. 维护建议
- 新增 SSH 连接入口时应复用 `formatSSHError`。
- 如果后续引入平台服务 API，可把 macOS 平台判断替换为 Wails runtime/platform 信息。
