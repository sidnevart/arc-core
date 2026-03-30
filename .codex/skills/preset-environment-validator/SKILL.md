---
name: preset-environment-validator
description: Validate ARC preset environment composition, manifest v2 fields, install-time conflicts, and runtime-facing environment artifacts before accepting preset work.
---

# Preset Environment Validator

Use this skill when preset manifests, install-time composition checks, hook rules, runtime permissions, or memory-scope enforcement change.

## Validate

1. Confirm manifest parsing and validation still cover:
   - `preset_type`
   - `compatible_providers`
   - `permissions.runtime`
   - `hooks`
   - `commands`
   - `memory_scopes`
   - `budget_profile`
   - `short_description`
2. Confirm install preview/install reject deterministic environment conflicts.
3. Confirm non-infrastructure presets cannot claim infrastructure-only memory/runtime access.
4. Confirm official preset catalog still validates end-to-end.
5. Confirm user-facing metadata for presets remains creator-authored and readable.

## Required checks

- `go test ./internal/presets ./internal/cli`
- `go run ./cmd/arc preset validate --root presets/official <preset-id>`
- if install logic changed: `go run ./cmd/arc preset preview ...`

## References

- `references/preset-environment-checklist.md`
