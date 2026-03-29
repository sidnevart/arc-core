# E2E Scenarios

This file is the operator playbook for testing `arc` against real repositories and real tasks.

## Prerequisites

1. Build or install `arc`.

```bash
make build
make install
```

2. Ensure the relevant local runtimes are available.

```bash
arc doctor
```

Expected on this machine:
- `codex` is installed and detectable.
- `claude` may be missing; that is reported explicitly.
- `ast-grep` may be missing; indexing still works, but doctor reports the gap.

## Scenario 1: Bootstrap A Project

Run inside any repository root:

```bash
arc init --path .
arc mode show --path .
arc index build --path .
```

Expected results:
- `~/.arc/` exists with global config, built-in mode docs, and built-in skills.
- `.arc/` exists in the repo with provider guidance files, maps, memory files, local skills, indexes, runs, evals, and cache.
- `.arc/project.yaml` contains `default_provider` and `enabled_providers`.
- `.arc/mode.yaml` contains both `mode` and `autonomy`.
- `.arc/index/` contains `files.json`, `symbols.json`, `dependencies.json`, `recent_changes.json`, `docs.json`, `tooling.json`, and `bundle.json`.

## Scenario 2: Hero Planning

Create a planning run without changing provider state:

```bash
arc task plan --path . --mode hero --provider codex "Implement a changelog command for this repo"
arc run status --path .
```

Expected results:
- A new run directory appears under `.arc/runs/<run-id>/`.
- Hero artifacts exist:
  - `ticket_spec.md`
  - `business_spec.md`
  - `tech_spec.md`
  - `unknowns.md`
  - `question_bundle.md`
  - `context_pack.md`
  - `context_pack.json`
  - `mode_policy.md`

## Scenario 3: Full Dry-Run Pipeline

Exercise the whole deterministic pipeline without invoking Codex:

```bash
arc task run --path . --mode hero --provider codex --dry-run --no-provider --run-checks "Self-verify the repository MVP"
arc task verify --path . --run-checks
arc task review --path .
arc docs generate --path . --apply
arc run status --path .
arc memory status --path .
arc questions show --path .
```

Expected results:
- The run ends with `status=done`.
- The run directory contains:
  - `implementation_log.md`
  - `verification_report.md`
  - `review_report.md`
  - `docs_delta.md`
- `.arc/maps/` also contains refreshed:
  - `REPO_MAP.md`
  - `DOCS_MAP.md`
  - `CLI_MAP.md`
  - `ARTIFACTS_MAP.md`
  - `RUNTIME_STATUS.md`
- `.arc/memory/entries.json` is populated.
- `.arc/memory/MEMORY_ACTIVE.md`, `DECISIONS.md`, and `OPEN_QUESTIONS.md` reflect structured memory.
- `.arc/evals/metrics.json` updates run counters by mode.

## Scenario 4: Real Codex-Backed Run

Use this only after `codex login` is already configured locally.

```bash
arc task run --path . --mode hero --provider codex --provider-timeout 30s "Add a new command to this repository and update docs"
```

Expected results:
- `provider_transcript.jsonl` is written into the run directory.
- `provider_transcript.stderr.log` is written when the provider emits stderr or crashes.
- `active_roles.md`, `active_skills.md`, and `anti_hallucination_report.md` are written into the run directory.
- If Codex emits a session id, `arc run resume` can reuse it.
- If the provider runtime itself fails, the run now finishes as `status=failed` with verification/review/docs artifacts still present.

To continue a provider-backed run:

```bash
arc run resume --path . "Continue from the last completed stage and summarize blockers"
```

If the task description itself is risky, pass explicit approval:

```bash
arc task run --path . --mode hero --provider codex --approve-risky "Deploy the service and apply the migration"
```

## Scenario 5: Work Mode

```bash
arc mode set work --path .
arc task plan --path . --mode work "Refactor provider adapter boundaries"
```

Expected artifacts:
- `task_map.md`
- `system_flow.md`
- `solution_options.md`
- `unknowns.md`
- `validation_checklist.md`

## Scenario 6: Study Mode

```bash
arc learn "Explain the hero pipeline and why it uses deterministic states"
arc learn quiz --path .
arc learn prove "why separating human-authored maps from machine indexes reduces hallucination"
```

Expected results:
- A study-mode run is created.
- `practice_tasks.md` exists in the latest study run.
- Study artifacts include:
  - `learning_goal.md`
  - `current_understanding.md`
  - `knowledge_gaps.md`
  - `challenge_log.md`
  - `practice_tasks.md`

## Scenario 7: Memory Maintenance

```bash
arc memory status --path .
arc memory compact --path .
```

Expected results:
- Status prints counts by kind and status.
- Compaction rewrites markdown summaries from structured memory.
- Entries older than the freshness window can move from `active` to `stale`.

## Scenario 8: Docusaurus Docs Site

```bash
make docs-install
make docs-build
make docs-dev
```

Expected results:
- Docusaurus build succeeds from `docs-site/`.
- Static site output is generated under `docs-site/build/`.
- Local docs server starts on `http://localhost:3000`.

## Known Constraints

- Built-in check execution is currently strongest for Go repositories because `task verify --run-checks` runs `go test ./...` when `go.mod` is present.
- Claude adapter logic follows Anthropic's official headless CLI shape (`claude -p`, `--output-format json`, `--resume`, `--continue`), but cannot execute unless the local `claude` binary is installed.
- `doctor` reports missing prerequisites rather than hiding them.
- `init` overwrites machine-managed config files such as `.arc/project.yaml` and `.arc/mode.yaml`, but does not overwrite human-authored map or guidance files.
- As validated on 2026-03-26 in this workspace, the local `codex` binary can panic inside its own OTEL/system-configuration path during live `exec` runs. `arc` now captures the partial transcript, stderr log, and marks the run `failed` rather than hanging indefinitely.
