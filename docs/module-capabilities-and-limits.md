# Module Capabilities & Limits — what GHL's API lets us do

**Status:** researched 2026-05-29 (session 4), reconciled against the prototype + live API.
**Purpose:** answer "what are our real limits and possibilities per module" so we stop guessing.

The recurring theme: **GHL's public API v2 (PIT or OAuth) is read-rich but write-poor.** Most
"management" actions in the prototype that *wrote* to GHL relied on a captured **browser-session
token** (internal API), not the public API. That path is fragile and ToS-grey.

---

## 0. Data persistence — what / where / how we save

**There is no database yet. This is by design (Phase 1).** Everything lives in the Go backend's
in-memory vault and is lost when the backend restarts.

| Data | Where | Lifetime | Notes |
|------|-------|----------|-------|
| Sub-account tokens (`pit-…`) | Go `store.Vault.locTokens` (RWMutex map) | in-memory, lost on backend restart | **Never** sent to the browser. Responses only carry `hasToken: bool`. |
| Location metadata (name, domain, acuityField, calendarIds, active) | Go `store.Vault.locMeta` | in-memory, lost on restart | Written via `PUT /api/accounts/{loc}/meta`. No token here. |
| Agency token + companyId | Go `store.Vault` | in-memory | Only set if you connect via the agency path. |
| Frontend `accounts.entries` | Svelte `$state` | per page load; re-fetched from `/api/accounts/library` on mount | Mirror of the server library + `hasToken`. |
| Frontend `session` (connected, locations, selectedIds) | Svelte `$state` | **per page load — NOT persisted** | A browser refresh resets it → you land back on `/connect`. |

**Consequences you saw:**
- "We still have no database connected" — correct. The vault is RAM only.
- A backend restart (e.g. `go run` reload) wipes tokens + meta → you must reconnect.
- A browser refresh loses `session.connected` (frontend state isn't persisted to sessionStorage).

**Roadmap:** ADR-001/003 + the refactor plan call for Phase 2 to replace the in-memory vault with
an **encrypted persistent store** (the real DB) and httpOnly-cookie sessions. Until then, treat the
tool as a single-run workbench.

---

## 1. Sites & Funnels — pixel injection: the full option space

**Read:** ✅ fully supported (location PIT, `Version 2021-07-28`). List funnels, steps, funnel-level
tracking code (`trackingCodeHead/Body`), and audit live pages server-side. See `funnels/`.

**Write (inject pixel / set tracking code):** the public API **cannot do it.** Options, ranked:

| # | Approach | Feasible? | Cost / risk |
|---|----------|-----------|-------------|
| A | **Public API v2 funnel/page write** | ❌ **Does not exist.** The Funnels API group is GET-only (list funnels, list/count pages, redirect). No POST/PUT/PATCH to a funnel or page, with PIT *or* OAuth. There is no `funnels.write` endpoint to scope to. | n/a |
| B | **Captured browser-session token** | ⚠️ Works (the prototype's real write path). Send `token-id` + browser `Authorization` + `channel:APP` + `source:WEB_USER` headers, captured from a logged-in GHL session (bookmarklet). | High: breaks when GHL rotates session auth; ToS-grey; the captured token is *more* sensitive than a PIT — raises the vault's threat model. |
| C | **Assisted manual** ✅ chosen | ✅ Shipped (s4). Audit flags the gap; tool hands the operator the exact snippet + copy + GHL deep link to paste into Funnel → Settings → Tracking Code (Head). | Low: no fragile auth, no live-page mutation. Operator does the final paste. |
| D | **Official GHL Marketplace OAuth app** | 🔎 Even a full OAuth app gets the *same* documented endpoints — and funnels have **no write endpoint documented for any auth type.** OAuth would help for *multi-agency distribution* and finer scopes on the read side, not for funnel writes. | Worth it only if we productize/distribute; doesn't unlock pixel write. |

**Bottom line:** automated pixel injection is only possible via (B) the internal browser-session API.
We chose (C) assisted-manual as the safe default. If you ever want (B), it's a separate, clearly
flagged sub-project (capture flow + token storage + breakage monitoring).

**Evidence:** prototype's own failure toast — *"GHL API does not allow writing funnel head code with
pit- tokens"* (HTML L5434); its write attempts were a PUT/PATCH cascade that fails (L5410, L5461);
its working form-save used the browser-session headers (L7888-7926). See `funnels/funnels-api-contract.md` §2.

---

## 2. Automations (Workflows) — what's possible beyond a list

**Current:** read-only list (`GET /workflows/?locationId=`) → name, status, version, dates.

**API limit:** the Workflows API group is **list-only.** There is **no** documented endpoint to:
- get a single workflow's detail/steps,
- update a workflow,
- publish / unpublish / toggle status,
- trigger a workflow.

So "click a workflow to see/edit its steps" is **not possible via the public API** — same
browser-session limitation as funnels. The workflow builder is an internal-API surface.

**What we CAN still add (supported, contact-side):**
| Enhancement | Endpoint | Value |
|-------------|----------|-------|
| Enrollment count per workflow | `GET /workflows/{id}/contacts?locationId=` (prototype used at L3960 — verify live) | "How many contacts are in this workflow" column |
| Add a contact to a workflow | `POST /contacts/{contactId}/workflow/{workflowId}` (contact API) | Manual enrollment action |
| Remove a contact from a workflow | `DELETE /contacts/{contactId}/workflow/{workflowId}` | Manual un-enrollment |
| Richer list columns | already have the list payload | surface more fields we already receive |

**Recommendation:** keep Automations read-only for the workflow itself; optionally add the
enrollment-count column (needs a live check that `/workflows/{id}/contacts` works with a PIT).
Editing workflows is out of scope unless we adopt the browser-session path.

---

## 3. Dashboard — how it actually works (vs the prototype)

**Why it was blank:** the connected sub-account (`6KtnVX1w8kxKgXeMzNGd`) genuinely has **0 contacts**
(`GET /contacts/ → meta.total: 0`). Not a bug — there was simply nothing to chart. **Fixed:** the
page now shows an explicit "no leads in range" empty-state + an always-visible per-account table so
you can see the query ran. (Earlier it rendered nothing when leads = 0.)

**Data model — contacts-based (this matches the prototype's real dashboard, `loadDashboard` L5854):**
- Paginate `GET /contacts/?locationId=&limit=100&sortBy=date_added&page=N`.
- Aggregate by day (`dateAdded || createdAt`) and by `source`.
- The prototype's *other* "metrics" view (`findMetricValue`, L2056) read leads/conversions/campaigns
  straight off the **location object** (`loc.stats`/`loc.metrics`) — those fields are usually absent,
  so that view mostly showed zeros. Our contacts-based approach is the real one.

**Pagination caveat (fixed this session):** GHL's `page` param is **not reliable** — it can return the
same page again. The prototype guards with a per-page signature (L3306-3308). We were missing that →
on any account with ≥100 contacts we'd have **re-counted** pages. Added a duplicate-page guard +
`meta.nextPageUrl` authoritative stop in `dashboard.go`.

**Enhancements available (the prototype did these; we don't yet):**
- **Facebook-attributed leads** (`source`/`attributionSource.utmSource`/`fbclid`).
- **Converted/booked leads** (tags + status + custom-field matching `booked|appointment|deposit|…`).
- **Campaign breakdown** from contact custom fields (`campaign`/`form name`/`ad name`), which needs a
  `GET /locations/{loc}/customFields` lookup to resolve field names.
These are real, API-supported upgrades — worth doing once we test against an account that has contacts.

---

## Cross-cutting takeaway

The honest ceiling of this tool on the **public API** is: **read everything, write a narrow set**
(custom values ✅, tokens/meta in our vault ✅, contact↔workflow enrollment ✅). Funnel pages and
workflow definitions are **read-only** unless we take on the browser-session internal API. Every
"can we edit X?" question reduces to: *is there a public write endpoint?* — and for funnels/workflows
the answer is no.
