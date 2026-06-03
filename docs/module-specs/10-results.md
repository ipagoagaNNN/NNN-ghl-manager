# Module 10 — Results (core-feature spec)

- **Status:** draft (cross-cutting shared component + job-status surface)
- **Target home:** Shared UI component + backend job/run status (reads sync/outbox/run records)
- **Current state in repo:** inline in the prototype (`logBox`, `sOk`, `sErr`, `progFill`); not yet a discrete component.

## Purpose
A shared progress/results surface that any bulk or long-running operation (CV bulk update, sync run, CSV import, publish) reports into: OK/Error counts, a progress bar, and navigation back to the originating module.

## Core features
1. **Progress log** — running OK / Error counts with per-item detail.
2. **Progress bar** — completion percentage for the active operation.
3. **Back navigation** — return to the module that launched the operation.

## Data model
- No new domain entity. Reads from operation outcomes that already exist in the platform:
  - `outbox_events` / `sync_state` (sync run results),
  - bulk-CV PUT results (Module 3),
  - `events.dialer.*` outcomes (Modules 7/8 import/match),
  - `publishes` (Module 4 publish results).

## Target behavior in the new platform
- A shared Svelte component (`components/shared/ResultsLog.svelte`) **plus** a thin backend **job/run status** read model so results survive a page refresh and are queryable (the prototype's were ephemeral DOM state).
- Naturally exposed by the Claude shell too (`sync_state`, `dlq_peek`, `system_status` map to the same data).

## GHL API surface & ceiling
- None — purely internal operation reporting.

## Migration notes & kill-switch trigger
- Cross-cutting; nothing to retire. Grows as each module's operations start emitting run records.

## Open questions
- **Generic "activity / jobs" surface?** Should Results become a first-class jobs view (every async op, with history), backed by a `jobs`/`runs` table — or stay a lightweight per-operation log?
- **Persistence depth:** keep full run history, or just the last run per operation?
- **Real-time:** push progress (SSE/WebSocket from NATS) vs poll.
