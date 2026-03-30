# Context Tool Smoke Checklist

- `.context/config.json` exists after `ctx init`.
- `.context-tool.yaml` exists after `ctx init` and is reported by `ctx doctor`.
- `ctx doctor` reports the workspace as healthy and confirms `.context/` is excluded from the machine index.
- `ctx doctor` reports the resolved human config separately from machine-managed `.context/config.json`.
- `.context/index/` artifacts refresh without indexing `.context/` itself.
- `ctx index build` respects `.context-tool.yaml` include/exclude/docs path filters.
- `.context/memory/entries.json` stays readable and synced to `MEMORY_ACTIVE.md`, `MEMORY_ARCHIVE.md`, and `OPEN_QUESTIONS.md`.
- `ctx memory status|search|compact` behave deterministically on the same workspace.
- `assemble` writes `context_pack.json`, `context_pack.md`, and `metadata.json`, and the result reports `config_path` plus the resolved human config.
- `assemble` metadata records `section_provenance` and aggregate `accounting`.
- `assemble` metadata records reuse evidence via `reuse.index_source`, `reuse.memory_source`, and `reuse.reused_artifact_count`.
- `bench` writes baseline, optimized, and summary artifacts, and the summary includes candidate-vs-final totals plus reuse-source fields for both strategies.
- ARC run path can still mirror selected `ctx` artifacts into `.arc/runs/...`.
- ARC run metadata mirrors top-level `ctx` reuse fields such as `context_ctx_index_source` and `context_ctx_reused_artifact_count`.
