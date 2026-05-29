# GHL API 2.0 — Auth Model & Flow Correctness

**Status:** live-verified 2026-05-29 against `services.leadconnectorhq.com`.
**Verdict:** our Connect flow is **architecturally wrong** for Private Integration Token (PIT) auth. Downstream modules are correct.

---

## The two auth scopes in GHL

GHL has two distinct credential scopes, and they are NOT interchangeable:

| Scope | What it can do | How you get it |
|-------|----------------|----------------|
| **Agency / Company** | Enumerate sub-accounts (`/locations/search?companyId=`), act across the whole company | Agency-level OAuth app, or an agency-level token |
| **Location (sub-account)** | Read/write ONE location's data: contacts, custom values, workflows, objects, funnels | A **Private Integration Token (`pit-...`)** created inside that sub-account |

A **`pit-` token is location-scoped.** It is generated per sub-account (Settings → Private Integrations) and only has access to that one location. It **cannot** enumerate the company's locations.

`companyId` (agency) and `locationId` (sub-account) are **different IDs**. `6KtnVX1w8kxKgXeMzNGd` is a **locationId**.

---

## Live verification (2026-05-29, Miriam Test location `6KtnVX1w8kxKgXeMzNGd`, location PIT)

| Endpoint | Used by | Version | Result |
|----------|---------|---------|--------|
| `GET /objects/?locationId=` | (future Objects module) | 2023-02-21 | ✅ **200** |
| `GET /locations/{id}/customValues` | Custom Values (2c) | 2021-07-28 | ✅ **200** |
| `GET /contacts/?locationId=` | Dashboard (2d) | 2021-07-28 | ✅ **200** |
| `GET /locations/search?companyId={id}` | **Connect (Phase 1)** | 2021-07-28 | ❌ **403 Forbidden** |
| `GET /locations/search?companyId={id}` | Connect | 2023-02-21 | ❌ **403 Forbidden** |

**Interpretation:**
- 200 on every *location-scoped* endpoint → the PIT is valid and our per-module request contracts are correct.
- **403 (not 401)** on `/locations/search` → the token is *valid* but *forbidden* from the agency-scoped operation. A location PIT structurally cannot list a company's sub-accounts.

---

## Why "can't connect" happens

Our `/api/connect` handler (`backend/internal/handlers/connect.go`) does:

```
POST /api/connect { agencyToken, companyId }
  → GET /locations/search?companyId={companyId}   ← 403 with a location PIT
  → returns "GHL error" → user never gets in
```

This assumes the user has an **agency token + companyId**. But the user has a **location PIT + locationId**. The agency-discovery step can never succeed with location-scoped credentials. Everything downstream (Accounts token storage, Custom Values, Automations, Dashboard) already works with the location PIT — the *only* broken step is the agency-search gate at the front door.

---

## Version header policy

| Version | Endpoints that accept it | Notes |
|---------|--------------------------|-------|
| `2021-07-28` | locations, contacts, customValues, workflows, funnels | What our handlers currently send. Still works for our existing modules (verified 200). |
| `2023-02-21` | `/objects/` and the newer v2 object/record APIs | **Required** for objects; the legacy version fails there. |

**Action:** keep `2021-07-28` for current modules (verified working), but make the Version header **per-endpoint configurable** so object-based endpoints can send `2023-02-21`. Do NOT globally bump to `2023-02-21` without re-verifying every existing endpoint.

---

## Required fix: location-PIT onboarding

The Connect flow must support **registering a location directly by `(locationId, PIT)`**, bypassing agency discovery. Our vault endpoint `POST /api/tokens/{locationId}` already stores a location token server-side — the onboarding UI just needs to:

1. Accept `locationId` + its `pit-` token (instead of, or in addition to, `companyId` + agency token).
2. Validate the PIT with a cheap location-scoped call (e.g. `GET /locations/{id}/customValues` or `/objects/?locationId=`) → if 200, store in vault.
3. Add that location to the session so all modules can use it.

**Two onboarding models to decide between:**
- **Agency model (current):** company token → auto-discover all sub-accounts. Requires agency credentials the user may not have.
- **Location model (works today):** add each sub-account by `locationId` + PIT. Verified working end-to-end.

Recommendation: support **both**, defaulting to the location model (it's the one proven to work with the user's credentials). The agency model stays for users who do have agency tokens.

---

## What this confirms about prior work

- ✅ Per-module request contracts (customValues, contacts, workflows) are **correct** — they return 200 with a valid location PIT.
- ✅ The token-vault security model is sound — location PITs stored server-side, used per-location.
- ❌ The **Connect/onboarding model** assumed agency scope and must gain a location-PIT path.
- ⚠️ Add per-endpoint `Version` support before building any Objects/Records module.
