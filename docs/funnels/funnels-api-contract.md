# GHL Funnels API v2 — Verified Contract

**Status:** docs-verified 2026-05-29. Official marketplace docs reconciled against the prototype
(the production oracle). Not yet live-verified against `services.leadconnectorhq.com` — see
"Open verifications" at the bottom.

**Headline:** the public Funnels API is **read-only**. Every documented endpoint is a `GET`.

---

## 1. Official endpoint surface

From <https://marketplace.gohighlevel.com/docs/ghl/funnels> — the entire Funnels API group:

| Capability | Method | Path | Notes |
|------------|--------|------|-------|
| List funnels | `GET` | `/funnels/funnel/list` | query by `locationId` (+ `limit`, `offset`, `type`, `category`, `parentId`) |
| List funnel pages | `GET` | `/funnels/page` | by `funnelId` (+ `locationId`, paging) |
| Count funnel pages | `GET` | `/funnels/page/count` | by `funnelId` |
| Redirect lookup | `GET` | `/funnels/lookup/redirect` (group) | URL-redirect listing; not needed for Phase 2e |

**There is NO documented `POST` / `PUT` / `PATCH` / `DELETE`** — no create funnel, no update funnel,
no update page, no set head/body tracking code. This is the single most important fact for Phase 2e.

---

## 2. What the prototype actually called (the oracle)

The prototype hardcodes `Version: 2021-07-28` on **every** funnel call (`apiFetch`, HTML L6311-6321)
and it works in production. Trust that over any inferred value.

### 2a. Read calls — these work with a location PIT

| Prototype call | HTML ref | Maps to official |
|----------------|----------|------------------|
| `GET /funnels/funnel/list?locationId={loc}&limit=100` | L5239, L5400 | ✅ `GET /funnels/funnel/list` |
| `GET /funnels/funnel/page/{pageId}?locationId={loc}` | L5449 | ⚠️ page-by-id detail — NOT in the official list group (see open verifications) |
| `GET /funnels/page/{pageId}?locationId={loc}` (alt) | L2983 | ⚠️ alternate page-detail shape |
| `GET /locations/{loc}/domains/{domainId}` or `/domains/{domainId}` | L2834-2835 | domain resolution to build page URLs |

**Funnel object shape** (envelope fallback `funnels \|\| data \|\| list \|\| items`):
```
{
  id | _id,
  name | title,
  steps: [ { name|title, slug|path, ... } ],   // steps are the pages
  domain | domainName | url | previewUrl,        // may be absent → resolve via domainId
  domainId,
  headCode | headerCode | head,                  // present on some accounts, not all
  bodyCode | footerCode | body
}
```

Page URL is built as `https://{domain}/{step.slug}` (L5271).

### 2b. Write calls — these FAIL with a location PIT

The prototype brute-forces a cascade and falls back to an explicit failure toast:

| Attempt | HTML ref | Result with PIT |
|---------|----------|-----------------|
| `PUT /funnels/funnel/{id}` `{name, headCode, locationId}` | L5410 | ❌ |
| `PATCH /funnels/funnel/{id}` `{headCode, locationId}` | L5411 | ❌ |
| `PUT \| PATCH /funnels/{id}` `{headCode, locationId}` | L5412-5413 | ❌ |
| `PUT \| PATCH /funnels/funnel/page/{pageId}` `{locationId, headCode}` | L5464-5467 | ❌ |

On exhausting the cascade: **`showToast('GHL API does not allow writing funnel head code with pit- tokens')`** (L5434).
The author knew. The write feature was never functional via the public PIT API.

### 2c. The real write path — captured browser-session token

Form saves (and by extension any builder mutation) use a **different auth** captured from a
logged-in GHL browser session (`getFormSessionToken`, used at L7888):

```
POST /forms/{id}            // and analogous builder endpoints
headers:
  token-id: <captured from browser session>
  Authorization: <browser bearer, NOT the pit->
  channel: APP
  source: WEB_USER
  Version: 2021-07-28
```

This is an **internal/private** request signature, not the public API. It is fragile (breaks when
GHL rotates session auth), ToS-grey, and the captured token is *more* sensitive than a PIT. Treat
this as out-of-scope unless the user explicitly opts in (see `verified-implementation-path.md` §Write).

---

## 3. Pixel / tracking detection (read-only — the real value)

Detection is pure string/regex matching on fetched page HTML. From `extractTrackingScanDetails`
(L2863) and `scanFunnelTracking` (L5286-5293):

| Signal | Detection |
|--------|-----------|
| Has FB pixel | HTML includes `fbq(` or `fbevents.js` or `Meta Pixel` |
| Pixel ID | regex `fbq\('init',\s*['"]?(\d+)['"]?\)` → captured digits |
| Has UTM/tracking script | includes `user_ip_address` \| `full_url_utm` \| `buildfbc` \| `api.ipify.org` \| `fbclid` |
| Has Acuity footer | includes `ACUITY_FIELD_ID` \| `acuityscheduling.com` \| `MARKETING DATA TRACKER` |
| Acuity field id | regex `ACUITY_FIELD_ID\s*=\s*['"]([^'"]+)['"]` |

**How the prototype fetched live HTML:** browser-side via a rotation of **public CORS proxies**
(`corsproxy.io`, `api.allorigins.win`, `thingproxy.freeboard.io`, `api.codetabs.com` — L5251-5256),
because a browser can't fetch a customer's funnel page cross-origin. Plus a manual **bookmarklet**
fallback (L5501) the operator clicks while on the page.

**Known brand pixels** (hardcoded in the bookmarklet, L5509) — these belong in a config keyed by domain:

| Pixel ID | Brand |
|----------|-------|
| `1180664739949233` | First Touch Beauty |
| `540031443835405` | No Needle Needed |
| `819221296693583` | Adv Beauty Treatments |

---

## 4. Version header

| Endpoints | Version | Source |
|-----------|---------|--------|
| All funnel reads (list, page) | `2021-07-28` | prototype hardcodes it everywhere; verified-working in production |
| Domain resolution | `2021-07-28` | same |

Do **not** assume `2023-02-21` for funnels. (A docs-fetch model *guessed* 2023-02-21 for the pages
endpoint based on the doc date, but that was inference, not a read. The production oracle uses
2021-07-28 and gets 200s.) If a funnel read returns 4xx during implementation, re-check the official
page's required Version then — but start from 2021-07-28.

---

## 5. Live verification (2026-05-29, session 4 — location `6KtnVX1w8kxKgXeMzNGd`, location PIT)

`GET /funnels/funnel/list?locationId=…&limit=100`, `Version: 2021-07-28` → **HTTP 200**, 6 funnels.

**Confirmed real field shapes (differ from the prototype's guessed names):**

| What | Prototype assumed | Live reality | Handler behavior |
|------|-------------------|--------------|------------------|
| Envelope | `funnels\|data\|list\|items` | `funnels` | ✅ fallback covers it |
| Funnel id | `id\|_id` | `_id` (e.g. `qf6r9ZTw0FjUUe74JWRo`) | ✅ `id()` falls back to `_id` |
| Funnel name | `name\|title` | `name` | ✅ |
| Funnel head/body code | `headCode\|headerCode` / `bodyCode\|footerCode` | **`trackingCodeHead` / `trackingCodeBody`** | ✅ added both + legacy fallbacks |
| Funnel domain | `domain\|domainName\|url\|previewUrl` | `url` is a **relative path** (`/triple-lift`); `domainId` empty | ✅ `cleanDomain` rejects paths → falls back to `LocMeta.Domain` |
| Step path | `slug\|path` | step `url` (e.g. `/triple-lift-form-79-95`); no `slug`/`path` | ✅ `slug()` falls back to `url` |

**Key consequences (drove implementation changes):**
1. **Funnel-level pixel is readable without a live fetch.** `trackingCodeHead`/`Body` carry the pixel
   snippet. `ListFunnels` now reports `configuredPixel` + `hasTrackingPixel` per funnel. Live result:
   2/6 funnels have pixel `819221296693583` (Adv Beauty) in their tracking code; 4/6 have none.
   This makes the Funnels page useful **even when no domain is configured** (which is this account).
2. **Live-page audit needs a configured domain.** Funnels here carry only a relative path + empty
   `domainId`, so `buildStepURL` can't form a URL until the location's domain is set in Accounts
   (`LocMeta.Domain`). Until then audit rows show `no-url`; the funnel-level `configuredPixel` still works.
3. `steps[]` from the list call is sufficient to enumerate pages — no separate `/funnels/page` call needed
   for the audit. (Each step also has a nested `pages[]` we don't yet descend into — future refinement.)

**Still open (non-blocking):** descend into step `pages[]` for multi-page steps; optionally read
`trackingCodeHead` as a fast pre-filter before the live-page scan.
