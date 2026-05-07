# Runtime Context Compact

## Summary
- Branch: `fix/runtime-agent-review-findings`.
- Adds Runtime V2 context compaction state for long sessions.
- Replaces prompt preparation with token-aware windowing while preserving runtime state in a compact synthetic message.
- Persists compact state in `SessionData.RuntimeContextCompact` so reloads do not lose ToolSearch discovery, permission grants, task snapshots, file read ranges, recent edits, todos, plan state, or worktree routing.

## Implementation
- `internal/model/runtime_compact.go` defines the persisted compact schema.
- `internal/context/windowing.go` adds approximate token budgeting, compact prompt formatting, and token-aware windowing.
- `internal/service/runtime_context_compact.go` builds compact state from `SessionRun`, task manager, todo store, and workspace manager.
- Chat event processing records `Read` file ranges and `Write`/`Edit` diff summaries from runtime tool results.
- Runtime tasks can be restored from compact snapshots; previously running tasks become visible as failed after reload because the original process is no longer attached.
- Todo state can be snapshotted/restored per sessionID to avoid losing the plan/todo context on session reload without leaking state across concurrent sessions.

## Boundaries
- This does not call an LLM to summarize old turns. The compact summary is deterministic and token-aware.
- Full `SessionData.Messages` remain persisted for audit/recovery; compaction affects prompt construction, not on-disk history truncation.
- Read-state hashes are computed from the returned read range content, not from whole-file remote mtimes.

## Verification
- `go test ./internal/model ./internal/context ./internal/tools ./internal/service`
- Full `go test ./...`
- `cd frontend && npm run build`
