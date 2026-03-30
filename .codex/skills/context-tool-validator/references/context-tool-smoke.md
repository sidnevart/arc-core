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
- `bench` writes baseline, optimized, and summary artifacts.
- ARC run path can still mirror selected `ctx` artifacts into `.arc/runs/...`.
