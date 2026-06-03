# Module 2 — Accounts (core-feature spec)

- **Status:** draft (gate for Track A·3)
- **Target home:** Own core — `organizations` table (+ optional Salesforce Account mirror)
- **Current state in repo:** `handlers/accounts.go` (`GET /api/accounts`, `GET /api/accounts/library`, `PUT /api/accounts/:loc/meta`, `POST /api/tokens/:loc`), `routes/accounts/+page.svelte`, `stores/accounts.svelte.ts`.

## Purpose
The control panel for the agency's sub-accounts: which locations exist, which have tokens, their per-account metadata, and which are **selected** for bulk operations. This selection drives Custom Values, Dashboard, Funnels, etc.

## Core features
1. **Sub-account list** — render all locations with a `hasToken` indicator.
2. **Per-account token input** — `POST /api/tokens/:locationId` → vault (token never returned).
3. **Per-account metadata** — domain, Acuity field, calendar IDs, `active` flag → `PUT /api/accounts/:loc/meta`.
4. **Library editor** — 6-column grid (name, token-present, domain, acuity, calendarIds, active).
5. **CSV export / import** of the library — export client-side; import = frontend parse → `POST /api/accounts/library`.
6. **Select / deselect accounts** — `$state<Set<string>>`; the selection set is the input to every bulk module.

## Data model
`AccountLibraryEntry { name, token, domain, acuity, calendarIds, active }` keyed by locationId →
- identity/meta → `organizations` (agency→location tree via `parent_id`; `ghl_location_id` UNIQUE).
- `token` → `integration_credentials` (never in the library response).
- `active` + selection → drives `sync_state.direction` / bulk-op targeting.

## GHL API surface & ceiling
- No direct GHL writes — pure vault/Neon read+write. The location list originates from Connect (`/locations/search`) or, post-migration, from the `organizations` table.
- **Ceiling:** none relevant; this is local state management.

## Target behavior in the new platform
- The "library" **becomes the `organizations` table** (durable, not RAM). Selection persists.
- Each org optionally mirrors to a **Salesforce Account** (Neon→SF, Neon wins per the ownership matrix).
- Multi-brand grouping (e.g., NNN / FTB / Advanced Beauty) modeled via the agency→location tree.

## Migration notes & kill-switch trigger
- **Trigger:** account list served from Neon `organizations` (not `vault.AllLocTokens()`); GHL location read disabled with no UI gap.

## Open questions
- **Salesforce mapping:** does each sub-account = one Salesforce Account? How are brands grouped (tree vs tag)?
- **Acuity / calendar IDs:** still needed post-migration, or superseded by a real scheduling integration?
- **Library CSV:** keep as an import/export format, or replace with direct org CRUD once persisted?
