# Agent Runtime Claude Code-style Loop

## Summary
- Default top-level runtime now uses an Eino v0.9 `ChatModelAgent` ReAct loop plus Starxo behavior middleware instead of the fixed `deep.New` transfer topology.
- Plan mode remains a permission/tool-surface mode on the same runtime loop. It no longer builds an Eino PlanExecute graph by default.
- Each user turn receives a `RunObjective` sidecar. Standalone turns isolate older conversation history; continuation turns can inherit the recent active task.
- The legacy deep-transfer path remains available only behind `agent.runtime.enableBuiltinDeepTransferFallback`.

## Behavior
- `BuildRuntimeAgent` assembles direct runtime tools, ToolSearch middleware, context middleware, and `RuntimeBehaviorMiddleware`.
- Top-level session execution is now driven by Eino `TurnLoop`; `ChatModelAgent` remains the inner ReAct agent for each turn.
- TurnLoop owns user turn queueing, safe-point preemption, explicit stop, interrupt checkpointing, and resume dispatch.
- `<current-objective>` is pinned before model calls so older messages are treated as historical context unless the current turn is a continuation.
- `ask_user` and `ask_choice` reject prompts that clearly pursue stale debug/release/review work unrelated to the active objective.
- `Agent` supports two delegation styles:
  - omit `subagent_type` to fork the current agent context for isolated exploration
  - set `subagent_type` to create a fresh worker constrained by registry policy
- Runtime compact persists the active objective along with ToolSearch, permissions, tasks, todos, worktree state, file read state, and diff summaries.

## Refactor Boundary
- `ChatService.prepareRunnerBundleFromSurface` now delegates runner construction to `runtimeBundleBuilder`, keeping Wails/session lifecycle separate from runtime catalog and middleware assembly.
- Top-level agent selection is a standalone `buildTopLevelRuntimeAgents(...)` helper, not a `ChatService` method.
- `Agent` tool execution is handled by `runtimeSubagentRunner`, which receives clock/task/worktree dependencies explicitly instead of reaching through `ChatService`.
- Generation-bound tool provider code lives in `runtime_tool_provider.go`; Eino v0.9 ToolSearch bridge code lives in `runtime_toolsearch_eino.go`.
- Forked subagents use a dedicated `RuntimeForkAgentPrompt` plus direct ask/notify/todo tools; the prompt no longer advertises recursive Agent delegation.
- Subagent mode is monotonic: explicit plan can tighten default sessions, but explicit default cannot lower a parent plan session. The effective mode is propagated through provider and permission contexts.
- `runtime_turn_loop.go` isolates TurnLoop lifecycle from `chat.go`; subagent sync execution intentionally remains on a local runner.

## Validation
- Added targeted tests for standalone objective isolation, continuation inheritance, compact sidecar filtering, ask guard behavior, Agent fork normalization, and active objective cloning.
- Required checks:
  - `go test ./...`
  - `cd frontend && npm run build`
  - `/Users/zhaoziqian/go/bin/wails build -skipbindings -trimpath`
