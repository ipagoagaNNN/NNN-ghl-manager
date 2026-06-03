# Module 5 — Automations / Workflows (core-feature spec)

- **Status:** draft (gate for own-core automation engine · order 7)
- **Target home:** Own core + NATS event rules (GHL workflow list becomes informational, then dropped)
- **Current state in repo:** `handlers/automations.go` (`GET /api/workflows/:loc`), `routes/automations/+page.svelte`. Read-only list shipped (s3).

## Purpose
Today: view GHL workflows (read-only) and optionally manage contact enrollment. In the new platform: **our own automation/event engine** ("lead.created → notify/tag/enroll") driven by the NATS event stream.

## Core features (today, GHL)
1. **List workflows per account** — `GET /workflows/?locationId=` → name, status, version, createdAt, updatedAt.
2. **Workflow table** — sortable list with those columns.
3. _(Optional, needs live verify)_ **Enrollment count** per workflow — `GET /workflows/{id}/contacts?locationId=`.
4. _(Optional)_ **Manual enroll / un-enroll a contact** — `POST` / `DELETE /contacts/{cid}/workflow/{wid}` (contact API — supported).

## HARD LIMIT
The Workflows API is **list-only.** No endpoint to get a workflow's steps/detail, update, publish/unpublish, toggle, or trigger. Editing workflows requires the browser-session internal API (rejected). So "open a workflow to edit its steps" is **not possible** on the public API.

## Target behavior in the new platform
- A **native automation engine** consumes `events.>` (NATS): triggers (lead.created, message.inbound, stage.changed), conditions, actions (notify, tag, enroll-in-GHL-workflow via the contact API, fan-out to Zapier/WhatsApp).
- GHL's workflow **list** stays as a read-through reference ("what automations exist upstream") and is dropped once our engine carries the load.
- Contact↔workflow enrollment (the one supported write) can remain as an action our engine invokes.

## Data model
- `automations` (Neon) — our own rule definitions (trigger, conditions, actions). 
- GHL workflow list = read-through cache (no local source-of-truth needed).
- Enrollment actions recorded as `activities`.

## GHL API surface & ceiling
- `GET /workflows/?locationId=` (list-only). `GET /workflows/{id}/contacts` (enrollment count — verify with PIT). `POST`/`DELETE /contacts/{cid}/workflow/{wid}` (enrollment writes). Version `2021-07-28`.
- **Ceiling:** workflows read/list-only; the only writes are contact-enrollment via the contact API.

## Migration notes & kill-switch trigger
- **Trigger:** lead/contact automations run off our `events.>` consumers (notifier/fanout); GHL workflow list is informational only → drop the GHL read when the native engine covers the needed triggers.

## Open questions
- **Engine scope v1:** which triggers/conditions/actions are must-have? (Defines the whole subsystem.)
- **Rebuild vs reference:** do we replicate specific GHL workflows in our engine, or only build new ones?
- **Enrollment column:** worth the live `/workflows/{id}/contacts` verification, or skip?
