# Module 9 — TxtGen (core-feature spec)

- **Status:** draft (gate for Track E2 — folds into CMS authoring)
- **Target home:** CMS — `backend/internal/cms/textgen`
- **Current state in repo:** not yet ported (prototype `#panel-txtgen`). Self-contained; **no GHL API**.

## Purpose
Generate copy / scripts from a small form (spa picker + context) → produce a ready-to-use script the operator copies. Entirely self-contained in the prototype.

## Core features
1. **Spa picker** — choose the brand/location context.
2. **Context form** — inputs that parameterize the output.
3. **Script generation + copy** — produce text; clipboard copy.

## Data model
- Templates / presets (Neon, CMS-owned) — the spa list + the generation templates.
- No external data; no GHL.

## Target behavior in the new platform
- Becomes a **CMS authoring panel** (`cms/textgen`): generated copy fills CMS page **block text** (Module 4), so TxtGen is no longer a dead-end clipboard tool but feeds the publishing pipeline.
- Generation can be **template-based** (deterministic) or call an **LLM adapter** (treated as an integration) — owner's choice.

## GHL API surface & ceiling
- None. Greenfield; no GHL dependency ever existed.

## Migration notes & kill-switch trigger
- **No kill switch needed** — there's no GHL coupling to retire. It's net-new functionality inside the CMS.

## Open questions
- **Template vs LLM:** is TxtGen rule/template-based, or should it call an LLM (which model/provider, and does that secret go in the credential store)?
- **What are the "scripts"?** Ad copy, call scripts, page copy, SMS templates? (Defines inputs/outputs and where the output is used.)
- **Reuse:** should generated copy be saved as reusable snippets per brand?
