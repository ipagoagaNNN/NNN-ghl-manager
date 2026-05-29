# Get Objects by Location ("Get Objects — Miriam Test")

**Official doc:** https://marketplace.gohighlevel.com/docs/ghl/objects/get-object-by-location-id
**Verified live:** 2026-05-29 → **HTTP 200**

Returns the object schema (not records) for a location: the built-in `business` / `opportunity` / `contact` objects plus any custom objects, with their labels, required/searchable properties, primary display property, and icons.

## Request

```bash
curl --request GET \
  --url 'https://services.leadconnectorhq.com/objects/?locationId=6KtnVX1w8kxKgXeMzNGd' \
  --header 'Accept: application/json' \
  --header 'Authorization: Bearer pit-{{LOCATION_PIT}}' \
  --header 'Version: 2023-02-21'
```

- **Method:** `GET`
- **Path:** `/objects/`
- **Query:** `locationId` (required) — the sub-account ID
- **Headers:**
  - `Authorization: Bearer pit-{{LOCATION_PIT}}` — **location-scoped** Private Integration Token
  - `Version: 2023-02-21` — **required**; this v2 endpoint rejects/ignores the legacy `2021-07-28`
  - `Accept: application/json`

> Test location: `6KtnVX1w8kxKgXeMzNGd` (Miriam Test). Token redacted — it is a location PIT stored only in the Go vault at runtime.

## Response shape (200)

```jsonc
{
  "objects": [
    {
      "id": "6a04d25edb5557442e22c11d",
      "labels": { "singular": "Company", "plural": "Companies" },
      "description": "Contains list of all businesses, ...",
      "requiredProperties": ["business.name"],
      "searchableProperties": ["business.name", "business.email"],
      "primaryDisplayProperty": "business.name",
      "key": "business",
      "uniqueProperties": [],
      "locationId": "6KtnVX1w8kxKgXeMzNGd",
      "updatedAt": "2026-05-13T19:34:54.312Z",
      "createdAt": "2026-05-13T19:34:54.312Z",
      "icon": { "class": "...", "svg": "<svg .../>" },
      "type": "SYSTEM_DEFINED"
      // contact object additionally has "addRecordConfiguration": [ { key, order, isGroupField, required, isEditable }, ... ]
    }
    // ... opportunity, contact, and any custom objects
  ],
  "cache": true,
  "traceId": "afb625c9-..."
}
```

Built-in objects always present: `business` (Company), `opportunity` (Opportunity), `contact` (Contact).

## Notes for our codebase

- This endpoint is **not yet used** by any module. It's the entry point if we add an Objects/Records (custom objects, opportunities) module later.
- Confirms the v2 **Version `2023-02-21`** requirement — distinct from the `2021-07-28` our current handlers send. See `../ghl-api-v2-auth-model.md`.
- Confirms the token is **location-scoped** (works with a per-location PIT + `locationId`).
