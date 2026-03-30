---
name: context-tool-validator
description: Validate standalone ctx behavior, .context storage, memory flows, context assembly artifacts, and benchmark comparisons before accepting context-tool work.
---

# Context Tool Validator

Use this skill when `cmd/ctx`, `.context/`, retrieval, memory, assembly, or benchmark behavior changes.

## Validate

1. `ctx` still works as a standalone-first boundary, not just as an ARC helper.
2. `.context/` remains stable and the tool does not index its own workspace.
3. Memory add/list/search/status/compact behavior stays deterministic.
4. `assemble` writes provenance-bearing artifacts.
5. `bench` still compares baseline vs optimized with measurable outputs.

## Required checks

- `go test ./internal/contexttool ./internal/ctxcli ./internal/orchestrator`
- `go build ./cmd/ctx`
- `go run ./cmd/ctx init --path .`
- `go run ./cmd/ctx index build --path .`
- `go run ./cmd/ctx assemble --path . "<task>"`
- `go run ./cmd/ctx bench --path . "<task>"`

## References

- `references/context-tool-smoke.md`
