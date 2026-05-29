# Sites & Funnels — docs area

Reference docs for **Phase 2e (Sites & Funnels)**, the highest-risk module because pixel
injection mutates live customer-facing pages.

**Docs-first rule:** no Phase 2e code ships until the path here is verified. These three docs
are the ground truth; we do not re-derive from the 12.6k-line prototype during implementation.

| File | Purpose |
|------|---------|
| `funnels-api-contract.md` | The verified GHL API v2 funnel surface — official marketplace docs reconciled against how the prototype actually called it. Read vs write split, Version header, response shapes. |
| `verified-implementation-path.md` | The reconciled plan for Phase 2e: what we ship (read + pixel audit), what we defer (pixel write), and the user decision the write-side requires. |

## TL;DR (the load-bearing finding)

GHL's **public API v2 Funnels group is read-only.** There is **no** documented
POST/PUT/PATCH to update a funnel, update a funnel page, or set head/body tracking code.

- ✅ **Read** (list funnels, list/count pages, page detail) works with a **location PIT**, `Version: 2021-07-28`.
- ✅ **Pixel audit** (detect whether a live page has the right FB pixel / UTM / Acuity script) is a
  read-only capability — and our **Go backend does it server-side**, a strict upgrade over the
  prototype's flaky public-CORS-proxy approach.
- ❌ **Pixel injection / head-code write** is **not possible with a `pit-` token.** The prototype
  brute-forced a PUT/PATCH cascade that fails, then shows the toast literally reading
  *"GHL API does not allow writing funnel head code with pit- tokens."* Real writes in the prototype
  used a **captured browser-session token** (`token-id` + browser `Authorization` + `channel:APP` +
  `source:WEB_USER` headers) — a separate, fragile, ToS-grey auth path.

→ See `verified-implementation-path.md` for the ship/defer split and the write-side decision.

## Sources

- Official: <https://marketplace.gohighlevel.com/docs/ghl/funnels> (list funnels, list pages, count pages, redirect lookup — all GET)
- Prototype oracle: `my-ghl-manager/documentation/ghl-manager-final.2026-04-24-123140.html`
  (funnel read ~L2826/L5239; pixel scan ~L2863/L5221; write cascade ~L5066/L5410/L5461; browser-session
  auth ~L7888; bookmarklet ~L5501)
- Auth scopes: `../ghl-api-v2-auth-model.md`
