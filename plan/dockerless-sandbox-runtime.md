# Dockerless Sandbox Runtime

## Summary

Starxo will keep the existing SSH-based remote execution model and replace Docker-backed execution with lightweight OS sandboxes. Linux hosts use `bubblewrap` (`bwrap`); macOS hosts use Seatbelt through `sandbox-exec`. Each session owns a persistent workspace under `~/.starxo/sandboxes/<sandbox-id>/workspace`.

## Runtime Model

- `runtime=auto` selects `bwrap` for Linux and `seatbelt` for Darwin.
- Runtime setup is explicit from Settings: users can check availability and run installer actions instead of Starxo silently installing Docker on connection.
- A sandbox is a persisted workspace plus runtime metadata, not a long-running container process.
- Activate/deactivate only changes the active sandbox for command execution.
- Destroy removes the registered sandbox and its remote workspace.

## Compatibility

- Existing Wails service names may remain during the migration, but user-facing copy changes from container/Docker to sandbox/runtime.
- Existing `config.docker` values are migrated to `config.sandbox` on load.
- Existing `containers.json` is migrated to `sandboxes.json`. Legacy Docker records are retained as `unavailable` and are not started, stopped, or removed through Docker.

## Limitations

- Docker fallback is intentionally not preserved.
- Docker container contents are not automatically exported.
- CPU and memory limits are best-effort process limits, not container-level hard quotas.
- Remote root SSH users are supported but not recommended because OS sandbox isolation is weaker under root.
