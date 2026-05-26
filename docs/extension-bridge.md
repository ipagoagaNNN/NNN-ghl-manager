# Chrome Extension Bridge Protocol

The GHL Manager app communicates with a companion Chrome extension (`cnam-extension`) via two parallel channels — `window.postMessage` and `CustomEvent`. Both are sent/received for every operation to ensure compatibility.

**Extension identity**: messages from extension have `source: 'cnam-extension'`. Messages from app have `source: 'ghl-manager'`.

---

## Inbound (Extension → App)

### `window.addEventListener('message', ...)`

Filter: `ev.source !== window` (same-page messages only) + `ev.data.source === 'cnam-extension'`

| `data.type` | Payload fields | Effect |
|-------------|----------------|--------|
| `ghl-extension-ready` | — | Sets `ghlExtensionBridgeReady = true` |
| `ghl-register-number-result` | `ok: boolean`, `mode: 'nv'|'hiya'|'both'`, `items: [{number}]`, `error?: string` | Updates `numbersRegistrationState[key].nv/hiya`, re-renders panel, shows toast |
| `ghl-form-save-auth-result` | `ok: boolean`, `locId: string`, `formId?: string`, `auth: object` | Calls `importBrowserSaveAuthPayload(data.auth, locId, formId)` |

### CustomEvent listeners (`window.addEventListener`)

| Event name | `ev.detail` shape | Effect |
|------------|-------------------|--------|
| `ghl-flagged-import` | `{ fileDataUrl: string, filename?: string, importedAt?: string }` | Parses CSV from data URL, imports into `flaggedData`, navigates to flagged panel |
| `ghl-numbers-sync` | `{ items: NumberItem[], totalCount, matchedCount, unmatchedCount, officeCount, syncedAt, source }` | Calls `importNumbersMatchFromExtension`, updates `numbersMatchData`, re-renders |
| `ghl-library-patch` | `Record<normalizedNumber, NumberLibraryEntry>` | Calls `applyNumbersLibraryPatch`, merges into library store |
| `ghl-numbers-sync-error` | `{ message: string }` | Updates status UI, shows toast |
| `ghl-register-number-result` | Same as postMessage variant above | Duplicate handler (CustomEvent path) |

---

## Outbound (App → Extension)

All outbound messages sent via **both** channels simultaneously:

```javascript
// Pattern used for every outbound dispatch:
window.postMessage({ source: 'ghl-manager', type: '...', ...payload }, '*');
window.dispatchEvent(new CustomEvent('...', { detail: payload }));
```

### Request numbers sync

```javascript
// postMessage
{ source: 'ghl-manager', type: 'ghl-request-numbers-sync', forceSync: boolean }
// CustomEvent: 'ghl-request-numbers-sync' with detail { forceSync: boolean }
```

### Register a number (NV/Hiya/Both)

```javascript
// postMessage
{
  source: 'ghl-manager',
  type: 'ghl-register-number',
  mode: 'nv' | 'hiya' | 'both',
  items: [{ number: string, hiyaNumber: string, departmentName: string }]
}
// CustomEvent: 'ghl-register-number' with same detail shape
```

### Request form save auth capture

```javascript
// postMessage
{
  source: 'ghl-manager',
  type: 'ghl-request-form-save-auth',
  locId: string,
  formId: string,
  // ...additional context
}
// CustomEvent: 'ghl-request-form-save-auth' with same detail shape
```

---

## localStorage Inbox Pattern

As a fallback when the extension fires before the app is ready, messages are also written to localStorage and consumed at app startup:

| Key | Purpose | Consumed by |
|-----|---------|-------------|
| `ghl_flagged_numbers_inbox_v1` | Extension-pushed flagged CSV | `consumePendingFlaggedInbox()` |
| `ghl_numbers_match_inbox_v1` | Extension-pushed numbers snapshot | `consumePendingNumbersInbox()` |
| `ghl_form_auth_inbox_v1` | Browser-captured form auth tokens | read at panel init |

Both inbox keys are **deleted** after consumption.

---

## Extension Readiness

```javascript
let ghlExtensionBridgeReady = false;
// Set to true when extension fires 'ghl-extension-ready' postMessage
// Check this before sending requests that require the extension
```

UI shows "Sync from Extension" button regardless — if extension isn't ready the request times out gracefully.

---

## New App Architecture Note

In the refactored app, the extension bridge stays **identical in protocol** — same event names, same payload shapes. The only change: the bridge is wrapped in a Svelte store (`ExtensionBridgeStore`) so components subscribe reactively instead of scattering `addEventListener` calls.

```typescript
// frontend/src/lib/stores/extension-bridge.svelte.ts
// Provides:
//   extensionReady: $state<boolean>
//   lastNumbersSync: $state<NumbersMatchData | null>
//   lastFlaggedImport: $state<FlaggedData | null>
//   sendRegisterNumber(mode, items): void
//   requestNumbersSync(forceSync): void
```
