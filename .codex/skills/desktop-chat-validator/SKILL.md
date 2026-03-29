---
name: desktop-chat-validator
description: Validate ARC Desktop chat UX and miniapp runtime behavior before accepting changes. Focus on message/demo state sync, chat detail usefulness, compact chat layout, and smoke coverage for launch/restart/stop flows.
---

# Desktop Chat Validator

Use this skill when changing ARC Desktop chat UX, drawer/details behavior, or live demo/simulation flows.

## Goal

Catch regressions where desktop chat looks correct in code but breaks in real usage:
- demo starts from one surface but not another
- message card and chat details show different runtime states
- stale `Приложение остановлено` survives after launch
- chat chrome grows until the thread stops feeling like a real chat

## Required checks

Before accepting a change:
1. Inspect the desktop chat shell in:
   - `apps/desktop/wailsapp/frontend/app.js`
   - `apps/desktop/wailsapp/frontend/styles.css`
   - mirrored preview files under `apps/desktop/static/`
2. Inspect backend runtime/detail assembly in:
   - `internal/app/service.go`
   - `internal/liveapp/liveapp.go`
   - `internal/desktop/bridge.go`
3. Run the smoke checklist in `references/desktop-chat-smoke-checklist.md`.

## Validation stance

Prefer concrete findings over aesthetic opinions.

Focus on:
- one source of truth for miniapp runtime state
- useful `Детали чата`
- compact chat-first proportions
- working `Запустить снова / Открыть в ARC / Остановить`
- visible launch feedback on the card that was clicked

## Required artifact updates

If you find a durable issue, update:
- `memory_bank/active_context.md`
- `memory_bank/backlog.md` or `memory_bank/unknowns.md`
- relevant desktop docs if user-visible behavior changed

