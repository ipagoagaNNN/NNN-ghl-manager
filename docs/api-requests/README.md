# GHL API Request Reference

Captured, working GHL API v2 requests + their real response shapes. Each file documents one endpoint we've verified live against `services.leadconnectorhq.com`.

## Conventions

- **Never commit a real token.** Tokens are redacted to `pit-{{LOCATION_PIT}}` (location-scoped) or `pit-{{AGENCY_PIT}}` (agency-scoped). Real tokens live only in the Go vault at runtime.
- **Never commit a real locationId/companyId if it's client-sensitive.** Test IDs (e.g. the Miriam Test location) are fine to keep for reproducibility.
- Each file records: the curl, the HTTP status we observed, the response shape, the required `Version` header, and the token scope it needs.

## Why this folder exists

GHL's published docs and our derived `ghl-api-surface.md` have both proven lossy. These files are **live-verified ground truth** — captured by actually hitting the API — so the backend handlers can be checked against real behavior, not documentation.

## Index

| File | Endpoint | Version | Token scope | Status |
|------|----------|---------|-------------|--------|
| `get-objects-by-location.md` | `GET /objects/?locationId=` | 2023-02-21 | location | ✅ 200 verified |

See also: `../ghl-api-v2-auth-model.md` — the auth-model analysis that explains agency vs location scope and why our Connect flow needs to change.
