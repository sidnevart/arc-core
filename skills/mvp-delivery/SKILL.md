---
name: mvp-delivery
description: Use when working on the Agent Runtime CLI MVP in this repository. Covers the default end-to-end workflow for turning the product brief into a verified implementation slice with updated docs and project memory.
---

# MVP Delivery

Use this skill for implementation work in this repository when the goal is to move the `arc` MVP forward, not to brainstorm in the abstract.

## Workflow

1. Read `plan.md` and the relevant files in `memory_bank/`.
2. Reduce the request to one shippable slice.
3. Identify unknowns that block implementation.
4. Implement the smallest useful end-to-end change.
5. Verify behavior with the best local checks available.
6. Update docs and `memory_bank/` if project understanding changed.

## Default Slice Order

Prefer this sequence unless the user explicitly redirects priorities:

1. CLI and config foundation
2. project scaffold and init
3. provider detection
4. mode policies
5. context and memory primitives
6. orchestration and artifact generation
7. verification, review, and docs automation

## Guardrails

- Do not invent repo facts.
- Mark uncertain statements as `INFERRED` or `UNKNOWN`.
- Keep provider-neutral logic in the core and provider-specific behavior in adapters.
- Do not add UI, mandatory vector DB, or speculative architecture not required for the current slice.
- Prefer simple durable code over framework-heavy abstraction.

## Subagent Use

Use subagents when they reduce wall-clock time without taking over critical-path design.

- `explorer` for spec or codebase fact gathering
- `worker` for isolated implementation
- `reviewer` for independent verification

## Artifacts

After meaningful work, leave behind durable artifacts:

- implementation
- verification notes
- updated docs
- updated decision/open-question/worklog entries when needed

For the default artifact checklist, read `references/artifact-checklist.md`.
