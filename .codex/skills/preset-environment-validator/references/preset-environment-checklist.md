# Preset Environment Checklist

- Official presets validate without manual fixes.
- Manifest rejects invalid runtime permissions, invalid hooks, bad memory scopes, and malformed short descriptions.
- Install preview distinguishes file conflicts from environment conflicts.
- Install preview surfaces deterministic environment resolution with layer ordering plus effective runtime/budget information.
- Install path blocks environment conflicts even with overwrite/force-like flags.
- Successful installs emit `.arc/presets/reports/<install-id>.environment.json` and `.environment.md`.
- Domain presets cannot claim `system` memory or elevated runtime permissions.
- `task plan|run` emit `environment_resolution.{json,md}` when preset environment resolution is active.
- `task plan|run` emit `memory_policy.{json,md}` and ARC-managed memory writes use allowed runtime scopes.
- Hook execution is audited into `hook_execution.{json,md}` and risky hooks block without approval.
- `sandboxed_exec` hooks use the stricter bounded sandbox profile instead of inheriting the full parent shell environment.
- Missing declared hooks now have deterministic policy: required layers fail, overlay-scoped hooks can soft-skip with explicit audit status.
- Hook-mediated memory writes go through `arc hook memory add`, respect `ARC_ALLOWED_MEMORY_SCOPES`, and append `hook_memory_events.jsonl` in the run directory.
- User-facing preset metadata still shows stable name, tagline, and short description.
