# Module 1 — Connect (core-feature spec)

- **Status:** draft (gate for Track A·0 / Track B credential store)
- **Target home:** Own core — the encrypted credential store + onboarding flow
- **Current state in repo:** `handlers/connect.go` (`POST /api/connect`, `POST /api/connect/location`), `store/vault.go` (in-memory), `routes/connect/+page.svelte` (two-card UI). Working against real data.

## Purpose
The security gateway and bootstrap for everything else: onboard agency and/or sub-account credentials so every other module can call GHL — while **tokens never reach the browser** (ADR-001). Nothing works until Connect works.

## Core features
1. **Agency-path connect** — agency token + companyId → `GET /locations/search` → load all sub-accounts (paginated).
2. **Location-direct connect** — a single location PIT validated cheaply via `GET /locations/{id}/customValues`; bypasses agency discovery (the path that works today, since most PITs are location-scoped).
3. **Server-side credential storage** — tokens stored in the vault; responses carry only `hasToken: bool` + metadata, never the token.
4. **Connection status** — reactive status dot + sub-account count (`stores/session.svelte.ts`).
5. **Missing-token / not-connected warnings** — conditional UI; refresh currently drops to `/connect` (no session persistence yet).

## Data model
- Agency token + `companyId` → `integration_credentials` (provider=`ghl`, scope=`agency`; companyId in `meta`).
- Location PITs → `integration_credentials` (provider=`ghl`, scope=`location`, ref=locationId), **AES-GCM encrypted**.
- Location identity/meta → `organizations` (`ghl_location_id`, name, domain, active).
- Today these live in `store.Vault` (volatile RAM).

## GHL API surface & ceiling
- `GET /locations/search?companyId=&skip=&limit=100` (agency token; paginate until `<100`).
- `GET /locations/{id}/customValues` (location PIT validation probe). Version `2021-07-28`.
- **Ceiling:** read-only discovery/validation. Agency vs location scope is mutually exclusive (a `pit-` is usually location-scoped; `/locations/search` → 403 for it).

## Target behavior in the new platform
- Connect becomes **credential onboarding into the encrypted persistent store** (survives restart — the #1 gap fix).
- Generalizes from "connect GHL" to "connect any integration": the same onboarding surface later registers Salesforce OAuth, Cloudflare API tokens, WhatsApp, etc. — each landing in `integration_credentials`.
- Real session model: httpOnly-cookie session so a browser refresh no longer drops the connection.

## Migration notes & kill-switch trigger
- **Trigger to retire GHL connect:** credentials live in Postgres (encrypted) and location-direct onboarding works without the agency path. GHL connect stays available as long as GHL is a read source.
- Connect itself is never "killed" — it evolves into the multi-integration onboarding surface.

## Open questions
- **Session model:** httpOnly cookie + CSRF? Single-user or multi-operator (RBAC)? (Auth/JWT track depends on this.)
- **Default connect mode:** location-direct as primary (most PITs are location-scoped), agency as advanced?
- **Credential lifecycle:** rotation/expiry UX; what happens when a PIT is revoked upstream.
