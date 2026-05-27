# CC Runtime Systems V2

## Summary
- Branch: `feature/cc-runtime-systems-v2`.
- Scope: move Starxo closer to Claude Code's engineering runtime by adding a persistent task graph surface and permission decision audit state on top of the existing Eino v0.9 beta runtime.

## Eino Boundary
- Keep using Eino v0.9 beta `TurnLoop`, `ChatModelAgent`, `AgentTool`-style child-agent composition, checkpoint/resume, and interrupt semantics as the runtime foundation.
- Implement CC-style product systems that Eino does not own directly in Starxo's runtime layer: task graph persistence, permission audit records, compact/snapshot restore, and tool metadata.

## Changes
- Runtime task graph:
  - Added `TaskCreate`, `TaskGet`, `TaskUpdate`, and `TaskList` runtime tools.
  - Added session-scoped task graph records with `title`, `description`, `status`, `owner`, `priority`, `depends_on`, timestamps, and closed-state tracking.
  - Kept task graph tools safe for plan mode because they only mutate Starxo runtime metadata, not sandbox files or processes.
- Runtime compact:
  - Persist task graph records into `RuntimeContextCompact.TaskItems`.
  - Restore task graph records after session reload, replacing stale in-memory records for the restored session.
  - Strip task graph records from standalone objective compact state so unrelated new turns do not resume old workflows.
  - Include task graph counts in the compact prompt summary.
- Permission audit:
  - Record non-read-only runtime permission decisions for bypass mode, session grants, user decisions, missing UI context, and canceled requests.
  - Persist recent audit state in both `SessionData.PermissionAudit` and `RuntimeContextCompact.PermissionAudit`.
  - Emit `runtime:permission_audit_changed` for future settings/audit UI.
- Agent instructions:
  - Teach default and forked runtime agents to use `TaskCreate/Get/Update/List` for work that must survive compact/reload, delegation, workflow checkpoints, or user review.

## Verification
- `go test ./internal/tools ./internal/model ./internal/service ./internal/agent`

## Follow-Ups
- Build Workflow/Cron/Monitor on top of the task graph and permission audit layer.
- Add project/session-level permission rules before enabling autonomous automations.
- Consider a UI panel for task graph items and permission audit history after the backend contract stabilizes.
