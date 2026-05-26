# Feature Inventory & Migration Map

Maps each module's current implementation to its target in the refactored architecture.

---

## Module 1: Connect

**Panel**: `#panel-connect`
**Migration priority**: 1 (blocking — must work before anything else)

| Feature | Current JS | New location |
|---------|-----------|--------------|
| Agency token input | `document.getElementById('agencyToken')` | `routes/connect/+page.svelte` |
| Company ID input | `document.getElementById('companyId')` | `routes/connect/+page.svelte` |
| Validate + load sub-accounts | `conectar()` → `GET /locations/search` | `POST /api/connect` → Go handler |
| Store agency token | `localStorage.setItem('ghl_agency_token', ...)` | Server-side vault, session cookie |
| Status dot + sub-account count | DOM manipulation | `stores/session.svelte.ts` reactive state |
| Missing token warning | DOM text | Svelte conditional block |

---

## Module 2: Accounts

**Panel**: `#panel-accounts`
**Migration priority**: 2 (low risk, mostly table + form)

| Feature | Current JS | New location |
|---------|-----------|--------------|
| Sub-account list render | `renderLocs(allLocs)` | `components/accounts/AccountList.svelte` |
| Individual token input per account | DOM inputs + `saveToken(id, token)` | `POST /api/tokens/:locationId` → vault |
| Domain / Acuity / Calendar ID fields | `saveDomain`, `saveAcuityField`, `saveCalendarIds` | Go handler, persisted server-side |
| Library editor (6-column grid) | `renderAccountsLibraryEditor()` | `components/accounts/LibraryEditor.svelte` |
| CSV export of library | `downloadAccountsLibraryCsv()` | Client-side blob (stays in frontend) |
| CSV import of library | `FileReader` + `parseCSV` | Frontend CSV parse → `POST /api/accounts/library` |
| Select / deselect accounts | `selected` Set | `stores/accounts.svelte.ts` — `$state<Set<string>>` |

---

## Module 3: Custom Values

**Panel**: `#panel-cv`
**Migration priority**: 3 (complex but self-contained)

| Feature | Current JS | New location |
|---------|-----------|--------------|
| Load CVs from selected accounts | `cargarCVs()` → sequential fetches | `GET /api/cv` → Go fan-out (parallel) |
| Display CVs grouped by key | `renderCVFolders()` | `components/custom-values/CVEditor.svelte` |
| Edit a single CV value | input `onchange` + `pendingValues` map | component local state |
| Bulk apply changes | `aplicar()` → sequential PUT | `POST /api/cv/bulk` → Go parallel PUT |
| Brand presets (NNN/FTB/AdvBeauty/General) | hardcoded bulk-fill | `components/custom-values/BulkFill.svelte` |
| Inner tabs (Values / Forms / Trigger Links) | `currentCvInnerTab` + DOM show/hide | Svelte tab component |
| New Patient Link scanner | `renderNewPatientLinkPanel()` | `components/custom-values/NPLScanner.svelte` |

---

## Module 4: Sites & Funnels

**Panel**: `#panel-sf`
**Migration priority**: 5 (riskiest — pixel injection touches live GHL pages)

| Feature | Current JS | New location |
|---------|-----------|--------------|
| List funnels per account | `GET /funnels/funnel/list` | `GET /api/funnels/:locationId` |
| Per-account tabs | `sfTabs` DOM | Svelte tab component |
| Funnel page scanner | bookmarklet + page scan | `components/funnels/PixelScanner.svelte` |
| Pixel injection (headCode/bodyCode) | `PUT /funnels/funnel/page/{id}` | `PUT /api/funnels/page/:pageId` |
| Meta Pixel detection | string search in headCode | utility function in `lib/utils/pixel.ts` |
| UTM script injection | string manipulation | utility function |
| Pixel status bar | DOM | Svelte reactive component |

---

## Module 5: Automations

**Panel**: `#panel-auto`
**Migration priority**: 2 (read-only, safe)

| Feature | Current JS | New location |
|---------|-----------|--------------|
| List workflows per account | `GET /workflows/?locationId=` | `GET /api/workflows/:locationId` |
| Workflow table with enrollment stats | `renderAutomations()` | `components/automations/WorkflowTable.svelte` |

---

## Module 6: Dashboard

**Panel**: `#panel-dash`
**Migration priority**: 4 (read-only analytics, multiple API calls)

| Feature | Current JS | New location |
|---------|-----------|--------------|
| Leads by day chart | `loadDashboard()` | `components/dashboard/LeadsChart.svelte` |
| Contacts paginated fetch | `loadContactsSnapshot()` | `GET /api/dashboard/:locationId/contacts` |
| Campaign stats | contacts + custom fields join | Go aggregation, cached 60s |
| Top campaigns table | DOM table build | `components/dashboard/CampaignTable.svelte` |
| Date range filter | DOM inputs | Svelte form + store |

---

## Module 7: Dialers — Numbers

**Panel**: `#panel-numbers`
**Migration priority**: 6 (extension bridge + Rust worker)

| Feature | Current JS | New location |
|---------|-----------|--------------|
| Sync from extension | `requestNumbersSyncFromExtension()` | `stores/extension-bridge.svelte.ts` |
| Display NV/Hiya match by office/dept | `renderNumbersPanel()` | `components/dialers/NumberTable.svelte` |
| Filter by office / department / search | JS filter + `numbersOfficeFilter` | Svelte reactive filters |
| Register number (NV/Hiya/Both) | `postMessage ghl-register-number` | extension bridge store method |
| Numbers library (persist match state) | `numbersRegistrationState` + `NumbersLibrary` localStorage | `stores/dialers.svelte.ts` + `POST /api/numbers/library` |
| Matching logic (set operations) | JS nested loops | **Rust worker**: `workers/number-matcher` |
| Deleted numbers from library | `buildDeletedNumbersFromLibrary()` | Rust worker output |

---

## Module 8: Dialers — Flagged Numbers

**Panel**: `#panel-flagged`
**Migration priority**: 6 (alongside Numbers)

| Feature | Current JS | New location |
|---------|-----------|--------------|
| CSV import (file picker) | `FileReader` + `parseCSV` | Frontend CSV parse |
| CSV import from extension | `ghl-flagged-import` event | extension bridge store |
| Preview table with search | DOM table + filter | `components/dialers/FlaggedTable.svelte` |
| Department chart | canvas/DOM chart | Svelte chart component |
| Spam label tracking | display only | data from `FlaggedData.rows` |

---

## Module 9: TxtGen

**Panel**: `#panel-txtgen`
**Migration priority**: 7 (self-contained, port last)

| Feature | Current JS | New location |
|---------|-----------|--------------|
| Spa picker | embedded UI | `routes/txtgen/+page.svelte` |
| Context form | embedded UI | same |
| Script copy | clipboard API | same |

Content is fully self-contained — no GHL API calls.

---

## Module 10: Results

**Panel**: `#panel-results`
**Migration priority**: Shared — displayed by any bulk operation

| Feature | Current JS | New location |
|---------|-----------|--------------|
| Progress log (OK/Error counts) | `logBox`, `sOk`, `sErr` DOM elements | `components/shared/ResultsLog.svelte` |
| Progress bar | `progFill` | Svelte progress component |
| Back navigation | `nav('cv')` | Svelte routing |

---

## Shared Utilities to Extract

| Utility | Current | New |
|---------|---------|-----|
| `parseCSV(text)` | inline function | `frontend/src/lib/utils/csv.ts` |
| `compactPhone(number)` | inline function | `frontend/src/lib/utils/phone.ts` |
| `esc(str)` / `escAttr(str)` | inline XSS escaping | not needed in Svelte (auto-escaped) |
| `showToast(msg)` | DOM append | `components/shared/Toast.svelte` |
| `sleep(ms)` | `new Promise(resolve => setTimeout(resolve, ms))` | only needed in Go proxy (rate limiting) |
| `getDomainForLoc(loc)` | heuristic domain extraction | `frontend/src/lib/utils/domain.ts` |
