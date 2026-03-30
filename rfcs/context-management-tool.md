# RFC: Standalone Context Management Tool

## Status

Accepted; Phase 2 first slice implemented on 2026-03-29.

## Problem

ARC already assembles context through `internal/contextpack`, but that logic is still embedded as an internal helper. The product requirement is stronger: context management should be measurable, reusable, and eventually portable beyond ARC.

## Current repo truth

- `internal/contextpack` already builds a context bundle for orchestrator runs.
- orchestrator writes context artifacts such as `context_pack.md`.
- indexing and memory already exist locally in the ARC runtime.
- `cmd/ctx` now exists with first working commands: `init`, `doctor`, `index build|refresh`, `memory add|list|search|status|compact`, `assemble`, and `bench`.
- the first standalone workspace contract now lives under `.context/` with separate `index/`, `memory/`, and `artifacts/assemble/`.
- a human-authored `.context-tool.yaml` now sits beside the workspace and controls standalone include/exclude/docs/memory/language-hint behavior without mixing operator intent into machine-managed `.context/config.json`.
- `.context/memory/entries.json` now stores standalone context-tool memory entries, and the tool also syncs readable memory artifacts under `.context/memory/` including active, archive, and open-question views.
- `.context/benchmarks/` now stores baseline-vs-optimized comparison artifacts for context assembly.
- ARC orchestrator now builds both the classic ARC pack and a `ctx` pack during `task plan|run`, persists both, and records a deterministic selection artifact for the provider-facing context path.

## Goals

- define a standalone-first `ctx` tool boundary,
- make context assembly budgeted and artifact-backed,
- support baseline vs optimized comparisons,
- let ARC act as host/adapter instead of the only integration surface.

## Non-goals

- semantic/vector retrieval as a hard dependency,
- distributed indexing or remote services,
- full IDE integration.

## Proposed CLI surface

- `ctx init`
- `ctx doctor`
- `ctx index build`
- `ctx memory add`
- `ctx memory list`
- `ctx memory search`
- `ctx memory status`
- `ctx memory compact`
- `ctx assemble`
- `ctx bench`

## Storage model

The tool owns `.context/`:

- `index.db`
- `maps/`
- `memory/`
- `artifacts/`
- `runs/`
- `benchmarks/`

This keeps context optimization separate from `.arc/`, even if ARC remains the first host.

The standalone config layer lives outside `.context/`:

- `.context-tool.yaml` is human-authored and stable
- `.context/config.json` remains machine-managed workspace metadata
- `ctx doctor` reports both surfaces separately

## Architecture

- Context Ingestor
- Indexing Layer
- Retrieval Engine
- Context Assembler
- Memory Manager
- Metrics Collector
- Evaluation Runner

## ARC integration

ARC should gradually move from directly calling ad hoc context helpers to calling the `ctx` boundary for:

- targeted retrieval
- budgeted context assembly
- provenance reporting
- benchmark/eval comparisons

## Risks

- duplicated storage during transition from `internal/contextpack`
- premature CLI surface before enough engine reuse exists
- confusion between ARC memory and context-tool memory if boundaries are not documented

## Rollout

### Phase 1

- RFC and storage contract
- artifact model and CLI shape
- map current `internal/contextpack` responsibilities to future `ctx` modules

### Phase 2

- first `ctx` CLI commands backed by current engines
- status: in progress; current slices implement `cmd/ctx`, `internal/contexttool`, `.context/` scaffold, cached index artifacts, standalone `.context/memory` storage with add/list/search/status/compact commands, a first workspace `doctor`, human-authored `.context-tool.yaml`, assembly artifacts, and a first `bench` comparison flow for baseline vs optimized packs.

### Phase 3

- ARC orchestrator calls through the `ctx` boundary instead of directly owning assembly logic
- status: started beyond compatibility mode; current runs now persist both `arc_context_pack.*` and `ctx_context_pack.*`, write `context_selection.json`, and choose the provider-facing `context_pack` with a simple deterministic token-based heuristic.

### Current selection rule

- Build both:
  - classic ARC pack from `internal/contextpack`
  - standalone `ctx` pack from `internal/contexttool`
- Prefer the `ctx` pack when:
  - `ctx.approx_tokens > 0`
  - and either:
    - `ctx.approx_tokens <= arc.approx_tokens` and quality is not worse
    - or `ctx` is within a small token window and has better quality
- Otherwise keep the classic ARC pack
- Persist all of the following for audit and later tuning:
  - `context_pack.{md,json}` as the selected provider-facing pack
  - `arc_context_pack.{md,json}`
  - `ctx_context_pack.{md,json}`
  - `ctx_context_metadata.json`
  - `context_selection.json`
- Mirror the same decision in run metadata:
  - `context_source`
  - `context_arc_tokens`
  - `context_ctx_tokens`
  - `context_token_reduction`
  - `context_arc_quality`
  - `context_ctx_quality`
  - `context_selection_reason`
- `ctx_context_metadata.json` also stores first retrieval-quality signals:
- `ctx_context_metadata.json` now also stores section-level provenance and candidate-vs-final accounting:
  - `quality_score`
  - `term_coverage`
  - `matched_sections`
  - `memory_match_count`
  - `matched_memory_ids`
  - `memory_boost`
  - `memory_trust_bonus`
  - `memory_recency_bonus`
  - `section_provenance`
  - `accounting`
- `.context/memory` now already participates in provider-facing assembly:
  - `ctx assemble` reads saved memory entries,
  - matching entries appear in `Relevant Memory`,
  - memory summary appears in `Memory Summary`,
  - these sections contribute to `matched_sections` and quality scoring.
- ARC run metadata now also mirrors the first memory-aware selection signals:
  - `context_ctx_memory_matches`
  - `context_ctx_memory_boost`
- current selection stays conservative:
  - smaller-or-equal `ctx` packs still win first
  - higher-quality `ctx` packs can win within a narrow token window
  - memory-matched `ctx` packs get an additional extended-window path instead of forcing a full memory-first policy jump immediately
- the next ranking refinement now distinguishes:
  - trust signals such as `decision`, `human`, `high confidence`, `active`
  - recency signals from `last_verified_at` / `created_at`
- those signals flow into:
  - `ctx_context_metadata.json`
  - `ctx bench` summary fields for optimized memory trust/recency
  - ARC run metadata via `context_ctx_memory_trust_bonus` and `context_ctx_memory_recency_bonus`
- the next retrieval refinement also strengthens doc/code ranking without adding a new dependency class:
  - docs are scored more heavily on title and heading coverage, not only path hits
  - files are scored more heavily on basename and multi-term path coverage
  - symbols are scored more heavily on symbol-name matches than on path-only hits
- a later provenance slice on 2026-03-30 made the retrieval path more inspectable:
  - `ctx assemble` metadata now records per-section provenance with `source_paths`, `candidate_count`, and `selected_count`
  - `ctx bench` summaries now compare baseline vs optimized candidate-vs-final totals
  - ARC run metadata now mirrors `context_ctx_candidate_total` and `context_ctx_selected_total` so selection pressure is visible without opening the full standalone artifact
- a later reuse-evidence slice on 2026-03-30 made cache behavior inspectable too:
  - `ctx assemble` metadata now persists `reuse.index_source`, `reuse.memory_source`, and `reuse.reused_artifact_count`
  - `ctx bench` summaries now mirror the same reuse source fields so baseline-vs-optimized comparisons can distinguish retrieval quality from artifact reuse
  - ARC run metadata now mirrors `context_ctx_index_source` and `context_ctx_reused_artifact_count` so top-level run inspection can explain whether `ctx` reused stable standalone artifacts or rebuilt them for the selected pack
- a later diversity slice on 2026-03-30 made optimized-pack quality less one-dimensional:
  - `ctx assemble` metadata now persists `source_kinds`, `source_diversity`, and `diversity_bonus`
  - `ctx bench` summaries now compare `baseline_source_diversity` vs `optimized_source_diversity`
  - ARC run metadata now mirrors `context_ctx_source_diversity` and `context_ctx_diversity_bonus`
  - selection can now also prefer a slightly larger `ctx` pack when it stays within an extended token window and proves stronger cross-surface coverage instead of relying only on smaller token size, raw quality score, or memory matches

## Verification

- deterministic context bundle output
- provenance for included sources
- benchmark comparison artifacts for baseline vs optimized
- live smoke on the repository now confirms the first `ctx bench` path and currently shows a meaningful token reduction on the real repo workload.
- live smoke now also confirms provider-facing selection on a real ARC run: for the task `explain ctx context selection`, ARC selected `ctx` over the classic pack with `794` vs `3465` approximate tokens and persisted the full selection artifact set under the run directory.
- a later smoke on `explain ctx retrieval quality selection` also confirmed the new quality metadata path: `ctx_context_metadata.json` recorded `quality_score=591`, `term_coverage=5`, and matched-section evidence, while `run.json` mirrored `context_arc_quality`, `context_ctx_quality`, and `context_selection_reason`.
- a 2026-03-29 smoke also confirmed the first standalone memory path end-to-end: `ctx memory add` wrote a durable decision to `.context/memory/entries.json`, `ctx memory list` returned it, and `ctx assemble "explain ctx memory influence on context assembly and retrieval quality"` included the entry in both `Relevant Memory` and `Memory Summary`.
- a later smoke on `explain preset environment memory rules and ctx memory influence` confirmed the next quality slice too: `.context/benchmarks/20260329T215004Z/summary.json` reported `optimized_memory_matches=1` alongside a better optimized quality score, `.arc/runs/20260329T215005Z-167175000/ctx_context_metadata.json` recorded `memory_match_count=1` and `memory_boost=40`, and `run.json` mirrored those values as `context_ctx_memory_matches` and `context_ctx_memory_boost`.
- a later smoke on the same task after the trust/recency refinement confirmed deeper memory-aware ranking: `.context/benchmarks/20260329T221937Z/summary.json` reported `optimized_memory_trust_bonus=17` and `optimized_memory_recency_bonus=6`, while `.arc/runs/20260329T221938Z-423559000/ctx_context_metadata.json` and `run.json` mirrored the same trust/recency signals into provider-facing run artifacts.
- a later smoke on the same task after the doc/code ranking refinement confirmed that the optimized pack is now also more discriminative at the surface-selection layer: `.context/benchmarks/20260329T222330Z/optimized_pack.json` elevated the preset-environment docs/code set more cleanly, and `.arc/runs/20260329T222331Z-352551000/run.json` still selected `ctx` while preserving the same memory-aware metadata contract.
- a later smoke on 2026-03-30 confirmed the first operator-support slice too: `ctx doctor --path . --json` reported a healthy `.context/` workspace with self-indexing excluded, `ctx memory status --path . --json` exposed stable artifact paths plus counts, `ctx memory search --path . --json "preset environment"` returned deterministic matches from `.context/memory/entries.json`, and `ctx memory compact --path . --json` preserved the same artifact contract while applying stale-marking rules.
- a later smoke on 2026-03-30 confirmed the first managed-config slice too: `ctx init --path .` ensured `.context-tool.yaml` exists, `ctx doctor --path . --json` reported `config_path` plus the resolved human config, `ctx index build --path .` respected the config-driven include/exclude/docs filters, and both `ctx assemble --json ...` and `ctx bench --json ...` persisted `config_path` + `human_config` in their result metadata.
- a later smoke on 2026-03-30 confirmed the first reuse-evidence slice too: `.context/artifacts/assemble/20260330T203305Z/metadata.json` recorded `reuse.index_source=reused_existing`, `reuse.memory_source=reused_existing`, and `reuse.reused_artifact_count=2`; `.context/benchmarks/20260330T203302Z/summary.json` mirrored the same reuse summary fields; and `.arc/runs/20260330T203303Z-771727000/run.json` surfaced `context_ctx_index_source=reused_existing` plus `context_ctx_reused_artifact_count=2` without needing to open the full standalone artifact.
- a later smoke on 2026-03-30 confirmed the first diversity slice too: `.context/artifacts/assemble/20260330T205625Z/metadata.json` recorded `source_kinds=["task","docs","code","memory","index"]`, `source_diversity=5`, and `diversity_bonus=90`; `.context/benchmarks/20260330T205625Z/summary.json` mirrored `baseline_source_diversity=5` vs `optimized_source_diversity=5` plus `optimized_diversity_bonus=90`; and `.arc/runs/20260330T205626Z-351423000/run.json` surfaced `context_ctx_source_diversity=5` with `context_ctx_diversity_bonus=90`.
