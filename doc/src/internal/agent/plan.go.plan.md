# plan.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/plan.go
- 文档文件: doc/src/internal/agent/plan.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件定义了计划（Plan）相关的数据模型和 Markdown 格式化工具。它提供 `Step`、`Plan`、`FullPlan` 等结构体用于表示执行计划的步骤及其状态（todo/doing/done/failed/skipped），以及将计划格式化为 Markdown 文档并写入沙箱容器的功能。这些数据结构被 plan_wrapper.go 用于持久化计划状态。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - `FormatPlanMD`: `[]*FullPlan` 切片
  - `WritePlanMD`: `context.Context`、`commandline.Operator`、工作区路径、`[]*FullPlan` 切片
- 输出结果:
  - `FormatPlanMD`: Markdown 格式字符串
  - `WritePlanMD`: 将 plan.md 文件写入沙箱容器工作区根目录
  - `PlanString`: 单行 Markdown 复选框格式字符串

## 4. 关键实现细节
- 结构体/接口定义:
  - `Step`: 计划步骤，包含 `Index`（序号）和 `Desc`（描述）
  - `Plan`: 计划结构体，包含 `Steps []Step`（步骤列表）
  - `PlanStatus`: 步骤执行状态枚举类型（string），包含 `PlanStatusTodo`、`PlanStatusDoing`、`PlanStatusDone`、`PlanStatusFailed`、`PlanStatusSkipped`
  - `FullPlan`: 完整计划步骤，包含 `TaskID`、`Status`、`AgentName`、`Desc`、`ExecResult`
- 导出函数/方法:
  - `(*FullPlan) PlanString(n int) string`: 将单个计划步骤格式化为 Markdown 复选框行，不同状态使用不同标记（[x]/[~]/[!]/[-]/[ ]）
  - `FormatPlanMD(plans []*FullPlan) string`: 将完整计划列表格式化为 Markdown 文档
  - `WritePlanMD(ctx, op, workspacePath, plans) error`: 将计划写入沙箱容器的 `{workspacePath}/plan.md` 文件
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `github.com/cloudwego/eino-ext/components/tool/commandline`: Operator 用于文件写入
- 关键配置: 计划文件固定写入 `{workspacePath}/plan.md`

## 6. 变更影响面
- `FullPlan` 和 `PlanStatus` 被 `plan_wrapper.go` 和 `internal/service/events.go`（PlanStepDTO）引用
- Markdown 格式变更影响前端解析和沙箱中 plan.md 文件的可读性
- `WritePlanMD` 的路径逻辑变更影响计划文件在容器中的位置
- `Step` 和 `Plan` 用于 JSON 反序列化 planexecute 的计划数据

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `PlanString` 中的 Markdown 标记（[x]/[~]/[!]/[-]）是自定义扩展，前端渲染需与此保持一致。
- `Step` 结构体的 JSON 标签需与 Eino planexecute 框架的计划输出格式兼容。
- 如需支持更丰富的计划状态（如 "blocked"、"cancelled"），需在此文件中扩展 `PlanStatus` 常量。
