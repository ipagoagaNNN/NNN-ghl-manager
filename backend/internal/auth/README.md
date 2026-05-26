# Auth (Phase 2 — not yet implemented)

This package will contain JWT-based authentication when Phase 2 ships.

## Planned contents

- `jwt.go` — token generation + validation (RS256)
- `session.go` — refresh token rotation, session revocation list
- `bcrypt.go` — password hashing (single-user to start)
- `claims.go` — JWT claims struct with user ID + role

## Why it's empty now

The `auth.go` middleware in `internal/middleware/` is already wired into every
route as a no-op. Adding auth in Phase 2 requires:
1. Implementing this package
2. Changing `middleware/auth.go` to call `auth.ValidateJWT(r)`
3. Zero handler changes needed

## Phase 3 — RBAC

Roles: `agency_owner`, `location_manager`, `read_only`

See `docs/security-audit.md` for the full auth roadmap.
