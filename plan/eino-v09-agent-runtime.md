# Eino v0.9 Agent Runtime Migration

## Summary

Starxo's default agent runtime is migrated to `github.com/cloudwego/eino v0.9.0-beta.1` while preserving the existing Wails, SSH sandbox, MCP, permission queue, task manager, and session persistence contracts. The default provider path remains `*schema.Message`; Eino agentic models are available only as an explicit beta probe and must fall back cleanly.

## Runtime Shape

- Always-loaded tools: `Read`, `Write`, `Edit`, `Bash`, `Glob`, `Grep`, `TodoWrite`, `Agent`, `TaskOutput`, `TaskStop`, `ExitPlanMode`, and Eino v0.9 `tool_search`.
- Deferred tools: worktree, LSP, Skill, Notebook, WebFetch/WebSearch, MCP resource/action tools, and future large tool classes.
- `Agent` is the only default subagent delegation entry. Fixed deep-transfer subagents are kept behind `agent.runtime.enableBuiltinDeepTransferFallback` for debugging.
- `agent.runtime.subagents` defines built-in and user-configured subagent definitions: name, description, instruction, allowed tools, default isolation, and background policy.
- Custom subagent registries do not receive an implicit unrestricted `general`; empty `subagent_type` uses the registry default.

## Eino v0.9 Middleware

- `dynamictool/toolsearch` owns model-visible deferred tool discovery.
- Starxo remains the authority for catalog metadata, plan-mode filtering, permission, discovered-tool persistence, and workspace guard.
- Starxo filters Eino ToolSearch candidates through the same runtime availability rules used by the permission surface, and persists both JSON and model-native structured search results.
- `summarization` and `reduction` middleware handle token-aware compact and large tool result storage.
- `skill` loads workspace-local `.starxo/skills/<name>/SKILL.md` and `.claude/skills/<name>/SKILL.md`.
- `agentsmd` loads `AGENTS.md` and `.starxo/AGENTS.md` as transient runtime instructions.

## Agentic Beta Path

- `agent.runtime.agenticProtocol=off` is the default.
- `agentic_openai`, `agentic_ark`, and `auto` only probe Eino agentic model creation.
- Failure never blocks normal operation; Starxo logs the error and continues on the Message runtime.

## Migration Boundaries

- No security boundary moves into Eino. Permission prompts, private web access approval, task lifecycle, sandbox execution, and workspace path guarding remain Starxo-owned.
- Existing ChatService/Wails APIs remain compatible.
- The old transfer subagent files stay in the tree for fallback/debug compatibility but are no longer the main runtime path.

## Verification

- `go test ./...`
- `cd frontend && npm run build`
- `wails build -skipbindings -trimpath`
- Manual regression with local LLM and remote sandbox should cover file read/write/edit, Bash, ToolSearch discovery, dynamic Agent subagent sync/background, worktree isolation, Skill/AGENTS.md injection, compact recovery, and permission queue behavior.
