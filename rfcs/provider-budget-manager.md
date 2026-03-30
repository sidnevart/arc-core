# RFC: Provider Budget Manager

## Status

Accepted; Phase 1 first slice implemented on 2026-03-29.

## Problem

ARC already has provider adapters, timeouts, and guarded execution, but it does not yet have a true budget-aware decision layer. Provider calls are still expensive operations that need classification, accounting, and graceful degradation.

## Current repo truth

- provider adapters already exist for Codex and Claude
- provider failures are already captured with deterministic artifacts
- normal chat uses stricter reply-only flows and timeouts
- there is no provider usage ledger or budget policy engine yet
- `internal/budget` now exists with a first policy, assessment, and usage-event model
- `arc task plan|run` now accept `--budget-mode`
- `task run` now records `budget_assessment.json` in the run dir and appends `.arc/budget/usage_events.jsonl`
- `task run` can now reroute local-first work into a local-only path before provider execution, and records that decision in both run metadata and usage events
- `task run` now also records `prompt_minimization.json`, embedding the same evidence into `budget_assessment.json` and `budget_usage_event.json`

## Goals

- make provider calls measurable,
- route cheap work away from premium providers when possible,
- warn or block before wasteful execution,
- support graceful low-limit behavior.

## Non-goals

- hard-coding public vendor pricing or plan limits,
- forcing all tasks through a remote accounting service,
- removing user override for intentionally expensive tasks.

## Core components

- Usage Tracker
- Request Classifier
- Budget Policy Engine
- Prompt Minimizer
- Local Work Router

## Request classification

Each provider-bound task should first be classified into one of:

- `no_provider`
- `local_first`
- `cheap_provider_ok`
- `premium_required`
- `premium_high_risk`

## Budget modes

- `ultra_safe`
- `balanced`
- `deep_work`
- `emergency_low_limit`

## Policy direction

- local work should happen before provider work whenever possible
- expensive or repeated calls should surface warnings or require approval
- retries should not loop blindly without changed inputs

## Preset relationship

Presets may declare a `budget_profile`, but they do not get to self-authorize expensive behavior. Budget enforcement remains infrastructure-owned.

## Risks

- soft estimates may be mistaken for exact provider billing
- budget UI without hard policies could create false trust
- low-limit mode could degrade into confusing failures if not explicit

## Rollout

### Phase 1

- RFC and event model
- usage event schema
- budget policy schema
- status: implemented as the first slice, including run-time request classification and persisted budget artifacts/ledger events.

### Phase 2

- local classification hooks before provider execution
- budget artifacts and session-level visibility
- status: started in minimal form; request classification now happens before provider execution in `task run`, local-first requests can already be rerouted away from the provider when policy prefers local work, but routing heuristics and session-level visibility are still shallow.

### Current local-first routing rule

- Build an assessment before provider execution
- If policy says `prefer_local=true`
- and classification is:
  - `no_provider`
  - or `local_first`
  - or `cheap_provider_ok` with low confidence under `emergency_low_limit`
- then ARC switches the run onto the local-only path
- Persist the decision in:
  - `budget_assessment.json`
  - `budget_usage_event.json`
  - run metadata:
    - `budget_route_locally`
    - `budget_routing_reason`
    - `provider_execution_mode`

### Current heuristic rule

- routing no longer uses first-match wins
- signals carry simple weights
- precedence is:
  - `premium_high_risk`
  - `premium_required`
  - `local_first`
  - `cheap_provider_ok`
- practical effect:
  - `inspect context tool budget schema` still classifies as `local_first`
  - `inspect and implement the budget schema` now classifies as `premium_required`, not `local_first`
  - weak generic prompts such as brainstorming labels can stay `cheap_provider_ok` and, under `emergency_low_limit`, reroute locally when confidence is low enough

### Current mode precedence rule

- an explicit `--budget-mode` is the effective mode for the run
- if no explicit mode is provided, ARC may inherit the effective mode from a session override file passed via `--budget-override-file`
- if no session override mode is present, ARC may inherit it from `.arc/budget/project_override.json`
- if neither override provides a mode, ARC may inherit the effective default from the resolved preset stack via `environment_budget_profile`
- if neither explicit mode nor preset-linked profile is present, ARC falls back to `balanced`
- after the effective mode is chosen, ARC materializes the default policy for that mode and then applies optional project/session policy overrides on top
- each run now persists `budget_policy_resolution.json` so the final policy is explainable as mode source plus applied override layers
- the override layers now also have first operator-facing commands:
  - project override: `arc budget show`, `arc budget override set|clear`
  - session override file: `arc budget session write|show|clear`
- this keeps:
  - `requested_budget_mode`
  - `environment_budget_profile`
  - `budget_mode_source`
  - `assessment.mode`
  - `policy.mode`
  - persisted `policy.json`
  - run metadata
  aligned for the same execution

### Phase 3

- approval gates, low-limit mode, and preset-linked budget policies
- status: first low-limit slice implemented. Assessments and usage events now persist `low_limit_state`, `confidence`, and `matched_signals`; `emergency_low_limit` blocks `premium_required` in addition to `premium_high_risk`; and low-confidence `cheap_provider_ok` requests can now reroute locally to conserve provider budget.
- the same slice now also persists `confidence_tier` and `signal_breakdown`, so follow-up routing and operator review can see not only the final class but also the relative local/premium/high-risk score shape that produced it.
- the current routing layer now also persists `routing_trigger`, and `ultra_safe` (`low_limit_state=constrained`) joins `emergency_low_limit` in rerouting weak low-confidence `cheap_provider_ok` work locally instead of spending provider budget on generic prompts.
- the next explainability slice on 2026-03-30 also made budget attribution prompt-aware:
  - each run now persists `prompt_minimization.json`
  - `budget_assessment.json` embeds the same `prompt_minimization` payload
  - `budget_usage_event.json` and the global usage ledger now persist:
    - `provider_model`
    - `provider_session_id`
    - `project_root`
    - `budget_mode_source`
    - `environment_budget_profile`
    - `context_source`
    - `context_selection_reason`
    - `context_arc_tokens`
    - `context_ctx_tokens`
    - `context_selected_tokens`
    - `context_token_reduction`
    - `context_token_reduction_percent`
    - `prompt_minimized`
    - `route_locally`
  - this closes the gap between “provider work was allowed” and “ARC actually minimized the prompt before allowing it”

## Verification

- request classification tests
- usage event generation tests
- degraded-mode and approval-gate tests
- live smoke now also confirms local-first routing on this repository: `arc task run --budget-mode balanced "inspect context tool budget schema"` completed with `budget_classification=local_first`, `budget_route_locally=true`, `provider_execution_mode=local_routed`, and `used_provider=false` in the usage event.
- a later smoke also confirmed the mode-precedence fix: on a repository that previously carried a stale `ultra_safe` policy file, rerunning the same command with `--budget-mode balanced` rewrote `.arc/budget/policy.json` back to `balanced` and kept `assessment.mode` and `policy.mode` aligned in `budget_assessment.json`.
- another smoke confirmed the heuristic refinement: `arc task run --budget-mode balanced "inspect and implement the budget schema"` now produces `classification=premium_required` with `route_locally=false`, proving weak local signals no longer dominate provider-bound work.
- the current low-limit smoke now also confirms `emergency_low_limit` behavior: `implement the budget schema` is blocked as `premium_required`, while a weak generic prompt such as `brainstorm three friendly names for the budget modes` stays `cheap_provider_ok`, records `low_limit_state=emergency`, and reroutes locally with `provider_execution_mode=local_routed`.
- a later smoke on 2026-03-30 confirmed the prompt-minimization slice too:
  - `.arc/runs/20260330T214448Z-476991000/prompt_minimization.json` recorded `context_source=ctx`, `token_reduction=3327`, and `token_reduction_percent=82` for `inspect context tool budget schema`
  - `.arc/runs/20260330T214448Z-476991000/budget_usage_event.json` mirrored the same context-token and minimization fields together with `project_root` and `route_locally=true`
  - `.arc/runs/20260330T214448Z-612250000/budget_assessment.json` embedded the same prompt-minimization structure for a `premium_required` case, proving the artifact is not limited to local-routed runs
