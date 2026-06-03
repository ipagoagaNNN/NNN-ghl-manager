# Module Core-Feature Specs

> **Purpose — the build gate.** Before any platform track from
> [`../architecture/platform-architecture-and-migration.md`](../architecture/platform-architecture-and-migration.md)
> is implemented, the affected GHL module's core features must be pinned down here. Standing up the
> scaffolding (Neon, NATS, adapters, CMS) ahead of these specs would force a major refactor of every
> module. **One spec = the contract for building that module's track.**

These specs are derived from the prototype reference docs — `feature-inventory.md`,
`module-capabilities-and-limits.md`, `ghl-api-surface.md`, `data-models.md`, `extension-bridge.md` —
not invented. They capture the *real* features and the *real* GHL API ceiling so we don't design
something GHL can't support.

## Cross-cutting reality: GHL's public API is read-rich, write-poor

| Module | GHL API ceiling | Implication |
|---|---|---|
| Custom Values | **READ + WRITE ✅** | The *only* module GHL lets us write. Keep its adapter live longest. |
| Connect / Accounts / Dashboard | read-only | Pull into Neon; we own the writes locally. |
| Sites & Funnels | **read-only — page write impossible** | CMS must become the system of record (publish to Cloudflare, not GHL). |
| Automations | **list-only** | Our own event engine replaces it; GHL list is informational. |
| Dialers (Numbers/Flagged) | not GHL — Dialpad/Hiya/NV | External APIs / Chrome extension; biggest unknown (see specs). |
| TxtGen / Results | no GHL API | Greenfield / shared UI. |

## Index

| # | Module | Target home | Track / order | GHL ceiling | Spec |
|---|---|---|---|---|---|
| 1 | Connect | Own core (encrypted creds) | A · 0 (foundation) | read | [01-connect.md](01-connect.md) |
| 2 | Accounts | Own core (`organizations`) + SF mirror | A · 3 | read | [02-accounts.md](02-accounts.md) |
| 3 | Custom Values | Own core (config KV) + GHL write-through | A · 6 (kept longest) | **read+write** | [03-custom-values.md](03-custom-values.md) |
| 4 | Sites & Funnels | **CMS + Cloudflare** | E2 · 4 | read-only | [04-sites-funnels.md](04-sites-funnels.md) |
| 5 | Automations | Own core + NATS rules | · 7 | list-only | [05-automations.md](05-automations.md) |
| 6 | Dashboard | Own core (Neon) + Grafana | · 5 | read-only | [06-dashboard.md](06-dashboard.md) |
| 7 | Dialers — Numbers | Dialer subsystem (Rust via NATS) | E1 · 2 | external | [07-dialers-numbers.md](07-dialers-numbers.md) |
| 8 | Dialers — Flagged | Dialer subsystem (Rust via NATS) | E1 · 2 | external | [08-dialers-flagged.md](08-dialers-flagged.md) |
| 9 | TxtGen | CMS (`cms/textgen`) | E2 · 4 | none | [09-txtgen.md](09-txtgen.md) |
| 10 | Results | Shared component + job status | cross-cutting | n/a | [10-results.md](10-results.md) |

## Spec template (each file follows this)

1. **Header** — status, target home, track/order, current state in repo.
2. **Purpose** — the job this module does for the user.
3. **Core features** — the must-haves (numbered, derived from the reference docs).
4. **Data model** — entities/fields it owns or touches (prototype shapes → new Neon tables).
5. **GHL API surface & ceiling** — endpoints + read/write reality.
6. **Target behavior in the new platform** — how it works post-migration.
7. **Migration notes & kill-switch trigger** — how it leaves GHL; what proves the replacement.
8. **Open questions** — what the owner must decide before building.

> **How to use:** when starting a track, open the relevant spec, resolve its Open Questions with the
> owner, then implement. Update the spec's status from `draft` → `agreed` once questions are closed.
