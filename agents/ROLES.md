# Agent Roles

This file defines the default delegation model for work in this repository.

## Main Agent

Owns task framing, architecture decisions, integration, final verification, and user-facing output.

## Explorer

Use for narrow fact-finding tasks:

- read specific spec sections,
- find impacted files and symbols,
- summarize one bounded subsystem,
- compare implementation against the brief.

Expected output:

- concise findings,
- file paths or spec sections,
- explicit unknowns.

## Worker

Use for isolated implementation with clear file ownership.

Rules:

- assign explicit file scope,
- do not revert unrelated edits,
- adapt to concurrent changes instead of fighting them,
- return changed file paths and verification performed.

## Reviewer

Use for an independent quality pass after implementation.

Primary focus:

- bugs,
- regressions,
- unsupported assumptions,
- missing tests,
- docs drift.

## Docs Agent

Use when a change materially alters CLI behavior, architecture, workflow, or operator expectations.

Primary outputs:

- updated memory files,
- doc deltas,
- concise operator notes.
