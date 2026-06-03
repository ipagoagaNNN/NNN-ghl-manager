# Module 7 — Dialers: Numbers (core-feature spec)

- **Status:** draft (gate for Track E1 — Dialer subsystem · order 2). **Biggest external unknown.**
- **Target home:** Dialer subsystem — Dialpad/Hiya/Number-Verifier adapters + the `number-matcher` Rust worker wired via NATS
- **Current state in repo:** `workers/number-matcher` (Rust/axum, **standalone :8081, not wired to Go**), `stores/dialers.svelte.ts` + `stores/extension-bridge.svelte.ts` (types + state), `handlers/stub.go` (`UpdateNumbersLibrary` stub), `routes/dialers/+page.svelte` (placeholder). Architecturally orphaned from both ends.

## Purpose
Match the agency's phone numbers across **Dialpad ↔ Number-Verifier (NV) ↔ Hiya**, register numbers with NV/Hiya, and track spam labels — so outbound calls aren't flagged as spam.

## Core features
1. **Sync numbers** — pull the current numbers snapshot (today: from the Chrome extension).
2. **Display NV/Hiya match by office / department** — per-number registration + spam-label state.
3. **Filter** — by office, department, free-text search.
4. **Register a number (NV / Hiya / Both)** — today via `postMessage ghl-register-number`.
5. **Numbers library** — persist match/registration state across syncs (survives re-sync).
6. **Matching logic** — set operations (which numbers are in NV, in Hiya, deleted) → the **Rust `number-matcher`** (`match_numbers`).
7. **Deleted-numbers detection** — numbers in the library no longer present upstream.

## Data model
`NumberItem` (rich: number, hiyaNumber, hiyaSpamLabel, office/department, assignedType/Name, status, `inNumberVerifier`, `inHiya`, nv/hiya status) + `NumbersMatchData` (counts, syncedAt, source) + `NumberLibraryEntry` (keyed by `compactPhone`) → new **`dialer_numbers`** table (Neon) with an `external_ids`/state JSONB for NV/Hiya. See `data-models.md`.

## Integration surface (NOT GHL)
- **Today:** Chrome extension `cnam-extension` via `postMessage` + `CustomEvent` (`ghl-numbers-sync`, `ghl-register-number`, `ghl-register-number-result`, `ghl-library-patch`) + a localStorage inbox fallback. See `extension-bridge.md`.
- **Target:** server-side via **NATS** — `number-matcher` becomes a JetStream consumer: consumes `commands.dialer.match`, emits `events.dialer.match.completed`; reconciler writes results to Neon. Dialpad/Hiya/NV become **adapters** (auth + register calls).

## Target behavior in the new platform
- Replace `axum::serve` in `number-matcher` with an `async-nats` subscribe loop; **reuse `match_numbers` verbatim** (transport-only change).
- A small Go orchestrator pulls Dialpad numbers → publishes `commands.dialer.match` (with Dialpad+Hiya+NV data) → worker matches → results to Neon. Registration goes through the Hiya/NV adapters.
- **Goal:** drop the Chrome-extension dependency entirely — *if* the upstreams expose server APIs (see Open questions).

## Migration notes & kill-switch trigger
- **Trigger:** numbers sync + match run Dialpad→worker→Hiya/NV through NATS with results in Neon, replacing the extension bridge.

## Open questions (CRITICAL — resolve before building)
- **Do Dialpad / Hiya / Number-Verifier expose server-side APIs?** The prototype used a Chrome extension specifically — likely because some of these have **no public API** (the extension drives their web UI). If so, the extension **cannot be fully dropped**; the "adapter" may have to remain a browser automation. *This determines whether Track E1 is a clean server integration or a kept browser bridge.*
- **Auth** for each (Dialpad API token? Hiya partner API? NV?).
- **Number source of truth:** Dialpad as the roster, or GHL/Twilio?
