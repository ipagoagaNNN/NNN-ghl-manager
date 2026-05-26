# GHL API Surface

All requests target: `https://services.leadconnectorhq.com`

## Common Headers

```
Authorization: Bearer {token}    // pit-... token (agency or sub-account level)
Version: 2021-07-28
Content-Type: application/json   // only when sending a body
```

---

## Endpoints

### Agency — Locations

#### List sub-accounts
```
GET /locations/search?companyId={companyId}&skip={n}&limit=100
Authorization: Bearer {agencyToken}
```
**Response:**
```json
{
  "locations": [
    {
      "id": "abc123",
      "name": "Spa Name",
      "businessName": "Spa Business LLC",
      "email": "owner@spa.com",
      "phone": "+15555551234",
      "address": "123 Main St",
      "city": "Los Angeles",
      "state": "CA",
      "country": "US",
      "postalCode": "90001"
    }
  ]
}
```
Pagination: call until `locations.length < 100`, increment `skip` by 100 each time.

---

### Custom Values

#### Get all custom values for a location
```
GET /locations/{locationId}/customValues
Authorization: Bearer {locationToken}
```
**Response:**
```json
{
  "customValues": [
    { "id": "cv_abc", "name": "Brand Name", "value": "No Needle Needed", "fieldType": "TEXT" }
  ]
}
```

#### Update a custom value
```
PUT /locations/{locationId}/customValues/{customValueId}
Authorization: Bearer {locationToken}
Content-Type: application/json

{ "value": "New Value" }
```
**Response:** `{ "customValue": { ...updated } }`

---

### Custom Fields

#### Get custom fields schema for a location
```
GET /locations/{locationId}/customFields
Authorization: Bearer {locationToken}
```
**Response:**
```json
{
  "customFields": [
    { "id": "cf_abc", "name": "Field Name", "dataType": "TEXT", "position": 0 }
  ]
}
```

---

### Contacts

#### Paginated contacts fetch
```
GET /contacts/?locationId={locationId}&limit={pageSize}&sortBy=date_added&page={n}
Authorization: Bearer {locationToken}
```
Optional date filter: `&startDate={ISO}&endDate={ISO}`

**Response:**
```json
{
  "contacts": [
    {
      "id": "contact_abc",
      "email": "patient@example.com",
      "phone": "+15555551234",
      "firstName": "Jane",
      "lastName": "Doe",
      "dateAdded": "2026-01-15T10:00:00Z",
      "source": "facebook",
      "tags": ["new-patient", "consulted"]
    }
  ],
  "count": 1,
  "total": 1
}
```
Pagination: call until `contacts.length < limit` or duplicate page signature detected. Uses 80ms sleep between pages.

---

### Workflows (Automations)

#### List workflows for a location
```
GET /workflows/?locationId={locationId}
Authorization: Bearer {locationToken}
```
**Response:**
```json
{
  "workflows": [
    {
      "id": "wf_abc",
      "name": "New Patient Onboarding",
      "status": "published",
      "version": 2,
      "createdAt": "2026-01-01T00:00:00Z",
      "updatedAt": "2026-01-10T00:00:00Z"
    }
  ]
}
```

---

### Funnels

#### List funnels for a location
```
GET /funnels/funnel/list?locationId={locationId}&limit=100
Authorization: Bearer {locationToken}
```
**Response:**
```json
{
  "funnels": [
    {
      "id": "funnel_abc",
      "name": "Botox Landing Page",
      "steps": [
        { "id": "step_abc", "name": "Optin", "url": "https://spa.com/botox" }
      ]
    }
  ]
}
```

#### Get a specific funnel
```
GET /funnels/funnel/{funnelId}
Authorization: Bearer {locationToken}
```

#### Update a funnel
```
PUT /funnels/funnel/{funnelId}
Authorization: Bearer {locationToken}
Content-Type: application/json

{ "name": "New Name", "headCode": "...", "bodyCode": "..." }
```

#### Get a funnel page
```
GET /funnels/funnel/page/{pageId}
Authorization: Bearer {locationToken}
```
**Response:**
```json
{
  "page": {
    "id": "page_abc",
    "name": "Optin Page",
    "headCode": "<!-- existing head code -->",
    "bodyCode": "<!-- existing body code -->"
  }
}
```

#### Update a funnel page (pixel/tracking injection)
```
PUT /funnels/funnel/page/{pageId}
Authorization: Bearer {locationToken}
Content-Type: application/json

{
  "headCode": "<!-- Meta Pixel + UTM scripts injected here -->",
  "bodyCode": "..."
}
```

---

### Forms

#### List forms for a location
```
GET /forms/?locationId={locationId}
Authorization: Bearer {locationToken}
```
**Response:**
```json
{
  "forms": [
    {
      "id": "form_abc",
      "name": "New Patient Consultation",
      "submissionWebhook": "https://hooks.zapier.com/..."
    }
  ]
}
```

---

## Error Handling

- `429 Too Many Requests` — GHL rate limit hit; retry with back-off
- `401 Unauthorized` — Token expired or invalid
- `404 Not Found` — Resource doesn't exist or token lacks access
- All errors: response body is plain text, not JSON

## Rate Limits

- No documented limit published by GHL
- In practice: ~10 req/s per location token before 429
- App currently uses 80–150ms sleep between requests within a location
- Agency token `/locations/search` is more permissive

## Known Pixel IDs (hardcoded in prototype)

| Brand | Meta Pixel ID |
|-------|--------------|
| First Touch Beauty | 1180664739949233 |
| No Needle Needed | 540031443835405 |
| Advanced Beauty Treatments | 819221296693583 |
