# custom.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/tools/custom.go
- 文档文件: doc/src/internal/tools/custom.go.plan.md
- 文件类型: Go 源码
- 所属模块: tools

## 2. 核心职责
- 提供泛型辅助函数 `NewCustomTool`，简化自定义工具的创建流程。它包装了 Eino 框架的 `utils.InferTool`，利用 Go 泛型自动从输入/输出结构体的 `json` 和 `jsonschema` 标签推导 JSON Schema，使开发者无需手动定义工具参数描述即可快速注册新工具。文件中包含详细的使用示例注释。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 工具名称 `name string`、工具描述 `description string`、工具执行函数 `fn func(context.Context, I) (O, error)`（I/O 为泛型类型参数）
- 输出结果: 返回 `(tool.BaseTool, error)`；成功时返回可注册到 ToolRegistry 的工具实例

## 4. 关键实现细节
- 结构体/接口定义: 无自定义结构体
- 导出函数/方法:
  - `NewCustomTool[I any, O any](name, description, fn) (tool.BaseTool, error)` — 泛型工具工厂函数
    - 类型参数 `I`: 工具输入类型（需带 `json`/`jsonschema` 标签的结构体）
    - 类型参数 `O`: 工具输出类型
    - 直接委托给 `toolutils.InferTool`
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `github.com/cloudwego/eino/components/tool` — `BaseTool` 接口
  - `github.com/cloudwego/eino/components/tool/utils` — `InferTool`（别名为 `toolutils`）
  - `context`（标准库）
- 关键配置: 无

## 6. 变更影响面
- 所有使用 `NewCustomTool` 创建自定义工具的调用方
- `internal/tools/registry.go` — 创建的工具通过 `RegisterCustom` 注册
- `app.go` 或插件系统 — 可能在应用初始化或运行时动态创建自定义工具

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 该函数是对 `InferTool` 的薄封装，主要价值在于提供简洁 API 和使用文档；如 `InferTool` 签名变更需同步更新。
- 输入结构体的 `jsonschema` 标签直接影响 LLM 看到的工具参数描述，标签质量决定工具被正确调用的概率。
- 文件中的注释示例是开发者创建自定义工具的主要参考，保持示例的准确性和完整性。
