# build/darwin/Info.dev.plist 技术说明

## 文件定位
- 源文件: `build/darwin/Info.dev.plist`
- Wails macOS dev build 使用的应用 bundle plist 模板。

## 核心职责
- 与 production `Info.plist` 保持一致的基础 bundle 元数据，包括固定 bundle identifier `com.starxo.app`。
- 开发模式同样声明本地网络访问用途，确保 `wails dev` 下访问局域网 SSH sandbox 主机时能触发/通过 macOS Local Network 权限。
- 保留 `NSAppTransportSecurity.NSAllowsLocalNetworking`，便于本地网络地址访问。

## 维护要点
- production 与 dev plist 的本地网络权限声明应保持同步。
- 如果后续改 bundle identifier，需要同步 signing/release 校验，并重新验证 macOS Local Network 权限提示和 SSH 连接。
