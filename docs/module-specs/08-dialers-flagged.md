# Module 8 — Dialers: Flagged Numbers (core-feature spec)

- **Status:** draft — CSV source identified (s6: likely **Number Verifier's** daily spam/scam export); exact column schema still to confirm from a sample. Gate for Track E1 — Dialer subsystem · order 2, alongside Numbers.
- **Target home:** Dialer subsystem — `csv-processor` Rust worker via NATS + R2 for CSV blobs
- **Current state in repo:** `workers/csv-processor` (Rust/axum, **standalone :8082, not wired to Go**; `chrono_now()` returns `""` — `imported_at` bug), extension `ghl-flagged-import` handler in stores, `routes/dialers/` (placeholder).

## Purpose
Import and track **spam-flagged numbers** (CSV), visualize them by department, and surface spam labels so the agency can remediate flagged lines.

## Core features
1. **CSV import (file picker)** — frontend parse, or server-side parse via the worker.
2. **CSV import from extension** — `ghl-flagged-import` CustomEvent (`{ fileDataUrl, filename, importedAt }`).
3. **Preview table with search** — render rows, filter.
4. **Department chart** — counts by department.
5. **Spam-label tracking** — display the flagged label per row (display-only).

## Data model
`FlaggedData { headers: string[], rows: Record<string,string>[], fileName, importedAt }` →
- a `flagged_numbers` table (or rows folded into `dialer_numbers` with a `spam_label` + `flagged_at`) in Neon.
- Raw CSV files → **Cloudflare R2** (not the DB), referenced by key.

## Integration surface (NOT GHL)
- **Today:** file picker or the extension's `ghl-flagged-import` event + localStorage inbox fallback.
- **Target:** server-side — upload CSV → R2 → publish `commands.dialer.csv.process` (CSV ref) → `csv-processor` (NATS consumer) parses → emits `events.dialer.flagged.imported` → reconciler writes rows to Neon. Reuse `process_csv` verbatim; **fix `chrono_now()`** so `imported_at` is a real timestamp.

## GHL API surface & ceiling
- None — GHL is not involved. Data originates from an external spam/flag report (s6 finding: most likely **Number Verifier's daily spam/scam export** — see Open questions).

## Target behavior in the new platform
- `csv-processor` becomes a JetStream consumer; the Chrome-extension `ghl-flagged-import` path is replaced by a server-side upload endpoint.
- Flagged state can cross-reference Module 7 numbers (join on `compactPhone`) to show "this office line is flagged."

## Migration notes & kill-switch trigger
- **Trigger:** flagged CSV ingested server-side (upload → R2 → worker → Neon), replacing the extension `ghl-flagged-import` path.

## Open questions
- **CSV source & schema (s6 finding — likely Number Verifier):** the flagged-numbers CSV most plausibly comes from **Number Verifier's daily spam/scam alerts / Phone-Number Audit export** (numberverifier.com "provides daily alerts for any caller IDs labeled as scam or spam"). Hiya's Number Reputation API (M7) is an alternative/secondary source for per-number labels. **To confirm:** obtain a sample export from NV and pin the exact column schema — it drives the parser (`process_csv` must map its real headers). Schema stability is vendor-controlled.
- **Remediation action:** is this display-only, or should a flagged number trigger a re-register (Module 7) automatically?
- **Retention:** how long to keep flagged history / the raw CSVs in R2.
