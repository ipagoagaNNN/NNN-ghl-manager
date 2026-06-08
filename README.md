# NNN-GHL-manager

**Multi-sub-account management dashboard for a marketing agency running on GoHighLevel (GHL)** — a Svelte 5 SPA + Go backend proxy + Rust workers, engineered so GHL tokens never reach the browser.

> **Status:** active development. The core agency modules are live (Connect, Accounts, Custom Values, Schema, Dashboard, Automations, Sites & Funnels audit). The longer-term direction is a standalone **CRM + CMS platform** that absorbs GHL's data and then retires GHL adapter-by-adapter.

---

## What it is

NNN-GHL-manager began as a refactor of a 12,600-line HTML prototype into a maintainable stack. It lets an agency manage many GHL sub-accounts from one place: bulk-edit custom values, audit funnels for the Meta Pixel, view leads analytics, manage phone-number spam reputation, and more.

**Core invariant:** GHL Private Integration Tokens (`pit-…`) are held **server-side only**. The browser calls `/api/*` on the Go backend, which talks to GHL — tokens never cross to the client (responses carry `hasToken: bool`, never the value).

## Architecture

```
Svelte 5 SPA (frontend/, :5173)
  connect · accounts · custom-values · schema · funnels · automations · dashboard · dialers · help
      │  /api/*   (tokens never cross this line)
      ▼
Go backend (backend/, :8091, standard library only)
  middleware (CORS · rate-limit · auth-stub) · per-module handlers · GHL proxy · in-memory token vault
      │  Bearer pit-… + per-endpoint Version header
      ▼
GoHighLevel API v2
Rust workers (workers/): number-matcher · csv-processor   (standalone; wired in a future track)
```

- **Frontend** — SvelteKit + Svelte 5 runes, dependency-light SPA.
- **Backend** — Go 1.23, **stdlib only** (no web framework). Entry: `backend/cmd/server/main.go`.
- **GHL Version map** — the correct `Version` header is resolved per endpoint in `store.GHLVersionFor` (objects `2023-02-21`, customValues/customFields `2021-04-15`, everything else `2021-07-28`).
- **No database yet** — tokens + metadata live in an in-memory vault (cleared on restart). A persistent, encrypted store is the first platform-foundation milestone.

## Quick start

```bash
# Backend API (:8091 — 8080/8090/9090 are reserved by Docker on the dev machine)
cd backend && go run ./cmd/server

# Frontend SPA (:5173, proxies /api → :8091)
cd frontend && npm install && npm run dev
```

Open **http://localhost:5173** and start at **Connect** (paste a sub-account PIT). The in-app **Help** page (`/help`) renders the full manual.

Verify a change: `cd backend && go build ./... && go vet ./...` · `cd frontend && npm run check`.

## Features

| Module | Status | What it does |
|---|---|---|
| **Connect** | ✅ | Agency or location PIT → validate + store server-side; resolves a friendly account name from the `Location Name` custom value |
| **Accounts** | ✅ | Per-account library (token presence, domain/metadata), CSV import/export, select-to-scope |
| **Custom Values** | ✅ | Bulk view/edit across accounts, brand presets, **New-Patient-Link scanner** (booking-link audit vs the registered domain), delete |
| **Schema** | ✅ | Per-account objects + custom-field schema viewer |
| **Automations** | ✅ | Read-only workflow list (GHL's Workflows API is list-only) |
| **Dashboard** | ✅ | Leads-by-day, top sources, per-account analytics with date filtering |
| **Sites & Funnels** | ◐ | Funnel list + live-page **pixel/UTM audit** + duplicate-pixel detection + assisted-manual fix with a per-funnel **Verify** loop |
| **Dialers** | 📝 | Phone-number spam reputation (Dialpad / Hiya / Number-Verifier) — research complete; server integration pending |
| **TxtGen · Results** | 📝 | Copy generation · shared progress surface |

Full feature + API reference: **[`docs/MANUAL.md`](docs/MANUAL.md)**.

## The GHL API ceiling

GHL's public API v2 (PIT or OAuth) is **read-rich, write-poor**:

- ✅ Read almost everything; **write** only custom values (`PUT customValues`).
- ⛔ **No** funnel/page write (no endpoint, for any auth type) and workflows are list-only.

Funnel pixel injection is therefore **assisted-manual** today — the app audits the live page, flags missing/wrong/duplicate pixels, hands you the exact snippet + a deep link to the funnel, and a **Verify** button re-checks after you publish. The permanent fix is the CMS track (own the pages, inject the pixel at build time).

## Milestones

| Phase | Title | Status |
|---|---|---|
| 0 | Reference docs + scaffold (Go / Rust / Svelte 5) | ✅ |
| 1a–1c | Backend + frontend build, dev servers | ✅ |
| 1d | Connect flow end-to-end | ◐ |
| 1e | Pre-ship security checklist | ✅ |
| 2a | Accounts | ✅ |
| 2b | Automations | ✅ |
| 2c / 2c+ | Custom Values (core ✅ · NPL scanner ✅ · inner tabs pending) | ◐ |
| 2d | Dashboard | ✅ |
| 2e | Sites & Funnels (read + audit + assisted pixel ✅ · CMS replacement future) | ◐ |
| 2f | Dialers — Numbers + Flagged | 📝 |
| 2g | TxtGen | 📝 |
| 3 | Auth — JWT + sessions | 📝 |
| 4 | Enterprise hardening (RBAC, audit log, MFA, retention) | 📝 |

**Legend:** ✅ shipped · ◐ partial (core done) · 🚧 in progress · 📝 designed/researched · ⛔ blocked

Also shipped recently: the **GHL API request layer** (per-endpoint Version headers + CV get/delete, customFields, objects endpoints), the **Schema** viewer, and an **in-app Help** page.

## Roadmap — the CRM + CMS platform

The modules above are the GHL **read surface**. The longer-term design evolves this into a standalone CRM + CMS that absorbs GHL's data and retires GHL adapter-by-adapter. These tracks are **designed, not yet scheduled** — each module's spec must be agreed before its track is built.

| Track | Direction |
|---|---|
| **B — Foundation** | Postgres (Neon) + embedded migrations · **encrypted credential store** (state survives restart) · NATS JetStream · observability (OTel / Prometheus / Loki / Grafana) |
| **C — Integration hub** | One adapter interface + a per-(adapter, module) kill switch · GHL folds in as a *removable* adapter · webhook pipeline + circuit breakers |
| **D — Hybrid CRM spine** | Salesforce + Postgres as dual systems-of-record · bidirectional sync via a transactional outbox/inbox |
| **E — Subsystems** | Dialer (Dialpad / Hiya / Number-Verifier via the Rust workers) · **CMS on Cloudflare** (replaces GHL Sites & Funnels; build-time pixel injection) · messaging/egress (WhatsApp, Zapier, Squarespace) |
| **F — Claude shell** | A standalone MCP server + CLI over the backend API (not linked to any external system) |
| **G — GHL retirement** | Per-module keep/kill via the kill switch, in spec order |

Full design: `docs/architecture/platform-architecture-and-migration.md`; per-module specs: `docs/module-specs/`.

## Security

- **Tokens server-side only** — the browser never receives a `pit-…` value.
- **SSRF guard** on every outbound call and live-page fetch (host-locked to GHL / validated-public hosts).
- `gosec`-clean backend; the auth middleware (JWT) activates with the Auth milestone.
- Never commit `*.key`, `*.cer`, `.env`, or token values.

## Repository layout

```
backend/    Go API (cmd/server, internal/{handlers,proxy,store,middleware})
frontend/   SvelteKit SPA (src/routes, src/lib)
workers/    Rust workers (number-matcher, csv-processor)
docs/       MANUAL.md + GHL API surface, module specs, architecture
```

## Documentation

- **[`docs/MANUAL.md`](docs/MANUAL.md)** — full feature + API reference (also served in-app at `/help`)
- `docs/ghl-api-surface.md` · `docs/api-requests/` · `docs/ghl-api-v2-auth-model.md` — GHL API details
- `docs/module-capabilities-and-limits.md` — what GHL's API can and cannot do
- `docs/architecture/` · `docs/module-specs/` — platform architecture + per-module specs

---

_Active development — internal agency tool. Not yet released._
