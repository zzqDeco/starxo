# scripts/sign-darwin-app.sh 技术说明

## 文件定位
- 源文件: `scripts/sign-darwin-app.sh`
- macOS app bundle 签名与校验脚本。

## 核心职责
- 对 `build/bin/starxo.app` 执行 ad-hoc 或指定 identity 的 codesign。
- 校验 bundle identifier 必须为 `com.starxo.app`。
- 校验签名绑定 `Info.plist` 并 seal resources，避免发布无稳定 macOS bundle identity 的包。

## 维护要点
- 默认使用 ad-hoc identity `-`；正式证书通过 `MACOS_CODESIGN_IDENTITY` 注入。
- release workflow zip macOS 产物前必须运行本脚本。
