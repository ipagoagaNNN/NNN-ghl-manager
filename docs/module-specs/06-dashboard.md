# Module 6 — Dashboard (core-feature spec)

- **Status:** draft (gate for own-core analytics · order 5)
- **Target home:** Own core (Neon aggregates) for business analytics + Grafana for ops metrics
- **Current state in repo:** `handlers/dashboard.go` (`GET /api/dashboard/:loc/contacts`, paginated, 60s cache, duplicate-page guard), `routes/dashboard/+page.svelte`. Contacts-based; empty-state added (s4).

## Purpose
Leads analytics for each sub-account: leads by day, by source, campaign breakdown, and (eventually) conversions — so the agency can see what's working.

## Core features
1. **Leads-by-day chart** — aggregate contacts by `dateAdded || createdAt`.
2. **Contacts paginated fetch** — `GET /contacts/?locationId=&limit=100&sortBy=date_added&page=N` with a **duplicate-page guard** + `meta.nextPageUrl` authoritative stop (GHL's `page` param is unreliable and can re-return a page).
3. **Source aggregation + top-campaigns table** — group by `source`; campaign stats join contacts with custom fields.
4. **Date-range filter** — `startDate`/`endDate` (server-side filtering).
5. **Empty-state** — explicit "no leads in range" + always-visible per-account table (so a 0-contact account doesn't render blank).

### Enhancements available (prototype did these; not yet built)
- **Facebook-attributed leads** — `source` / `attributionSource.utmSource` / `fbclid`.
- **Converted / booked leads** — tags + status + custom-field match (`booked|appointment|deposit|…`).
- **Campaign breakdown** — from contact custom fields (`campaign` / `form name` / `ad name`); needs `GET /locations/{loc}/customFields` to resolve field names.

## Data model
- `contacts` (Neon) — `{id, email, phone, firstName, lastName, dateAdded, source, tags[]}` + custom JSONB.
- Aggregates computed from `contacts` (+ `activities` for conversions). No new tables; reuse the CRM core.

## GHL API surface & ceiling
- `GET /contacts/?locationId=&limit=&sortBy=date_added&page=&startDate=&endDate=` → `{contacts, count, total}`.
- `GET /locations/{loc}/customFields` (resolve campaign field names). Version `2021-07-28`. ~80ms sleep between pages.
- **Ceiling:** read-only; pagination unreliable (must guard).

## Target behavior in the new platform
- Dashboard reads **Neon aggregates** (contacts synced in by the reconciler) instead of live-paginating GHL every load — faster, no rate-limit exposure, works once GHL is off.
- Ops/system metrics (sync lag, DLQ depth, adapter health) live in **Grafana**, not this page; this page stays product analytics.

## Migration notes & kill-switch trigger
- **Trigger:** dashboard queries Neon `contacts`/aggregates instead of paginating GHL `/contacts/`. GHL contact read disabled once the reconciler keeps Neon current.

## Open questions
- **Metric definitions:** the exact rules for "converted/booked" and "FB-attributed" (tag names, status values, custom-field keys) — owner must specify.
- **Refresh model:** live query vs nightly rollup table? (Affects whether we precompute aggregates.)
- **Cross-account roll-up:** is an agency-wide dashboard (all locations) needed, or per-account only?
