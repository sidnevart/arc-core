# Provider Budget Checklist

- `budget_assessment.json` is present for every run path under test.
- `prompt_minimization.json` is present for every run path under test.
- `budget_usage_event.json` and `.arc/budget/usage_events.jsonl` stay in sync.
- `route_locally=true` appears only for `no_provider`, `local_first`, or explicitly low-confidence `cheap_provider_ok` cases under emergency low-limit policy.
- Premium-required work is not misclassified because of weak local signals.
- Persisted `.arc/budget/policy.json` matches the effective requested mode.
- Assessment and run metadata persist `low_limit_state`, `confidence`, and `matched_signals` when budget routing changes behavior.
- Assessment and run metadata also persist `confidence_tier` plus a readable `signal_breakdown` when classification/routing behavior changes.
- When low-limit routing changes behavior, `budget_routing_trigger` must explain whether the reroute came from `local_first`, `emergency_low_confidence`, `constrained_low_confidence`, or another explicit trigger.
- If `--budget-mode` is omitted, preset-linked `environment_budget_profile` can supply the effective mode; an explicit flag must still win over that preset default.
- Optional `.arc/budget/project_override.json` and `--budget-override-file` must appear in `budget_policy_resolution.json` and change the effective policy deterministically when present.
- `budget_assessment.json` and run metadata must agree on `budget_mode_source`, `budget_project_override_present`, `budget_session_override_present`, and `budget_override_sources`.
- `budget_assessment.json` must embed `prompt_minimization`, and `budget_usage_event.json` must mirror the same context-token/minimization fields.
- `budget_usage_event.json` should also carry the real provider execution footprint when available, including `provider_model` and `provider_session_id`.
- `arc budget show` and `arc budget override set|clear` must read/write the project override file without breaking subsequent run-time policy resolution.
- `arc budget session write|show|clear` must round-trip a session override file cleanly, and a later `task run --budget-override-file` must reflect that file in `budget_policy_resolution.json`.
