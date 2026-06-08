# NNN-GHL-manager → CRM + CMS Platform: Architecture Verification, Migration & Claude Shell

> **Status — accepted as a reference architecture / starting point (2026-05-29). NOT scheduled for implementation.**
> Owner direction: keep this as a durable design doc; write **no code** and make **no roadmap/file edits** yet.
> **Build prerequisite (owner's gate):** the *core feature set of each GHL module* must be specified before any
> track below is implemented. Standing up the platform scaffolding ahead of those per-module specs would force a
> major refactor of every module. Read the tracks as the target *shape*; each is gated behind a per-module
> core-feature spec. The natural next document is a per-module feature spec set, not code.
>
> _Source: planning session 2026-05-29 (3 exploration agents + 3 design agents). Mirrored from the session plan file._

## Context

**Why this exists.** NNN-GHL-manager began as a refactor of a 12,600-line HTML GoHighLevel (GHL) prototype into Svelte 5 + Go + Rust. The owner now wants it to become **part of a larger system**: a hybrid CRM + CMS that integrates Salesforce, WhatsApp, Squarespace, Dialpad, Zapier, Hiya, Meta Ads, and Cloudflare — and to **migrate off GHL step by step**. An architecture diagram (4 phases, ~40 components) was drafted. The owner asked to (1) verify the diagram, (2) produce a complete target architecture, (3) define a step-by-step GHL migration, (4) define the integration architecture, (5) design a "shell for Claude to interact with the system," and (6) modify the roadmap.

**What prompted it.** The current roadmap is "port the 10 GHL modules" (Phases 0–2g) + Auth + hardening. That framing doesn't express the real goal — a platform that absorbs GHL's data and then retires it. The diagram is the first attempt to express the bigger system; it needs a reality check before it drives a roadmap.

**Intended outcome.** A corrected target architecture, a strangler-fig migration sequence, an integration hub with a per-module GHL kill switch, a Claude shell (MCP + CLI), and a rewritten roadmap — all grounded in the actual codebase, lean enough for a solo dev to operate.

**This plan was produced from:** 3 parallel codebase/doc explorations + 3 parallel design agents (data/sync, integration/migration/CMS, shell). The diagram itself was verified visually against those findings.

---

## Locked decisions (from the owner, this session)

| # | Decision | Consequence |
|---|---|---|
| D1 | **GHL = removable adapter**, decide keep/kill per-module later | GHL folds in as one adapter behind a per-(adapter,module) kill switch; never a permanent peer |
| D2 | **CRM of record = HYBRID Salesforce + Neon Postgres** | Need an entity-ownership matrix + bidirectional sync; Salesforce wins sales objects, Neon wins operational/CMS/dialer |
| D3 | **Infra = lean + observability early** | Stand up Neon + NATS JetStream + slog + OTel/Prometheus/Loki/Grafana. **Defer** Meilisearch, MinIO (reuse Cloudflare R2), CDC/Debezium, read replica, HashiVault |
| D4 | **Claude shell = BOTH** MCP + CLI over one shared core | Build a shared Go core; two thin front-ends |
| D5 | **Cloudflare = domain management** (DNS + hosting) | Cloudflare becomes the runtime for the CMS that replaces GHL Sites & Funnels |

---

## Part 1 — Diagram verification (the explicit ask)

**Headline:** the diagram is ~90% aspirational. Almost nothing in Phases 2–4 exists; two "Phase-1/done"-colored boxes don't exist either. It's a fine north star, but the phase colors don't encode build-order or dependency reality.

### Box-by-box verdict

| Diagram box (phase color) | Reality in repo | Verdict |
|---|---|---|
| frontend `Next\|Svelte` (P1) | Svelte 5 / SvelteKit only; **no Next.js anywhere** | ✏️ Rename "Svelte 5". Drop Next. |
| backend `Rust+Go` (P1) | Go (stdlib) API + **two standalone Rust workers not wired in** | ✏️ Split: Go orchestrator + Rust jobs behind a queue |
| `DB Postgres` + `/migrations` (P1) | **Do not exist.** Token vault is an in-memory map (`store/vault.go`), lost on restart | 🔴 #1 gap. A CRM with no DB. Build first |
| NATS + MQueues/Pub-Sub + NATS DLQ (P3) | Absent | ⚠️ Triplicated — **one** NATS JetStream gives pub/sub + work queue + DLQ |
| Valkey/redis (P3) | Absent; 60s in-handler caches exist | ⏸️ Defer; JetStream KV / Postgres cover early needs |
| Meilisearch, MinIO, CDC, replica DB (P3) | Absent | ⏸️ Defer all. Postgres FTS + `LISTEN/NOTIFY`; **reuse Cloudflare R2** not MinIO |
| HashiVault (P2) | Absent | ⏸️ Defer. Encrypt secrets with env-key AES-GCM column now |
| OTel/Prometheus/Loki/Grafana + BI (P4) | Absent | ✅ Bring early per D3 (lean+observability) |
| AUTH + PASETO (P2) | `middleware/auth.go` is a **no-op stub** | ⚠️ Depends on DB; sequence after persistence. JWT (bcrypt) per sibling, not PASETO |
| Webhooks handler (P2) feeding integrations, queue at P3 | Absent | ⚠️ Webhooks must land on a durable queue/outbox **from day one** |
| API bridge + Data Sync Engine + Circuit Breaker (P2) | Absent | ⚠️ One subsystem; needs the queue (P3) it's drawn ahead of |
| `Notifications Worker` fed by `NATS DLQ` | Absent | 🔴 **Wiring bug** — notifications must consume the *main* event stream; DLQ is a failure/alert path |
| `Go High Level` as permanent blue integration | Exists (the whole app today) | 🔁 Rethink: model GHL as a **removable transition source** (D1) |
| CMS `Next\|svelte` → templating → TextGen/Canvas (P3) | Absent | 🔁 Good direction, wrong premise — see below |
| CMS ↔ GHL funnel writes (implied) | **Impossible** — GHL funnel-page write doesn't exist (`docs/module-capabilities-and-limits.md`) | 🔴 Dead end. CMS must be the *system of record* and publish to Cloudflare, not sync to GHL |
| Dialpad (P2) | Targeted by Rust `number-matcher` | ✏️ Missing **Hiya + Number-Verifier** boxes (the worker matches all three) |
| CMS → Cloudflare → customer | Absent | ✅ Correct path; this is the GHL-funnels replacement |

**Keep (genuinely good):** tokens-server-side invariant (ADR-001 — *generalize to all integration secrets*); frontend/backend/workers split; phase-coloring discipline; CMS→Cloudflare→customer publishing path.

---

## Part 2 — Target architecture (corrected, lean)

```
                         ┌──────────────── Svelte 5 SPA (frontend/) ────────────────┐
                         │  connect · accounts · custom-values · funnels(→CMS) ·      │
                         │  automations · dashboard · dialers · /cms (Canvas)         │
                         └───────────────────────────┬──────────────────────────────┘
                                                      │ /api/* (tokens never cross)
                         ┌────────────────────────────▼─────────────────────────────┐
   observability ◀───────│  Go backend (backend/) — module path …/backend, Go 1.23   │
   slog + OTel →         │  middleware: CORS · ratelimit · AUTH(JWT, was stub)        │
   Prom/Loki/Grafana     │  integrations/ (adapter hub + kill switch)                 │
                         │  sync/ (outbox→relay / webhook→inbox→reconciler)          │
                         │  cms/ (templating · canvas · publish · sanitize)          │
                         │  store/ (Neon pool + AES-GCM credential store)            │
                         └───┬───────────────┬───────────────┬──────────────┬────────┘
                             │               │               │              │
                      ┌──────▼─────┐  ┌──────▼──────┐  ┌─────▼──────┐ ┌─────▼──────┐
                      │ Neon       │  │ NATS         │  │ Cloudflare │ │ Rust workers│
                      │ Postgres   │  │ JetStream    │  │ Pages/DNS  │ │ via NATS    │
                      │ (CRM core) │  │ (+KV +DLQ)   │  │ (CMS host) │ │ matcher/csv │
                      └──────┬─────┘  └──────┬──────┘  └────────────┘ └────────────┘
                             │   outbox/inbox │ commands.*/events.*
                   ┌─────────▼────────────────▼──────────────────────────────────┐
                   │ Integration adapters (one interface, per-module kill switch) │
                   │ ghl(removable) · salesforce · whatsapp · squarespace ·       │
                   │ dialpad · hiya · number-verifier · meta · zapier · cloudflare│
                   └──────────────────────────────────────────────────────────────┘

   Claude shell:  cmd/ghl-mcp (MCP, stdio)  +  cmd/ghl (CLI)  →  internal/shellcore  →  backend /api/*
```

**Stack (lean):** Neon Postgres · NATS JetStream (queue+DLQ+KV in one) · slog + OTel→Prometheus/Loki/Grafana · Cloudflare (R2 blobs + Pages/DNS) · pgx + one NATS client are the only new Go deps; adapters stay `net/http`.

---

## Part 3 — Data layer & hybrid Salesforce↔Neon sync

**The #1 fix:** replace the volatile in-memory vault with **Neon Postgres + an AES-256-GCM encrypted credential store** so the app survives restart. Mirror the sibling `auth-server` conventions: **pgx/v5 + `//go:embed`-ed migrations run idempotently on startup** (the sibling does NOT use the golang-migrate library), secrets from env (`CRED_MASTER_KEY`, 32-byte base64), encryption in the Go **stdlib** (`crypto/aes`+`crypto/cipher`) → no new dep.

**Preserve the `store.Vault` method surface byte-for-byte** (`SetLocToken/LocToken/SetAgency/AgencyToken/CompanyID/SetLocMeta/LocMetaFor/AllLocMeta/GHLBase`); swap the backing fields (maps → `*pgxpool.Pool` + `*Cipher`). **Handlers and `proxy/client.go` do not change** — this is ADR-003's "swap the backing, keep the surface" made literal. `main.go` wiring change ≈ 6 lines.

### Schema (`backend/migrations/00X_*.up.sql/.down.sql`, schema `crm`)
- **Core:** `organizations` (agency→location tree; `ghl_location_id` UNIQUE = old vault key), `contacts`, `activities`, `pipelines`, `pipeline_stages`, `opportunities`, `conversations`, `messages`, `custom_fields`, `custom_values`, `dialer_numbers`.
- **Ubiquitous sync columns on every syncable row:** `external_ids JSONB` (GIN-indexed multi-system id map), `row_version BIGINT`, `updated_by TEXT` ('local'|'salesforce'|'ghl' — loop suppression), `etag TEXT` (remote version).
- **System tables (the heart):** `integration_credentials` (encrypted `bytea secret_ct`, `key_version`, UNIQUE(provider,scope,COALESCE(ref,''))), `sync_state` (per entity/system cursor+etag), `outbox_events` (BIGSERIAL order, `idempotency_key` UNIQUE, state machine pending→publishing→done/failed/dead, backoff), `inbox_events` (`dedup_key` UNIQUE drops duplicate webhooks), `sync_conflicts`, `integration_switches` (kill switch).

### Entity-ownership matrix (source of truth)
| Entity | Direction | Conflict policy |
|---|---|---|
| organizations/accounts | Neon→SF | Neon wins |
| contacts | **bidirectional, field-level** | SF wins {name,email,owner,lifecycle}; Neon wins {tags,source,custom-ops}; else newest etag |
| opportunities, pipelines, stages | SF→Neon | **Salesforce always wins** |
| conversations, messages, activities | Neon→SF (as Activity, optional) | Neon wins |
| dialer_numbers, automations, funnels/CMS | Neon only | Neon wins |
| integration_credentials, sync_* | Neon only, never leaves server | n/a |

**GHL during transition = pull-only source:** GHL adapter reads → lands via **inbox** with `updated_by='ghl'`, stamping `external_ids.ghl`. GHL fills Neon; Neon (not GHL) pushes the sales subset to Salesforce. A GHL-origin row is never re-pushed to GHL unless a module is explicitly kept.

### Sync engine (`backend/internal/sync/`)
- **Outbound:** domain write + `outbox_events` insert in **one tx** (the guarantee that removes the need for CDC) → relay goroutine (`FOR UPDATE SKIP LOCKED`) publishes to `sync.out.<target>.<aggregate>` with JetStream `Msg-Id` dedup → adapter consumer calls remote.
- **Salesforce mechanism (v1, justified):** **REST upsert by External ID** (natively idempotent, keyed on our UUID; composite collections ≤200/call) + **SOQL `SystemModstamp` polling** (1–5 min cursor in `sync_state`). **Defer** Pub/Sub API + CDC (gRPC+HTTP/2+Avro shatters the low-dep ethos; one-way only); v2 trigger = poll latency or governor pressure. Auth = OAuth2 JWT bearer, token in `integration_credentials`.
- **Inbound:** webhook → `inbox_events` (`ON CONFLICT DO NOTHING`) → 200 immediately → reconciler upserts Neon, detects conflicts (both sides changed since `last_synced_*`), applies winner with `updated_by=source` (no ping-pong).
- **Retry/DLQ:** exponential backoff in the row; after ~8 attempts → DLQ + `dead`.

---

## Part 4 — Integration hub, webhooks, migration

### Adapter abstraction (`backend/internal/integrations/`)
One Go interface every system implements: `Name()`, `Capabilities(module)` (CanRead/CanWrite/HasWebhook/CanEgress + `WriteNote`), `RateLimit()`, **`Enabled(ctx,module)` kill switch**, `Do(ctx,scope,req)` (resolves creds *internally*, ADR-001), `VerifyWebhook(r,body)→Event`, `HealthCheck()`. Central enforcement in `Registry.Dispatch` — one check guards every call; non-GET on `CanWrite=false` → `write_unsupported`.

**The existing GHL code folds in, not rewritten:**
- `proxy/client.go` host-lock + 429 retry → `integrations/httpbase/client.go` (generic) + `integrations/ghl/proxy.go` (sets `Version: 2021-07-28`).
- `guardPublicHost`/`isPublicIP` (handler-private in `funnels.go`) → shared `integrations/httpbase/ssrf.go`.
- `store.Vault` → wrapped by `credentials/` then by the encrypted store.
- handlers (`funnels/customvalues/…`) → `integrations/ghl/*.go`, constructor takes `*ghl.Adapter`; **routes in `main.go` unchanged**.
- `pixels.go expectedPixels` → `integrations/meta/` (Meta Ads = pixel system of record).

**Kill switch:** `integration_switches` table + `INTEGRATIONS_DISABLE="ghl:funnels,…"` env break-glass; hot reads via JetStream KV `KILLSWITCH` bucket. Flip a row → GHL (or one module) stops dispatching with zero code change. **This is what makes GHL removable (D1).**

### Webhook pipeline (fixes the diagram bug)
`POST /api/webhooks/{adapter}` → `VerifyWebhook` (HMAC/sig per adapter) → durable land on JetStream `events.<adapter>.<entity>.<action>` → **three independent durable consumers**: `reconciler` (→Neon), **`notifier` (consumes the MAIN stream — "lead.created → notify")**, `fanout` (→ outbound adapters, each behind a per-adapter **circuit breaker**). **DLQ `events.dlq.>` feeds `dlq_alerter` only** (ops alert), never customer notifications. Subjects: `events.*`, `commands.*` (outbound intents), `events.dlq.*`.

### Strangler-fig migration (per GHL module)
| GHL module | Target home | Kill-switch trigger | Order |
|---|---|---|---|
| Connect | Own core (encrypted creds) | creds in Postgres store | 0 |
| Dialers Numbers/Flagged | Dialer subsystem (Rust via NATS + Dialpad/Hiya/NV; R2 for CSV) | match+flag run server-side, results in Neon | 2 |
| Accounts | Own core `organizations` (+SF mirror) | account list from Neon | 3 |
| Sites & Funnels + TxtGen | **CMS + Cloudflare** (write-first; GHL was never writable) | funnel authored in CMS, published, DNS cut over, pixel verified | 4 |
| Dashboard | Own core (Neon reads) + Grafana | dashboard reads Neon aggregates | 5 |
| Custom Values | Own core (kept longest — only GHL-writable thing) | Neon is CV source of truth | 6 |
| Automations/Workflows | Own core + NATS rules | automations run off `events.>` | 7 |
| Results | Own core + Dashboard | reads Neon aggregates | 8 |

**Integration onboarding waves:** W0 adapter hub+NATS+GHL fold-in → W1 Salesforce+Neon spine → W2 Dialpad/Hiya/Number-Verifier (wire orphaned Rust workers via NATS; **adds the Hiya/NV boxes the diagram omitted**) → W3 Cloudflare+Meta (CMS) → W4 WhatsApp+Zapier (egress) → W5 Squarespace.

**Wire the orphaned Rust workers:** replace `axum::serve` with an `async-nats` JetStream subscribe loop; reuse `match_numbers`/`process_csv` verbatim. `number-matcher` consumes `commands.dialer.match` / emits `events.dialer.match.completed`; `csv-processor` consumes `commands.dialer.csv.process` (CSV ref in R2). Kills the Chrome-extension dependency.

---

## Part 5 — CMS-on-Cloudflare (replaces GHL Sites & Funnels)

Because GHL funnel-page write is impossible, the CMS becomes the **system of record** for sites/funnels/pages and publishes to Cloudflare. `backend/internal/cms/`: `store.go` (Neon: `sites/funnels/pages/page_versions/pixels/domains/publishes`), `templating/` (stdlib `html/template` + block registry), `canvas/` (JSON doc model, Svelte `/cms` editor), `textgen/`, `publish/` (builder → `cloudflare` adapter: Pages Direct Upload + custom hostnames + DNS), `sanitize/`.

**Pixel injection becomes first-class & safe:** authors pick a brand; `builder.go` injects `fbq('init',…)` at **build time** (no operator paste, no captured token). The existing `AuditFunnels`/`auditHTML` **inverts** into a post-publish verification probe (reuses the same detectors + SSRF-guarded fetch).

**Custom HTML/JS sandbox (the top residual risk):** trusted parameterized blocks need no sanitization; raw custom HTML is role-gated, sanitized server-side at save (`page_versions.sanitized` gate — builder refuses unsanitized), strict CSP + per-deploy nonces at the Cloudflare edge, and the Canvas preview renders in a cross-origin `<iframe sandbox>`. The prototype's browser-session-token injection path is **never built**.

---

## Part 6 — Claude shell (MCP + CLI over one core)

**Shared core `backend/internal/shellcore/`** — calls the backend **HTTP API** (not Neon/NATS directly). Rationale: kill-switch + outbox + auth already live behind the API (a disabled adapter stays disabled for Claude too); ADR-001 holds (shell carries at most a *service JWT*, never an integration token); business logic isn't duplicated. A `WithReadDB(dsn)` **read-only** seam is designed-in but deferred (trigger: a tool needs >5k rows / multi-table aggregate).

**MCP server `cmd/ghl-mcp/main.go`** — hand-rolled JSON-RPC over stdio (~170-line loop); typed `ToolDef` **+ a `Category` field**; `internal/mcp/tools/{toollist,dispatch,core}.go`; walk-up `.ghl_config.json`; **`configure` tool day one** storing the token *env-var name*, never the value; a schema-driven vault cache. Keep `main.go` thin — the dispatch logic lives in `internal/mcp/`.

**Tool surface (21 tools / 7 categories):** crm (`crm_contacts_query/contact_get/contact_cache/opportunities_query/pipelines_list/conversations_query`), sync (`sync_run/state/dlq_depth/dlq_peek/replay_event`), integrations (`integrations_list/integration_health/integration_toggle`), cms (`cms_pages_list/page_get/publish`), dialer (`dialer_number_match`), observability (`system_status/recent_errors`), config (`configure/ghl_help`). Each maps 1:1 to a `shellcore` method **and** a CLI subcommand.

**CLI `cmd/ghl/main.go`** — stdlib `flag` + tiny subcommand switch (zero deps; revisit cobra >40 cmds). `noun verb`: `ghl contacts list --location L --limit 50`, `ghl sync run --entity contacts`, `ghl integration disable ghl --module funnels`, `ghl cms publish --site X`, `ghl dlq peek`. Human `tabwriter` tables by default, `--json` = the same struct the MCP serializes. Token from env only (never argv/flags/output); all output through `redact.Scrub()`.

**Backend API contract the shell needs** (new endpoints to add alongside existing `/api/accounts`, `/api/cv`, `/api/funnels/*`, `/api/ghl/*`): `/api/crm/{loc}/{contacts,opportunities,pipelines,conversations}`, `/api/sync/{run,state,dlq,dlq/{id}/replay}`, `/api/integrations[/{name}/health|/killswitch]`, `/api/cms/{site}/{pages,publish}`, `/api/dialer/match`, `/api/health/{status,errors}`. `shellcore/types.go` is the contract — same module as handlers, so drift fails `go build`.

**Build & register:**
```powershell
cd C:\Users\get_h\Documents\VSCode\NNN-GHLmanager\backend
go build -o bin\ghl-mcp.exe .\cmd\ghl-mcp
go build -o bin\ghl.exe     .\cmd\ghl
go vet .\...
```
`//go:embed` the schema files into `ghl-mcp.exe`. Add to `C:\Users\get_h\AppData\Roaming\Claude\claude_desktop_config.json` under `mcpServers`: `"ghl-mcp": { "command": "…\\backend\\bin\\ghl-mcp.exe", "args": [], "env": { "GHL_SHELL_TOKEN": "…" } }`. Full Claude Desktop restart to pick it up.
---

## Part 7 — Roadmap modification (primary deliverable)

**File:** `C:\Users\get_h\Documents\Obsidian\Obsidian-v1\8 - Claude Work\Projects\NNN-GHL-manager\roadmap\_roadmap.md` (canonical; single table + sections, legend `✅ ◐ 🚧 🆕 📝 ⛔`).

**Approach — reframe from "GHL module ports" to "platform tracks," preserving completed work.** Keep the existing Phase 0–2 rows as **Track A** (they provide the GHL *read surface* the migration consumes). Replace the planned Phase 3/4 with platform tracks B–F. Concretely, the edit will:

1. **Keep** the Phase 0–2g rows under a new `### Track A — GHL feature ports (transition read surface)` heading; flip **2e** stale `📝`→`◐` (funnels contract + verified path shipped in s4) and **1d** to `◐` pending the DevTools token audit.
2. **Add** these tracks/phases (status `📝`), each a row block:
   - **Track B — Platform foundation:** B1 Neon + embedded-migration runner; **B2 encrypted credential store (closes the #1 gap — survives restart)**; B3 NATS JetStream (stream+DLQ+KV); B4 slog + OTel/Prometheus/Loki/Grafana.
   - **Track C — Integration hub:** C1 adapter interface + `httpbase` + SSRF lift; C2 kill switch (`integration_switches` + env break-glass); C3 GHL fold-in; C4 webhook pipeline (notifier-on-main-stream, DLQ→alerter) + circuit breaker.
   - **Track D — Hybrid CRM spine:** D1 core schema + ubiquitous sync columns; D2 transactional outbox/inbox + reconciler; D3 Salesforce adapter (REST upsert + SOQL poll); D4 conflict surface + DLQ viewer.
   - **Track E — Subsystems:** E1 Dialer (wire Rust workers via NATS; Dialpad/Hiya/Number-Verifier); E2 CMS + Cloudflare (replace Sites & Funnels; build-time pixel; sanitization); E3 messaging/egress (WhatsApp, Zapier, Squarespace).
   - **Track F — Claude shell:** F1 `shellcore` + API contract; F2 MCP server; F3 CLI. (Can start in parallel after B1/C1.)
   - **Track G — GHL retirement (strangler-fig):** per-module keep/kill using the Part 4 table.
3. **Rewrite `## Current focus`** to: "Track B foundation (persistence + encrypted creds) — closes the restart-data-loss gap." Move "Auth JWT" into B (it depends on the DB). **Update `## Blocked by`** (none new). **Add to `## See also`** the link to `module-capabilities-and-limits.md` and to this plan.
4. **Backlog → done:** mark "Write ADR files 001/002/003" as superseded by Part 8 (we write them on execution).

*(The exact new table text will be produced at execution and shown in the diff before saving.)*

---

## Part 8 — ADRs to write (`Projects/NNN-GHL-manager/decisions/`)

Currently empty. On execution, write: **ADR-001** Go proxy / tokens server-side (existing decision, formalized + **generalized to all integration secrets**); **ADR-002** Svelte 5 `$state` (existing); **ADR-003** auth-stub→swap (existing); **ADR-004** secrets-at-rest = env-key AES-GCM column (not KMS/Vault v1; `key_version` rotation seam); **ADR-005** hybrid Salesforce+Neon ownership + conflict policy; **ADR-006** GHL = removable adapter + per-module kill switch; **ADR-007** sync = transactional outbox/inbox + JetStream (no CDC); **ADR-008** Claude shell calls backend API (not direct DB); **ADR-009** CMS = system of record, publish to Cloudflare (no GHL funnel write).

---

## Execution order (DEFERRED — gated on per-module core-feature specs)

> Nothing in this section runs now. It is the *future* sequence, unlocked only after each module's core
> features are specified (the owner's gate above). Recorded here so the starting point is complete.

**First (when unblocked):** update `_roadmap.md` (Part 7) + write ADRs (Part 8). No code yet.

**Then, recommended Slice 1 (one session, highest value):** B1+B2 — Neon pool + embedded idempotent migration runner (copy `auth-server/main.go` pattern) + `store/cipher.go` (AES-GCM) + refactor `store/vault.go` to Postgres-backed **with zero handler churn**. *This alone makes the app survive restart.* Migration `005` (`integration_credentials`) + `001` (`organizations`, backfill from current `locMeta`).

**Subsequent slices** follow the track order: C1–C3 (adapter hub + GHL fold-in) → B3 (NATS) + D1/D2 (schema + outbox) → D3 (Salesforce) → E1 (dialer) → F1–F3 (shell, parallelizable) → E2 (CMS) → E3 → G (retire GHL per module). Dependency rule: credentials before sync; outbox before adapters; GHL-in before SF-out; observability (B4) alongside B3.

---

## Verification

- **Each Go slice:** `cd backend; go build ./...; go vet ./...` (and the `/nnn-build` skill pattern — `go build`+`go vet`, report binary size). Slice 1 success = restart the server, confirm previously-saved location tokens still resolve (`proxy/client.go` token read works) — proves persistence.
- **Migrations:** run twice; idempotent (`IF NOT EXISTS`) → second run is a no-op. Confirm a dedicated DML role can't run DDL.
- **Sync engine:** integration test — write a contact in Neon → assert one `outbox_events` row in the same tx → relay marks `done` → Salesforce upsert returns the External-ID record; then simulate an inbound webhook with a stale `etag` → assert it's dropped (no ping-pong) and a genuine conflict writes a `sync_conflicts` row.
- **Kill switch:** set `INTEGRATIONS_DISABLE=ghl:funnels` → `GET /api/funnels/*` returns `disabled`; clear it → works again.
- **Webhook pipeline:** post a signed test webhook → 202 fast → assert `notifier` fired off the **main** stream (not DLQ); force N failures → assert DLQ + `dlq_alerter`, notifications unaffected.
- **CMS:** author a page → publish (dry-run then real) → Cloudflare deployment URL returns 200 → post-publish audit probe confirms the built-in pixel; unsanitized custom HTML → publish refused.
- **Claude shell:** `go build ./cmd/ghl-mcp ./cmd/ghl`; `ghl configure ping` → backend reachable; `ghl integration list` → table; register `ghl-mcp.exe` in `claude_desktop_config.json`, restart Desktop, confirm `tools/list` shows 21 tools grouped by category and `configure` works in degraded (no-config) mode; verify no token-shaped string ever appears in any tool/CLI output (`redact.Scrub`).

---

## Open decisions deferred to implementation (not blockers)

- **Migration tooling:** mirror the sibling's embedded idempotent `Exec` runner (recommended) vs golang-migrate CLI — pick **one** before first deploy; do not run both against one DB.
- **Password hashing** (when Auth/JWT lands): sibling uses **bcrypt cost-12**, not Argon2id — match the sibling unless we standardize otherwise. (Orthogonal to credential AES-GCM encryption.)
- **Direct read-only DB for the shell** — build the `WithReadDB` seam only when a reporting tool needs it.
- **Salesforce Pub/Sub+CDC (v2)** — adopt only if poll latency/governor limits bite.
- **NATS HA** — single node until volume justifies (Postgres outbox/inbox is the durable truth regardless).

## Critical files
- `backend/internal/store/vault.go` — refactor in place, **keep method surface** (Slice 1).
- `backend/cmd/server/main.go` — ~6-line wiring (pool, migrations, cipher, sync engine); existing routes unchanged.
- `backend/internal/proxy/client.go` — hot-path token read (stays unchanged) + the host-lock/429 core lifted into `httpbase`.
- `backend/internal/handlers/funnels.go` — SSRF guard to lift; `AuditFunnels` becomes the CMS post-publish verifier.
- `…/NNN-HF-Obsidian/auth-server/main.go` + `migrations/008_attachments.sql` — pgx pool + embedded-migration runner + state-machine/partial-index templates to copy.
- _(Claude-shell scaffolding is self-contained in `cmd/ghl-mcp` + `internal/mcp`; no external code dependencies.)_
- `roadmap/_roadmap.md` + `decisions/` — the roadmap rewrite (Part 7) and ADRs (Part 8).
