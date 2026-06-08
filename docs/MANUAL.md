# NNN-GHL-manager — Manual

> **Scope.** This manual documents **every feature** of NNN-GHL-manager — both **current** (shipped on `main`, builds verified) and **future** (the accepted reference architecture, designed but not yet built).
> **Legend:** ✅ shipped · ◐ partial (core works, sub-features pending) · 📝 designed, not built · ⛔ blocked.
> Last updated: 2026-06-08 (session 6 — live-verified; see §11). Authoritative status lives in `roadmap/_roadmap.md` (vault); per-module detail in `docs/module-specs/`.

---

## 1. What it is

NNN-GHL-manager is a multi-sub-account management dashboard for a marketing agency running on **GoHighLevel (GHL)**. It began as a refactor of a 12,600-line HTML prototype into a **Svelte 5 SPA + Go backend proxy + Rust workers**, and is evolving into a full **CRM + CMS platform** that absorbs GHL's data and then retires GHL adapter-by-adapter.

**Core invariant (ADR-001):** GHL Private Integration Tokens (`pit-…`) **never reach the browser.** The frontend only ever calls `/api/*` on our own Go backend; the backend holds the tokens server-side and talks to GHL.

---

## 2. Quick start (development)

Two processes. From the repo root (`C:\Users\get_h\Documents\VSCode\NNN-GHLmanager`):

```powershell
# Terminal 1 — backend API (:8091)
cd backend
go run ./cmd/server

# Terminal 2 — frontend SPA (:5173, proxies /api → :8091)
cd frontend
npm run dev
```

Then open **http://localhost:5173**. You land on **Connect**.

| Port | Process | Notes |
|------|---------|-------|
| `:8091` | Go backend | `:8080/:8090/:9090` are reserved by Docker on the dev machine |
| `:5173` | Vite dev server | `/api` is proxied to `:8091` (see `frontend/vite.config.ts`) |

**Build / verify:**
```powershell
cd backend ;  go build ./... ;  go vet ./...        # + gosec ./... if installed
cd frontend ; npm run check                          # svelte-check
```

**First-run reality:** there is **no database yet** (by design). All tokens + metadata live in the backend's in-memory vault and are **lost when the backend restarts**. A browser refresh also resets the frontend session → you return to `/connect`. Treat the tool as a single-run workbench until the platform foundation (Track B) lands.

---

## 3. Architecture (current)

```
 Svelte 5 SPA (frontend/, :5173)
   /connect /accounts /custom-values /schema /funnels /automations /dashboard /dialers
        │  fetch /api/*   (tokens never cross this line — ADR-001)
        ▼
 Go backend (backend/, :8091, stdlib only)
   middleware: CORS · RateLimit(stub) · Auth(stub until Phase 2)
   handlers/  → typed per-module endpoints
   proxy/     → generic /api/ghl/* passthrough (per-path Version header)
   store/     → in-memory Vault (tokens + per-location metadata) + GHLVersionFor
        │  Bearer pit-… + per-endpoint Version
        ▼
 GoHighLevel API v2  (services.leadconnectorhq.com)

 Rust workers (workers/, standalone — NOT yet wired):
   number-matcher (:8081)   csv-processor (:8082)
```

- **Frontend:** SvelteKit + Svelte 5 runes (`$state`/`$derived`), single SPA, dependency-light. State stores in `src/lib/stores/`.
- **Backend:** Go 1.23, **standard library only** (no web framework). Module `github.com/ipagoagaNNN/nnn-ghl-manager/backend`. Entry: `cmd/server/main.go`.
- **Token vault:** `store/vault.go` — RWMutex maps for `locTokens` + `locMeta`. Method surface is stable; Phase 2 swaps the backing for an encrypted Postgres store with zero handler churn (ADR-003).
- **GHL version map:** `store/version.go` → `GHLVersionFor(path)` returns the correct `Version` header per endpoint (see §6). One choke point; the proxy and every handler use it.
- **SSRF guard:** every outbound call is host-locked to the hardcoded GHL base; `#nosec` annotations document why (`gosec` clean).

---

## 4. Current features (by module)

### 4.1 Connect ✅
Two ways to authenticate, both store the token server-side and return only non-secret data:
- **Agency PIT** (`POST /api/connect`) → validates against `/locations/search`, lists all sub-accounts.
- **Location PIT** (`POST /api/connect/location`) → for location-scoped tokens that 403 on agency search but 200 on location endpoints. Validates via `/customValues`, stores the token, makes the location usable everywhere.
- Status dot + sub-account count in the top bar.

### 4.2 Accounts ✅
Per-sub-account library: save a token (password-masked; server-only), edit metadata (domain, Acuity field, calendar IDs, active flag), CSV import/export of the library, search + multi-select to scope other modules. Responses carry `hasToken: boolean` only — never the token value.

### 4.3 Custom Values ✅ (+ NPL scanner, + delete — new in s6)
The **only GHL-writable module** (`PUT customValues` is the single supported write).
- **Bulk view/edit** across many sub-accounts in one pass — parallel fan-out (`GET /api/cv`), per-cell staged edits, **bulk apply** (`POST /api/cv/bulk`) with partial-success results.
- **Brand presets** — one-click bulk-fill for NNN / FTB / Advanced Beauty / General.
- **New Patient Link (NPL) scanner** ✅ — a read-only cross-account audit: finds each account's `New Patient Link` custom value, validates the URL against the account's registered domain → **valid / domain-mismatch / malformed / missing**, with summary pills, a problems-only filter, and a one-click bulk-fix routed through the normal CV write. (CV key is configurable; defaults to "New Patient Link".)
- **Delete a custom value** ✅ — per-row 🗑 with confirm (`DELETE /api/cv/{loc}/{cvId}`).
- ◐ **Inner Forms / Trigger Links tabs** — planned (read-only), not yet built.

### 4.4 Schema ✅ (new in s6)
Per-account, read-only viewer of GHL **objects** (built-in `business`/`opportunity`/`contact` + any custom objects) and the **custom-field schema**. Backed by `GET /api/objects/{loc}` (Version `2023-02-21`) and `GET /api/customfields/{loc}` (Version `2021-04-15`). Pick an account, see both schemas.

### 4.5 Automations ✅ (read-only)
Lists GHL **workflows** per account (`GET /api/workflows/{loc}`) — name, status, version, dates. Read-only by necessity: the Workflows API is **list-only** (no detail, edit, toggle, or trigger via the public API).

### 4.6 Dashboard ✅
Leads analytics per sub-account: contacts paginated from GHL, aggregated **leads-by-day** (inline SVG chart, no chart dep), **top sources**, per-account table, date-range filter, explicit empty-state. Guards against GHL's unreliable `page` pagination (duplicate-page signature + `nextPageUrl` stop). Campaign breakdown + FB-attribution + converted/booked metrics are **available but not yet built** (need `customFields` name resolution — now provided by §4.4).

### 4.7 Sites & Funnels ◐
- **Read + audit** ✅ — list funnels/pages, server-side fetch published pages (SSRF-guarded), detect Meta Pixel + UTM markers, per-page present/missing verdict vs the expected brand pixel.
- **Assisted-manual pixel fix** ✅ — hands the operator the exact snippet + copy button + a GHL deep link to paste into Funnel → Settings → Tracking Code.
- ⛔ **Automated pixel/page write is impossible** via the public API (no funnel/page write endpoint for PIT *or* OAuth). The prototype's auto-injection used a captured browser-session token — **rejected** (fragile, ToS-grey, raises the threat model). `PUT /api/funnels/page/{id}` is a deliberate `not_yet_implemented` stub. The real solution is the CMS (§8, Track E2).

### 4.8 Dialers — Numbers + Flagged ◐ (research-complete, not wired)
Match the agency's phone numbers across **Dialpad ↔ Hiya ↔ Number Verifier**, register them, and track spam labels so outbound calls aren't flagged. Today this runs through a **Chrome extension** (`cnam-extension`) bridge; the Rust workers (`number-matcher`, `csv-processor`) exist standalone but are **not wired** to the backend.
- **s6 research finding:** all three vendors expose **server APIs** (Dialpad self-serve; Hiya Number Reputation API + Number Verifier are sales-gated). The extension was a workaround, not a necessity — Track E1 becomes a clean server integration once vendor API credentials are obtained. See `module-specs/07-dialers-numbers.md`.

### 4.9 TxtGen 📝
Generate copy/scripts from a small form. Self-contained, no GHL dependency. **Not yet ported**; folds into the CMS authoring panel (Track E2).

### 4.10 Results ✅ (shared)
A shared progress/results surface (OK/Error counts + progress bar) that bulk operations report into. Currently inline; a discrete component + a backend run-status read model is planned.

---

## 5. API reference (current backend endpoints)

All under `http://localhost:8091`. Tokens are resolved server-side from the vault; the browser never sends them.

| Method | Path | Version | Purpose |
|--------|------|---------|---------|
| POST | `/api/connect` | 2021-07-28 | Agency PIT → list + store sub-accounts |
| POST | `/api/connect/location` | 2021-04-15 | Location PIT → validate + store one sub-account |
| POST | `/api/tokens/{locationId}` | — | Save a sub-account token (server-only) |
| GET | `/api/accounts` | — | List accounts (vault) |
| GET | `/api/accounts/library` | — | Library (metadata + `hasToken`) |
| PUT | `/api/accounts/{locationId}/meta` | — | Update per-location metadata |
| GET | `/api/cv?locationIds=a,b,c` | 2021-04-15 | List custom values (parallel fan-out) |
| GET | `/api/cv/npl-scan?locationIds=&key=` | 2021-04-15 | New Patient Link audit |
| POST | `/api/cv/bulk` | 2021-04-15 | Bulk-update custom values (partial success) |
| GET | `/api/cv/{loc}/{cvId}` | 2021-04-15 | Get one custom value |
| DELETE | `/api/cv/{loc}/{cvId}` | 2021-04-15 | Delete one custom value |
| GET | `/api/customfields/{loc}` | 2021-04-15 | Custom-field schema |
| GET | `/api/objects/{loc}` | 2023-02-21 | Object schema (business/opportunity/contact + custom) |
| GET | `/api/workflows/{loc}` | 2021-07-28 | List workflows (read-only) |
| GET | `/api/funnels/{loc}` | 2021-07-28 | List funnels |
| GET | `/api/funnels/{loc}/audit` | 2021-07-28 | Live-page pixel/UTM audit |
| PUT | `/api/funnels/page/{pageId}` | — | ⛔ `not_yet_implemented` (GHL write impossible) |
| GET | `/api/dashboard/{loc}/contacts` | 2021-07-28 | Paginated contacts → leads analytics |
| POST | `/api/numbers/library` | — | Dialer numbers library (stub) |
| * | `/api/ghl/*` | per-path | Generic GHL passthrough (token + correct Version injected) |

---

## 6. The GHL API ceiling (read this before promising a feature)

GHL's public API v2 (PIT or OAuth) is **read-rich, write-poor.** The honest ceiling:
- ✅ **Read** almost everything (locations, custom values/fields, contacts, workflows list, funnels/pages, objects).
- ✅ **Write:** custom values only (`PUT customValues`). Plus our own vault (tokens/metadata) and contact↔workflow enrollment.
- ⛔ **No write** for funnel pages (no endpoint, any auth) or workflow definitions (list-only). Those were only writable via the internal browser-session token — out of scope.

**Version headers are per-endpoint** (centralized in `store.GHLVersionFor`):
- `/objects/*` → **`2023-02-21`** (required — rejects the legacy value)
- `/customValues`, `/customFields` → **`2021-04-15`**
- everything else (locations, workflows, contacts, funnels, forms) → **`2021-07-28`**

See `docs/api-requests/` and `docs/ghl-api-v2-auth-model.md` for verified request shapes.

---

## 7. Security model

- **Tokens server-side only (ADR-001).** Browser ↔ backend carries `hasToken: bool`, never `pit-…`. Generalizes to *all* integration secrets in the future.
- **SSRF guard.** Every outbound request is host-locked to the hardcoded GHL base before `client.Do`.
- **Auth middleware** is a no-op stub today; Phase 2 wires JWT (bcrypt, matching the sibling auth-server) once a DB exists.
- **Secrets at rest (future, ADR-004):** an env-key AES-256-GCM encrypted credential column — not a captured browser token, not plaintext.
- **Never commit** `*.key`, `*.cer`, `.env`, `*_secret*`, `*_token*`, or live `pit-…` values. (The `docs/"api requests.md"` token was redacted; rotate any exposed PIT.)

---

## 8. Future features — the CRM + CMS platform (designed, not built)

Accepted as a **reference architecture** in session 5 (`docs/architecture/platform-architecture-and-migration.md`). **Build gate (owner's rule):** each module's core-feature spec (`docs/module-specs/`) must flip `draft → agreed` before its track is built. Nothing below is in flight yet.

### Track A — GHL feature ports ◐
The modules in §4 — the **transition read surface** the migration consumes.

### Track B — Platform foundation 📝
- **B1** Neon Postgres + an embedded idempotent migration runner.
- **B2** **Encrypted credential store** (AES-GCM) replacing the volatile in-memory vault — *closes the #1 gap: state survives restart*, with zero handler churn.
- **B3** NATS JetStream (work queue + DLQ + KV in one).
- **B4** slog + OTel → Prometheus / Loki / Grafana (observability early).

### Track C — Integration hub 📝
One Go adapter interface every system implements; a per-(adapter, module) **kill switch** (`integration_switches` + `INTEGRATIONS_DISABLE` env). GHL folds in as a **removable** adapter. Webhook pipeline lands events on a durable stream (notifier consumes the *main* stream; DLQ feeds an alerter) with per-adapter circuit breakers.

### Track D — Hybrid CRM spine 📝
**Salesforce + Neon Postgres as dual systems-of-record** with an entity-ownership matrix and **bidirectional sync via transactional outbox/inbox** (no CDC). Salesforce sync = REST upsert-by-External-ID + SOQL `SystemModstamp` polling. GHL during transition = pull-only source filling Neon.

### Track E — Subsystems 📝
- **E1 Dialer** — wire the Rust workers via NATS; Dialpad/Hiya/Number-Verifier adapters (server APIs confirmed in s6; gated on vendor onboarding). Drops the Chrome extension.
- **E2 CMS + Cloudflare** — replaces GHL Sites & Funnels (funnel write is impossible). The CMS becomes the **system of record** for sites/funnels/pages and publishes to **Cloudflare Pages + DNS**; **build-time pixel injection** (no operator paste); trusted-blocks-only authoring with server-side HTML sanitization. Decisions in `docs/module-decisions/M4-sites-funnels-decisions.md`.
- **E3 Messaging / egress** — WhatsApp, Zapier, Squarespace adapters.

### Track F — Claude shell 📝
A standalone **MCP server** + a **CLI**, both over one shared `shellcore` that calls the backend HTTP API (so the kill switch, outbox, and auth gate the shell too; ADR-001 holds). ~21 tools across crm / sync / integrations / cms / dialer / observability / config.

### Track G — GHL retirement (strangler-fig) 📝
Per-module keep/kill via the kill switch, in spec order: Connect → Dialers → Accounts → Sites/Funnels(+TxtGen) → Dashboard → Custom Values (kept longest — only GHL-writable surface) → Automations → Results.

### Per-module future target (summary)

| Module | Today | Future home |
|--------|-------|-------------|
| Connect | agency/location PIT → vault | Own core (encrypted creds) |
| Accounts | vault library | Own core `organizations` (+ Salesforce mirror) |
| Custom Values | GHL read/write | Neon source-of-truth + GHL write-through (kept longest) |
| Schema | GHL read | Feeds the CRM core + objects/records module |
| Automations | GHL list | Own core + NATS rules off `events.>` |
| Dashboard | GHL contacts | Neon aggregates + Grafana for ops metrics |
| Sites & Funnels | GHL read + assisted pixel | **CMS on Cloudflare** (system of record) |
| Dialers | extension bridge | Dialpad/Hiya/NV adapters via NATS + Rust workers |
| TxtGen | (not ported) | CMS authoring panel (template or LLM) |
| Results | inline | Shared component + backend run-status read model |

---

## 9. Known limits & gotchas

- **No DB yet.** Backend restart wipes tokens + metadata; browser refresh resets the session. (Fixed by Track B2.)
- **GHL pagination is unreliable** — the `page` param can re-return a page; the dashboard guards with a per-page signature + `nextPageUrl` stop.
- **Funnel/page write is impossible** on the public API — use the assisted-manual flow until the CMS (E2) ships.
- **Workflows are list-only** — no edit/trigger via the public API.
- **`/objects` needs Version `2023-02-21`** — sending the legacy version fails. Handled centrally now.
- **Dialer vendor APIs are sales-gated** (Hiya Number Reputation, Number Verifier) — onboarding is a lead-time task before E1 builds.
- **customValues version moved 2021-07-28 → 2021-04-15** (s6) — **verified live** (102 CVs returned, no regression). One-constant rollback in `store/version.go` if ever needed.
- **GHL returns some numbers as floats** (e.g. customField `position` = `281.25`) — decode such fields as `float64`, not `int`, or the handler 502s. (Fixed for customFields in s6.)

---

## 10. Where to look

| Topic | File |
|-------|------|
| Milestone status (authoritative) | `roadmap/_roadmap.md` (vault) |
| Per-module core-feature specs (build gate) | `docs/module-specs/` |
| Module decisions (M3, M4) | `docs/module-decisions/` (vault) |
| Platform architecture + migration | `docs/architecture/platform-architecture-and-migration.md` |
| GHL API surface + versions | `docs/ghl-api-surface.md`, `docs/api-requests/`, `docs/ghl-api-v2-auth-model.md` |
| What GHL's API can/can't do | `docs/module-capabilities-and-limits.md` |
| Data models (TS shapes) | `docs/data-models.md` |
| Chrome extension bridge protocol | `docs/extension-bridge.md` |
| Pre-ship security checklist | `docs/security-audit.md` |

---

## 11. Verification log

**2026-06-08 (s6) — live-verified against a connected account (`6KtnVX…`):**
- `GET /api/customfields/{loc}` → **67 fields** ✅ — fixed a 502 (GHL returns `position` as a float like `281.25`; the struct now uses `float64`).
- `GET /api/objects/{loc}` → **3 objects** (business / opportunity / contact) ✅ — confirms `Version 2023-02-21`.
- `GET /api/cv?locationIds=` → **102 custom values** ✅ — confirms the `2021-04-15` version change is regression-free.
- `GET /api/cv/{loc}/{cvId}` (single) ✅.
- **Per-account PIT confirmed** — each account's calls use that account's connected PIT (vault keyed by `locationId`; typed handlers 401 if the account isn't connected).

**Build/CI (s6):** `go build` + `go vet` + `gosec` clean · `svelte-check` 288/0/0 · server boots with no route conflicts.

---
_This manual is generated/maintained by hand alongside the code. When a feature ships or a track starts, update its status marker here and in `roadmap/_roadmap.md`._
