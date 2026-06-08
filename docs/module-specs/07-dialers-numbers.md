# Module 7 — Dialers: Numbers (core-feature spec)

- **Status:** draft → **core API question RESOLVED (s6 research, 2026-06-04)**. Server APIs exist for all three vendors — the Chrome extension was a workaround, not a technical necessity. The residual unknowns are vendor API-*access* onboarding (sales-gated for Hiya Number Reputation + Number Verifier), not API existence. Gate for Track E1 — Dialer subsystem · order 2.
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
- **Goal:** drop the Chrome-extension dependency entirely. **s6 research confirms this is achievable** — Dialpad, Hiya (Number Reputation API), and Number Verifier all expose server APIs (see Findings). The extension is retained only as an **interim bridge** until vendor API credentials are provisioned — and possibly kept for Number Verifier alone if its (sales-gated, publicly-undocumented) API proves inadequate.

## Migration notes & kill-switch trigger
- **Trigger:** numbers sync + match run Dialpad→worker→Hiya/NV through NATS with results in Neon, replacing the extension bridge.

## Findings — vendor API surface (s6 research, 2026-06-04)

The CRITICAL unknown ("do these expose server APIs, or is the extension mandatory?") is **resolved: server APIs exist for all three.** The extension was a workaround, not a technical wall.

| Vendor | Server API? | Access model | Auth | Replaces extension? |
|---|---|---|---|---|
| **Dialpad** (roster source) | ✅ Documented REST — `GET /api/v2/numbers` lists all org numbers (cursor pagination, `status` filter, 1200 req/min) + `GET /api/v2/numbers/{number}`. | ✅ **Self-serve** — company-admin API key | Bearer API key | ✅ **Yes** — pull the roster server-side, no browser. *Dialpad gives the number roster + assignment, NOT spam labels.* |
| **Hiya** (register + reputation) | ✅ Documented **Number Reputation API** — register business, register/remove phone numbers (batched, named jobs), get number status, **fetch reputation data** (the `hiyaSpamLabel` source). | ◐ **Sales-gated** — "connect with a Hiya service agent" for Number Reputation access (Connect/Audio-Intel tiers are self-serve; a free Number-Registration tier exists via the Hiya Connect console). | Basic (AppID:AppSecret) → short-lived token, or long-lived API key | ✅ **Yes, once access granted** — register numbers + read spam label via API. |
| **Number Verifier** (numberverifier.com) | ◐ A "Verify API" product exists; **specs are not public** (gated behind sales/demo). Core product = caller-ID reputation monitoring, Phone-Number Audit, **daily spam/scam alerts**, STIR/SHAKEN attestation checks, "Device Cloud" of real-device screenshots. | ◐ Free trial (`/number-upload`); full access = contact sales | Unknown (gated) | ◐ **Partial** — API exists but its contract must be obtained from NV directly; keep the extension/manual path for NV until confirmed. |

**Bottom line for Track E1:** it is a **clean server integration**, NOT a permanently-kept browser bridge. Dialpad is immediately self-serve. Hiya and Number Verifier are technically API-capable but **gate access behind a vendor onboarding / sales conversation** — so the real E1 prerequisite shifts from "is it possible?" (yes) to "**obtain API credentials**" (a partner/sales lead-time task). The extension stays only as the interim bridge until those creds land.

_Sources: Dialpad — `developers.dialpad.com/reference/numberslist`, `/docs/welcome` · Hiya — `developer.hiya.com` (Number Reputation API; `getting-started/authentication` — Number Reputation access is agent-gated, Connect/Audio-Intel self-serve) · Number Verifier — `numberverifier.com` (pricing, device-cloud; "Verify API" referenced, specs sales-gated)._

## Residual open questions (access/onboarding — not blockers to *designing* E1)
- **Vendor API onboarding (lead-time — start early):** request Hiya Number Reputation API access (service agent) + Number Verifier API contract/specs (sales). *Until both land, the extension bridge remains the NV/Hiya registration path.*
- **Number Verifier vs Hiya overlap:** NV appears to be a paid monitoring/remediation layer that may itself register across analytics engines (Hiya/TNS/First Orion, à la Free Caller Registry). Confirm whether NV + Hiya are redundant or complementary before building two adapters. (Free Caller Registry — `freecallerregistry.com` — is the *free, no-API, web-form* equivalent; NV is the paid, API-bearing alternative.)
- **Number source of truth:** Dialpad as the roster (confirmed: it has the API + assignment/department data) vs GHL/Twilio. **Recommend Dialpad.**
- **Auth storage:** all three vendors' credentials live in the encrypted `integration_credentials` store (Track B2), never in the browser (ADR-001 generalized).
