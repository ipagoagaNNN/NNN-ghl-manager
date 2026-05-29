# Phase 2e — Verified Implementation Path

**Gate:** this doc is the "clear path" the user required before any Sites & Funnels code.
It splits Phase 2e into **2e-1 (ship now, read + audit)** and **2e-2 (deferred, pixel write —
needs a user decision)**.

Backed by `funnels-api-contract.md`. One-line reason for the split: **the public GHL API v2
Funnels group is read-only**, so the original "inject pixel" feature is not achievable with the
location PIT our vault holds.

---

## Phase 2e-1 — Sites & Funnels READ + Pixel Audit  ✅ ship

Fully supported by the public API + server-side page fetch. Mutates nothing. This is most of the
real operator value: *"which funnels/pages are missing the right FB pixel / UTM / Acuity script?"*

### Backend (Go)

1. **`GET /api/funnels/{locationId}`** — list funnels.
   - Vault token (401 if none), `Version: 2021-07-28`.
   - Call `GET /funnels/funnel/list?locationId={loc}&limit=100`, paginate if needed.
   - Envelope fallback `funnels || data || list || items`.
   - For each funnel: resolve domain (`funnel.domain|domainName|url|previewUrl`, else
     `GET /locations/{loc}/domains/{domainId}`), return funnel + steps (name, slug) + resolved domain.

2. **`GET /api/funnels/{locationId}/audit?funnelId=...`** — pixel/tracking audit.
   - For each step, build `https://{domain}/{slug}` and **fetch the live page server-side**
     (no CORS proxy, no browser). Reuse the SSRF-hardened request pattern from `connect.go`
     (`http.NewRequestWithContext` + host validation + `// #nosec G107 G704`), but here the host is
     the funnel's *resolved* domain, so lock the fetch to that exact host and cap redirects.
   - Bounded concurrency (semaphore = 6, like the CV fan-out in `customvalues.go`), per-page error
     isolation, 60s TTL cache keyed by `locationId+funnelId`.
   - Detect (regex/string, from contract §3): hasPixel, pixelId, hasUTM, hasAcuity, acuityFieldId.
   - Compare `pixelId` against the **expected pixel for that domain** → `ok | missing | wrong-pixel`.
   - Response per page: `{ stepName, url, fetchOk, hasPixel, pixelId, expectedPixel, status, hasUTM, hasAcuity }`.

3. **Expected-pixel config.** Move the hardcoded known-pixels map (contract §3) out of code into a
   domain→pixel config, mirroring `frontend/src/lib/data/brand-presets.ts`. Where the expected pixel
   lives server-side vs client-side: keep the *audit comparison* server-side so the verdict is
   authoritative; the frontend just renders it.

### Frontend (Svelte 5)

- **Sites & Funnels page** (`/funnels` route): per-connected-account funnel cards.
- Each funnel: list steps/pages with audit badges — ✓ pixel (id), ✗ missing, ⚠ wrong pixel
  (expected X got Y), UTM ✓/✗, Acuity ✓/✗.
- **"Super Scan all funnels"** summary across the location (counts: funnels, pages, ok, missing,
  wrong) — the prototype's most-used feature (HTML L5170).
- Skeleton loaders during scan; the scan is the slow part (N live fetches).

### Security notes (carry the vault invariants forward)

- Token stays in the vault; `/api/funnels/*` responses never echo it (`hasToken:bool` pattern).
- The live-page fetch is **SSRF-sensitive** — the target host is derived from GHL data, not user
  input, but still validate it's the resolved funnel domain and refuse internal/loopback hosts.
  Cap response size + timeout; don't follow cross-host redirects.
- **Drop the public CORS proxies entirely.** The prototype routed customer page HTML through
  third-party proxies (corsproxy.io etc.) — a privacy/reliability liability. Server-side fetch
  removes them. (Call this out as a security win in the session log.)

### Verify (same bar as 2b/2c/2d)

`go build ./... && go vet ./... && gosec ./...` (0 findings) + `svelte-check` (0 errors) + **live
smoke test** against Miriam Test location PIT: list funnels → audit → confirm pixel verdicts match
reality. Fill the live-status matrix in `funnels-api-contract.md` §5.

---

## Phase 2e-2 — Pixel WRITE / injection  ✅ DECIDED 2026-05-29 (s4): Option C — Assisted manual

**User decision (session 4):** Option **(C) Assisted manual.** No automated writes, no browser-session
token capture. The audit flags missing/wrong pixels; the tool hands the operator the exact snippet to
paste into GHL.

**Shipped this session:** for any funnel with no pixel in its tracking code, the Funnels page shows an
"Add Meta pixel" panel → brand picker (auto-selected by domain when known) → read-only snippet +
Copy button + "Open funnels in GHL" deep link + paste instructions (Funnels → Settings → Tracking
Code Head). Snippet config lives in `frontend/src/lib/data/pixel-snippets.ts` (template-generated;
brand list mirrors the backend's `expectedPixels`).

Options considered (C chosen):

| Option | What it is | Cost / risk |
|--------|-----------|-------------|
| **(A) Browser-session capture** | Port the prototype's `token-id`/`Authorization`/`channel:APP`/`source:WEB_USER` path (+ bookmarklet to capture the session token) to do real writes. | High. Fragile (breaks on GHL session-auth changes), ToS-grey, captured token is *more* sensitive than a PIT — raises the vault's threat model. |
| **(B) Read-only, fix in GHL UI** | Drop write. The tool reports what's missing; operator fixes pixels in GHL's native funnel settings. | Low. Honest. Loses one-click convenience. |
| **(C) Assisted manual** | Render the exact expected pixel/UTM/Acuity snippet + copy button + deep link to the page's GHL settings, so the operator pastes it. | Low. Keeps most of the convenience without the fragile auth. |

**Recommendation: ship 2e-1, then do (C) as the write story, hold (A) unless the user explicitly
wants automated injection and accepts the fragility.** Either way, 2e-1's audit value does not depend
on resolving this — do not let the write decision block the read/audit ship.

---

## Sequencing

1. **2e-1 backend** (list + audit handlers + expected-pixel config) → verify.
2. **2e-1 frontend** (funnels page + super-scan) → verify + live smoke.
3. **Surface the 2e-2 decision** to the user (A/B/C) — separate turn, separate scope.
4. Update roadmap: 2e read+audit shipped; 2e write tracked as a decision-gated sub-item.

## Out of scope for 2e (already deferred elsewhere)

Forms/TriggerLinks tabs + NPL scanner (2c+), Dialers (2f), TxtGen (2g). The funnel-step
form/booking detection (`isFormStep`/`isBookingStep`, HTML L2916) is read-only metadata we can use
to *label* steps in the audit, but form *editing* stays in 2c+.
