# websearch_diagnostics_test.go 技术说明

## 文件定位
- 源文件: `internal/service/websearch_diagnostics_test.go`
- 所属模块: service tests

## 覆盖范围
- 公网 HTTP provider 静态诊断通过。
- TinyFish 缺失 API key env 时返回 fail。
- 默认 provider 被 disabled 时整体不可用。
- 非公网 endpoint 返回 warn，交给运行时 permission queue 决策。
- `TestWebSearch` 对失败路径返回 `ok=false` 结果而不是抛出前端异常。

## 维护建议
- 单测避免真实 DNS/外网依赖，使用公网 IP literal 或 localhost endpoint。
