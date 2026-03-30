# RFC: Preset Environment Composition

## Status

Accepted for implementation sequencing.

## Problem

ARC already has a working preset install flow, but the current manifest is too thin to safely support multi-agent composition. Today presets are mostly file bundles plus metadata. That is not enough to answer:

- which layer a preset belongs to,
- what runtime actions it may request,
- what memory namespaces it can touch,
- which hooks are legal,
- how conflicting presets should be rejected.

Without a composition model, infrastructure concerns and domain behavior will leak into each other.

## Current repo truth

- `internal/presets/manifest.go` already loads JSON-compatible YAML manifests.
- `internal/presets/install.go` already installs preset payloads into project-local `.arc`.
- `.arc/hooks/` and `.arc/presets/installed.json` already exist in project scaffold.
- Built-in and official presets already live in `presets/official/`.

## Goals

- make preset layering explicit,
- make environment permissions machine-readable,
- reject obviously unsafe or conflicting manifests early,
- keep the contract provider-neutral.

## Non-goals

- full runtime enforcement for every permission in the same slice,
- marketplace distribution,
- dynamic third-party plugin execution.

## Canonical layer order

1. ARC base rules
2. provider adapter rules
3. infrastructure presets
4. selected domain preset
5. project overlays
6. session overlays
7. explicit user instruction

## Preset classes

- `infrastructure`
  - context, memory, safety, runtime, metrics layers
- `domain`
  - learning, delivery, investigation, architecture, tutoring
- `session_overlay`
  - temporary task/session-specific adjustments

## Manifest contract v2

The manifest now supports environment-aware metadata:

- `preset_type`
- `compatible_providers`
- `required_modules`
- `permissions.runtime`
- `hooks`
- `commands`
- `memory_scopes`
- `runtime_policy`
- `quality_gates`
- `metrics_expectations`
- `budget_profile`

## Validation rules in the first slice

- reject unsupported `preset_type`
- reject unsupported `permissions.runtime`
- reject unsupported hook lifecycles
- require bounded hook timeouts
- reject duplicate commands, hooks, and memory scopes
- require memory scopes to use approved roots or explicit namespaces

## Runtime policy model

First-class runtime permission levels:

- `none`
- `read_only`
- `preview_only`
- `sandboxed_exec`
- `risky_exec_requires_approval`

These levels are declarative in the first slice and will become enforced runtime policy later.

## Memory namespacing

ARC should treat memory as layered storage, not one shared scratchpad.

Approved roots:

- `system`
- `project`
- `session`
- `run_artifacts`
- `archive`
- `presets/<id>`
- `runs/<id>`

## Risks

- manifests become richer before install/runtime fully enforce every field
- preset authors may assume declarative fields imply automatic behavior
- the split between built-in modes and installable presets can drift if not normalized

## Rollout

### Phase 1

- manifest v2 fields
- parser validation
- official presets updated to the new contract

### Phase 2

- install-time conflict detection across installed presets
- explicit layer resolution and diagnostics
- install-time `environment_resolution.json` and `environment_resolution.md` artifacts
- validation-time rejection of non-infrastructure manifests that claim `required_modules`, `system` scope, or elevated runtime execution permissions

### Phase 3

- runtime enforcement for permissions, hooks, and memory scopes

Current implementation update:

- runs now materialize `environment_resolution.{json,md}` alongside normal run artifacts
- runs now also materialize `memory_policy.{json,md}` so runtime-visible allowed scopes are explicit
- the first hook runner now executes approved lifecycle hooks from `.arc/hooks/` with timeout and permission/approval gates
- hook execution is auditable via `hook_execution.{json,md}`
- sandboxed hook runs now also emit `hook_sandbox_profile.{json,md}` so the bounded execution profile is inspectable after the fact
- `risky_exec_requires_approval` hooks now block runs until approval instead of being advisory metadata only
- ARC-managed memory persistence now uses canonical `runs/<run-id>` scopes and is checked against the resolved allowlist before write
- `sandboxed_exec` is now a stricter bounded sandbox profile, not just a label:
  - hooks run from a dedicated sandbox directory under the run artifacts
  - the environment is sanitized instead of inheriting the full parent shell
  - `ARC_HOOK_SANDBOX_DIR` is exposed explicitly
  - only `.sh` / `.bash` hook scripts are allowed in this mode
  - a separate sandbox profile artifact records the working directory, env-key allowlist, allowed memory scopes, and the mediated memory-write path for each sandboxed hook execution
- hook-side memory writes now have a mediated path:
  - hooks can call `arc hook memory add`
  - the command validates scope against `ARC_ALLOWED_MEMORY_SCOPES`
  - successful writes append `hook_memory_events.jsonl` under the run directory
- missing-hook policy is now explicit too:
  - infrastructure/domain-owned hooks fail as `missing_required`
  - project/session-overlay-owned hooks soft-skip as `missing_skipped`
- true OS-level isolation for `sandboxed_exec` still remains a later refinement

## Verification

- manifest parser tests
- preset validation on official catalog
- install-time checks once conflict detection lands
- orchestrator tests for hook execution, approval-gated hooks, and per-run environment artifacts
