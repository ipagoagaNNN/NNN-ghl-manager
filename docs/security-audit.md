# Security Audit

## Current State (HTML Prototype)

### Critical Vulnerabilities

#### CRIT-01: Tokens in localStorage
**Severity**: Critical  
**Location**: `localStorage.setItem('ghl_loc_tokens_v2', JSON.stringify(tokenMap))`  
**Impact**: Any XSS vulnerability, browser extension, or malicious script on the page can read all sub-account tokens. An attacker with read access to localStorage has full agency access to all GHL accounts.  
**Fix**: Move all tokens to server-side session store (Go vault). Backend proxy injects tokens; frontend never holds them.

#### CRIT-02: Agency token in localStorage
**Severity**: Critical  
**Location**: `localStorage.setItem('ghl_agency_token', agencyToken)`  
**Impact**: Agency token grants read access to all sub-accounts under the company ID. Same risk as CRIT-01 but wider blast radius.  
**Fix**: Same — server-side only.

#### CRIT-03: Direct browser → GHL API calls
**Severity**: High  
**Location**: All `apiFetch(token, ...)` calls go directly to `https://services.leadconnectorhq.com`  
**Impact**: Tokens are transmitted from the browser, visible in DevTools Network tab. No server-side audit trail. Any CORS bypass or browser compromise exposes tokens.  
**Fix**: All API calls route through Go proxy at `/api/ghl/*`. Proxy reads token from vault, browser never sees it.

#### CRIT-04: No rate limit protection
**Severity**: High  
**Location**: App sends GHL API requests as fast as JS allows (only 80-150ms sleep)  
**Impact**: Easy to hit GHL rate limits, causing 429 errors for legitimate operations. A bug or user action could trigger API key suspension.  
**Fix**: Go middleware: per-token sliding window rate limiter with request queuing.

#### CRIT-05: No authentication layer
**Severity**: High  
**Location**: The HTML file — anyone who opens it has full access  
**Impact**: No access control. Tokens stored locally mean any user on the same machine can access any account.  
**Fix (Phase 2)**: JWT auth with bcrypt passwords. Single-user initially, RBAC in Phase 3.

---

### Medium Vulnerabilities

#### MED-01: Form session tokens in localStorage
**Severity**: Medium  
**Location**: `localStorage.setItem('ghl_form_session_tokens_v1', ...)`  
**Impact**: Browser-captured form auth tokens are persisted in localStorage, same risk as CRIT-01.  
**Fix**: Move to server-side session store.

#### MED-02: postMessage to `*` origin
**Severity**: Medium  
**Location**: `window.postMessage({ source: 'ghl-manager', ... }, '*')`  
**Impact**: Extension bridge messages are sent to any origin. Malicious iframes could intercept.  
**Fix**: In the refactored app, restrict to `window.location.origin` where technically possible. Extension bridge protocol does not carry tokens, so risk is lower.

#### MED-03: CORS extension dependency
**Severity**: Medium  
**Location**: Error message: `"Make sure CORS extension is active (orange icon)"`  
**Impact**: App currently requires a browser CORS bypass extension to make cross-origin requests to GHL API. This is a smell that the architecture is wrong, not a security feature.  
**Fix**: Backend proxy eliminates the CORS issue entirely — no extension needed for API calls.

#### MED-04: No input sanitization on API responses
**Severity**: Medium  
**Location**: API responses are directly rendered into innerHTML via `esc()` helper  
**Impact**: `esc()` function protects against basic XSS but is custom-rolled, not battle-tested.  
**Fix**: Svelte auto-escapes template interpolations. `{variable}` in Svelte is safe by default.

---

### Low / Informational

#### LOW-01: Agency token visible in input field
Displayed in `<input type="text">` — visible on screen. Not a code vulnerability but an operational risk.  
**Fix**: Use `<input type="password">` and never echo the token back after submission.

#### LOW-02: Pixel IDs hardcoded
Meta Pixel IDs are hardcoded in the HTML. Not a security issue per se, but they expose brand structure to anyone who views source.  
**Fix**: Move to server-side config/env vars.

#### LOW-03: No HTTPS enforcement
**Fix**: Serve through a reverse proxy (nginx/Caddy) with TLS. Go server serves HTTP; Caddy handles TLS termination.

---

## What the Backend Proxy Fixes (Phase 1)

| Threat | Fixed? |
|--------|--------|
| CRIT-01: Tokens in localStorage | ✅ Vault server-side |
| CRIT-02: Agency token in localStorage | ✅ Vault server-side |
| CRIT-03: Direct browser→GHL calls | ✅ All through proxy |
| CRIT-04: No rate limiting | ✅ Go middleware |
| MED-01: Form session tokens exposed | ✅ Server-side |
| MED-03: CORS extension dependency | ✅ Eliminated |
| MED-04: Custom XSS escaping | ✅ Svelte auto-escape |
| LOW-01: Token in visible input | ✅ Never echoed back |
| LOW-02: Pixel IDs in source | ✅ Server config |

---

## Auth Roadmap (Phase 2+)

### Phase 2 — Single-user auth
- `POST /api/auth/login` — bcrypt password check → issue JWT (access + refresh)
- `GET /api/auth/refresh` — refresh token rotation
- All routes protected by `auth.go` middleware (already wired as no-op from Phase 1)
- httpOnly cookie for JWT — never accessible to JS

### Phase 3 — RBAC
- Roles: `agency_owner`, `location_manager`, `read_only`
- `agency_owner` — full access, can manage tokens
- `location_manager` — access to assigned locationIds only
- `read_only` — dashboard + reports only, no mutations

### Phase 4 — Enterprise
- Audit log: every GHL mutation (PUT custom value, inject pixel, etc.) logged with user, timestamp, diff
- MFA via TOTP (authenticator app)
- Session revocation (invalidate all active sessions for a user)
- SOC2-relevant event log retention (90 days minimum)

---

## Security Checklist Before Phase 1 Ship

**Audited 2026-05-26 (session 2). 5/6 ✅, 1/6 partial (Phase 2-gated).**

- [x] **Go server has no route that returns token values to frontend** ✅
  - `Connect` returns `{LocationCount, Locations}` only (no token field)
  - `SaveToken` returns 204 No Content (no body)
  - `ListAccounts` returns `{configuredLocations: [ids], count}` — IDs only, never values
  - All stub handlers + GHL proxy responses verified — no echo paths exist
- [ ] **Session cookie is `HttpOnly`, `Secure`, `SameSite=Strict`** ⚠️ Phase 2-gated
  - No cookie is set in Phase 1 because there's no auth (no-op middleware stub)
  - Acceptable for localhost dev; **MUST be implemented in Phase 2 JWT** before any non-localhost deploy
  - Documented limitation: vault is shared across callers in Phase 1
- [x] **CORS on Go server allows only `localhost:5173` (dev)** ✅
  - `middleware/cors.go` allow-list: `http://localhost:5173`, `http://localhost:4173` (Vite preview)
  - `Access-Control-Allow-Origin` only set if origin is in the list — no wildcard
  - Production domain to be added at Phase 4 deploy time
  - **Note**: Backend listens on `:8091` (8080, 8090, 9090 all reserved by Docker on Iker's dev machine — see `com.docker.backend.exe` netstat). Vite proxy forwards `/api/*` from `:5173` → `:8091`. CORS allow-list is independent of backend port. Rust workers use `:8081` + `:8082`.
- [x] **GHL base URL is not configurable from frontend (hardcoded in proxy)** ✅
  - `store/vault.go:8` — `const ghlBase = "https://services.leadconnectorhq.com"`
  - No env var, no config file, no API endpoint to mutate
  - Proxy enforces a defensive prefix check on every forwarded request (rejects requests where `targetURL` doesn't start with the locked base)
- [x] **Rate limiter is enforced before proxy forwards request** ✅
  - `main.go:36-40` middleware chain: CORS → RateLimit → Auth → handler/proxy
  - `middleware/ratelimit.go` — per-IP sliding window (60 req/min default)
  - Per-token GHL-specific back-off ALSO runs inside proxy for 429 responses (3 retries with 500ms × attempt back-off)
- [x] **`go vet` + `gosec` pass with no findings on `internal/proxy` and `internal/store`** ✅
  - `go vet ./...` exit:0
  - `gosec ./internal/proxy/ ./internal/store/` exit:0 — zero findings
  - `gosec ./...` (full backend) exit:0 — zero findings
  - Fixed during Phase 1e audit: G114 (server timeouts), 3× G104 (json.Encode errors), 2× G704 (SSRF — suppressed with `#nosec` + defensive prefix check, explanation in code comments)

### Verification artifacts

```
$ cd backend && go vet ./...
exit 0
$ go build ./...
exit 0
$ gosec ./...
Summary: Files: 10, Lines: 547, Issues: 0
exit 0
```

### Known Phase 1 limitations carried forward

- No session/user identity. Anyone with network access to `:8080` can drive the vault. This is fine for `localhost` dev and is the explicit scope of Phase 1 (proxy + vault). Phase 2 closes the door with JWT.
- Per-IP rate limit is generic. Per-token GHL-specific quota tracking + queuing comes when we wire real workloads (target: Phase 2c when bulk CV updates start hitting GHL hard).
