---
name: provider-budget-validator
description: Validate ARC provider budget classification, routing, policy precedence, usage ledgers, and low-limit protections before accepting budget work.
---

# Provider Budget Validator

Use this skill when `internal/budget`, provider routing, approval behavior, or budget artifacts change.

## Validate

1. Every provider-bound request gets an assessment artifact.
2. Local-first tasks reroute locally only when classification is strong enough.
3. Premium/high-risk tasks do not bypass policy.
4. Requested budget mode and persisted policy stay aligned.
5. Usage events remain attributable and auditable.

## Required checks

- `go test ./internal/budget ./internal/orchestrator`
- `go run ./cmd/arc task run --path . --budget-mode balanced "inspect context tool budget schema"`
- `go run ./cmd/arc task run --path . --budget-mode balanced "inspect and implement the budget schema"`

## References

- `references/provider-budget-checklist.md`
