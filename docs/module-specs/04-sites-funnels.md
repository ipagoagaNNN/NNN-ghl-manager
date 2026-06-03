# Module 4 — Sites & Funnels (core-feature spec)

- **Status:** draft (gate for Track E2 — CMS + Cloudflare). **Highest-risk module.**
- **Target home:** **CMS + Cloudflare** (system of record for pages; GHL is read-only/audit only)
- **Current state in repo:** `handlers/funnels.go` (`GET /api/funnels/:loc`, `GET /api/funnels/:loc/audit`), `handlers/pixels.go` (expected-pixel lookup), `handlers/stub.go` (`UpdateFunnelPage` = `not_yet_implemented`), `routes/funnels/+page.svelte`. Funnels contract + verified path documented in `../funnels/`.

## Purpose
Manage landing pages / funnels and guarantee correct **Meta Pixel + UTM tracking**. In GHL this is read + assisted-manual only; in the new platform we **own** the pages and publish them to Cloudflare.

## Core features (today, GHL)
1. **List funnels per account** — `GET /funnels/funnel/list`.
2. **Per-account tabs** + funnel/step listing.
3. **Live-page audit** — server-side fetch of published pages (SSRF-guarded) → detect Meta Pixel (`fbq(`, `fbevents.js`) + UTM markers.
4. **Pixel status bar** — per-page present/missing verdict vs the expected brand pixel.
5. **Assisted-manual pixel** (Option C, chosen) — hand the operator the exact snippet + copy button + GHL deep link to paste into Funnel → Settings → Tracking Code.

## HARD LIMIT (decided)
GHL funnel/page **WRITE is impossible** via the public API (no POST/PUT/PATCH for any auth type). Automated injection worked in the prototype only via a captured **browser-session token** (Option B) — rejected (fragile, ToS-grey, raises the vault threat model). See `module-capabilities-and-limits.md` §1.

## Target behavior in the new platform (the real build)
1. **CMS = system of record** for `sites / funnels / pages / page_versions` (Neon).
2. **Authoring:** templating engine (stdlib `html/template` + block registry) + Canvas visual editor (Svelte `/cms`) + TextGen copy (Module 9).
3. **Publish:** builder renders a page version → **Cloudflare Pages** deploy → custom hostname + DNS via Cloudflare API. `CMS → Cloudflare → customer`.
4. **Pixel injection becomes build-time & first-class** — author picks a brand; the builder injects `fbq('init', …)` + UTM into `<head>` automatically (no paste, no captured token).
5. **Audit inverts** — `AuditFunnels`/`auditHTML` becomes a **post-publish verification probe** confirming Cloudflare served the pixel we built in (same detectors, same SSRF guard).
6. **Custom HTML/JS** — role-gated, server-side sanitized (`page_versions.sanitized` gate; builder refuses unsanitized), strict CSP + nonces at the edge, cross-origin sandboxed preview.

## Data model
`sites, funnels, pages, page_versions (content_json, custom_html, custom_js, head_snippets, sanitized), pixels (brand→pixelId, migrated from pixels.go), domains (cf_zone_id, custom hostname, ssl_status), publishes` — all Neon, CMS-owned.

## GHL API surface & ceiling
- Read: `GET /funnels/funnel/list`, `GET /funnels/funnel/{id}`, `GET /funnels/funnel/page/{id}`, domain lookups. Version `2021-07-28`.
- **Write: none.** The documented `PUT /funnels/funnel/page/{id}` does not function with PIT/OAuth.

## Migration notes & kill-switch trigger
- **Trigger:** a tenant's funnel is authored in the CMS → published to Cloudflare → DNS cut over → pixel verified by probe → disable `ghl:funnels` for that tenant. GHL funnels remain read/audit-only during transition.

## Open questions
- **Content import:** do we scrape/import existing GHL funnel pages into the CMS, or rebuild from templates?
- **Migration granularity:** per-tenant or per-funnel cut-over? Who flips DNS?
- **Custom-HTML policy:** allowed at all in v1, or trusted-blocks-only until the sanitizer is proven?
- **Scope of Canvas v1:** which block types (hero, form, booking, CTA, raw-html) ship first?
