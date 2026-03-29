# AGENTS.md

## Mission

Build the v1 MVP for `arc` (Agent Runtime CLI): a local-first, provider-agnostic CLI runtime for Claude and Codex with mode-driven behavior, durable project context, and reproducible task artifacts.

`plan.md` is the product source of truth. If implementation and this file diverge, update this file only after reconciling with `plan.md`.

## Operating Defaults

- Default delivery stance: autonomous MVP execution with approval on destructive, secret, network, publish, or external side-effect actions.
- Default engineering bias: simplest path to an end-to-end usable MVP, then refactor for maintainability.
- Core implementation target: Go, one binary, no GUI in v1.
- Architecture bias: provider-neutral orchestration core with provider-specific adapters.
- Evidence policy: do not state repo facts without code, docs, command output, or an explicit `INFERRED` label.

## Non-Negotiables

- Do not build a UI for v1.
- Do not make vector DB a required dependency.
- Do not mix human-authored maps/memory with machine-generated indexes.
- Do not fill unknown architecture details with guesses.
- Do not pull large repo context into prompts when cheap local search is enough.

## Task Loop

For every substantial task, follow this loop unless the user explicitly overrides it:

1. Read `plan.md` and the relevant `memory_bank/*` files.
2. Restate the current task in concrete engineering terms.
3. Identify impacted surfaces, unknowns, and likely acceptance criteria.
4. Implement the smallest end-to-end slice that moves the MVP forward.
5. Verify with the strongest local checks available.
6. Update docs affected by the change.
7. Update `memory_bank/decisions.md`, `memory_bank/unknowns.md`, and `memory_bank/run_journal.md` when the task changes project understanding.

## Repo Guidance

- `plan.md`: canonical product brief and MVP boundary.
- `memory_bank/`: persistent project memory. Keep it short, current, and evidence-backed.
- `memory_bank/active_context.md`: current execution focus and next concrete step.
- `agents/ROLES.md`: default delegation model for subagents.
- `skills/mvp-delivery/`: project-specific execution skill for MVP work.

## Delegation Model

Use subagents when parallel work materially shortens the path to a verified result.

- `explorer`: gather bounded codebase or spec facts.
- `worker`: own isolated implementation slices with disjoint file scope.
- `reviewer`: perform an independent pass for regressions, missing tests, and unsupported claims.

Keep critical-path design and integration decisions in the main agent. Delegate sidecar work, not ownership of the outcome.

## Evidence Labels

When uncertainty matters, use one of these labels in notes or reviews:

- `CODE_VERIFIED`
- `DOC_VERIFIED`
- `COMMAND_VERIFIED`
- `HUMAN_PROVIDED`
- `INFERRED`
- `UNKNOWN`

`INFERRED` is never a substitute for verification.

## Definition of Progress

Prefer progress that leaves reusable artifacts:

- code that runs,
- checks that prove behavior,
- docs that reduce future rediscovery,
- memory entries that survive session loss.

## Current MVP Direction

Immediate delivery priority:

1. Phase 1 foundation: CLI skeleton, config, `init`, provider detection, mode switch, basic logging.
2. Phase 2 context primitives: indexes, memory engine, context pack assembly.
3. Phase 3+ only after a usable Phase 1 flow exists.
