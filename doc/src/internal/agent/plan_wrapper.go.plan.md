# plan_wrapper.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/plan_wrapper.go
- 文档文件: doc/src/internal/agent/plan_wrapper.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件实现了 `planMDWrapper`，一个 `adk.Agent` 接口的装饰器，用于在 planner/replanner 代理完成运行后自动持久化计划状态。它拦截代理事件流，在代理退出或正常完成时从 ADK session 中提取已执行步骤和剩余计划步骤，构建 `FullPlan` 列表，然后通过回调写入 Markdown 文件并发射计划事件通知前端。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - `NewPlanMDWrapper`: 被包装的 `adk.Agent`、写入函数 `func(plans []*FullPlan)`、计划事件回调 `PlanEventCallback`
  - `buildPlans`: 从 `adk.GetSessionValue` 读取 `planexecute.ExecutedStepsSessionKey` 和 `planexecute.PlanSessionKey`
- 输出结果:
  - 返回包装后的 `adk.Agent`，行为与原代理相同但增加了计划持久化副作用
  - 通过 `writeFunc` 回调写入 Markdown 文件
  - 通过 `onPlan` 回调发射计划事件

## 4. 关键实现细节
- 结构体/接口定义:
  - `PlanEventCallback`: 函数类型 `func(plans []*FullPlan)`，计划状态变更时的回调
  - `planMDWrapper`: 内部结构体，实现 `adk.Agent` 接口（Name/Description/Run），包含原代理、写入函数和事件回调
- 导出函数/方法:
  - `NewPlanMDWrapper(a adk.Agent, writeFunc, onPlan) adk.Agent`: 创建计划持久化包装器
- 内部方法:
  - `(*planMDWrapper) Run(ctx, input, options...) *adk.AsyncIterator[*adk.AgentEvent]`: 启动异步 goroutine 消费原代理事件流，在 Exit 动作或正常完成时调用 `persistPlan`
  - `(*planMDWrapper) persistPlan(ctx)`: 构建计划列表并调用 writeFunc 和 onPlan 回调
  - `(*planMDWrapper) buildPlans(ctx) []*FullPlan`: 从 session 中提取已执行步骤（标记为 Done）和剩余计划步骤（标记为 Todo），支持 JSON 格式的步骤描述解析
- Wails 绑定方法: 无
- 事件发射: 通过 `PlanEventCallback` 间接触发前端计划事件

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/logger`: 日志记录
  - 同包 `agent`: `FullPlan`、`PlanStatusDone`、`PlanStatusTodo`、`Step`、`Plan`
- 外部依赖:
  - `github.com/cloudwego/eino/adk`: Agent 接口、AgentInput、AgentEvent、AsyncIterator、GetSessionValue
  - `github.com/cloudwego/eino/adk/prebuilt/planexecute`: ExecutedStepsSessionKey、PlanSessionKey、ExecutedStep
- 关键配置: 无

## 6. 变更影响面
- 事件流拦截逻辑的变更影响计划持久化的时机和可靠性
- `buildPlans` 的解析逻辑变更影响计划状态的正确性
- 影响 `runner.go` 中 `BuildPlanRunner` 的计划持久化功能
- `PlanEventCallback` 的签名变更影响 `internal/service/chat.go` 中的事件处理

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `Run` 方法中使用了 goroutine 和 panic recovery，修改事件消费逻辑时需注意并发安全和资源泄漏。
- `buildPlans` 依赖 Eino planexecute 框架的 session key 格式，框架升级时需验证兼容性。
- 步骤描述的 JSON 解析是容错的（解析失败则使用原始字符串），新增字段时保持此兼容策略。
