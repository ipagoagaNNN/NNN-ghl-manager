# Module 3 — Custom Values (core-feature spec)

- **Status:** draft → **D-3.1 (NPL scanner) AGREED + SHIPPED (s6)**; core also shipped. Gate for Track A·6 — kept longest; the ONLY GHL-writable module.
- **Target home:** Own core (config KV) with GHL write-through while kept
- **Current state in repo:** `handlers/customvalues.go` (`GET /api/cv` fan-out, `POST /api/cv/bulk` parallel PUT, concurrency 6) + **`handlers/npl.go` (`GET /api/cv/npl-scan`, s6)**, `routes/custom-values/+page.svelte` (CV grid + NPL scanner panel). Core + NPL scanner shipped; inner Forms/Trigger-Links tabs deferred (Phase 2c+).

## Purpose
Bulk-view and bulk-edit GHL **custom values** across many sub-accounts in one pass (e.g., set "Brand Name", booking links, brand presets). This is the highest-value module because it's the **only** place GHL's public API supports writes.

## Core features
1. **Load CVs from selected accounts** — Go fan-out `GET /locations/{id}/customValues` (parallel, concurrency 6).
2. **Display grouped by key** — union of all CV names across selected accounts.
3. **Edit a single CV value** — per-cell pending-value state.
4. **Bulk apply** — `POST /api/cv/bulk` → parallel `PUT .../customValues/{cvId}` with `{ value }`.
5. **Brand presets** — one-click bulk-fill for NNN / FTB / Advanced Beauty / General.
6. **Inner tabs** — Values / **Forms** (list `GET /forms/` incl. `submissionWebhook`) / **Trigger Links** _(deferred)_.
7. **New Patient Link scanner** — cross-account audit of the booking-link CV vs the account's registered domain (valid / domain-mismatch / malformed / missing) + one-click bulk-fix via the existing CV write. **Shipped s6** (`GET /api/cv/npl-scan`; CV-key default "New Patient Link").

## Data model
`LocationCVData { cvs: CVItem[], forms?: FormItem[], triggerLinks?: TriggerLinkItem[] }`,
`CVItem { id, name, value, fieldType }` →
- `custom_fields` (org_id, entity, key, label, field_type, `external_ids.ghl=cvId`) + `custom_values` (value, per entity).
- Forms/TriggerLinks are list-only reference data (read from GHL).

## GHL API surface & ceiling
- `GET /locations/{id}/customValues` → `{ customValues: [{id,name,value,fieldType}] }`.
- `PUT /locations/{id}/customValues/{cvId}` body `{ value }` → **WRITE ✅** (the only supported write).
- `GET /forms/?locationId=` (forms list), `GET /locations/{id}/customFields` (field schema). Version `2021-07-28`.
- **Ceiling:** read+write on custom values; forms/customFields read-only.

## Target behavior in the new platform
- Neon `custom_values` becomes the **source of truth**; brand presets become reusable named templates.
- Bulk update writes Neon **and** mirrors to GHL via the GHL adapter while GHL is kept (write-through).
- Optionally sync select CVs to Salesforce custom fields (per ownership matrix — follows parent entity).

## Migration notes & kill-switch trigger
- **Trigger:** Neon holds CV state as source of truth; bulk update writes Neon (+ optional GHL mirror); flip `ghl:customvalues` off once nothing reads GHL CVs directly. **Kept longest** precisely because it's GHL's only write surface.

## Open questions
- ~~**New Patient Link scanner:** exact definition?~~ **RESOLVED — D-3.1, shipped s6:** read-only audit (key default "New Patient Link", domain-match validation) + bulk-fix via the existing CV write. See `module-decisions/M3-custom-values-decisions.md`.
- **Forms & Trigger Links tabs:** read-only display, or do we need to manage them? (Form save historically used the browser-session path — out of scope unless revisited.)
- **Salesforce:** which CVs (if any) map to Salesforce fields vs stay GHL/Neon-only?
- **Brand presets:** static (code) or owner-editable templates in Neon?
